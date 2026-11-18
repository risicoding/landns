package landns

import (
	"context"
	"fmt"
	"net"
	"strings"
	"sync"
	"time"

	"github.com/grandcat/zeroconf"
	"github.com/wlynxg/anet"
)

type Node struct {
	Name string `json:"name"`
	Port int    `json:"port"`
}

func NewNode(name string, port int) Node {
	return Node{
		Name: name,
		Port: port,
	}
}

type Instance struct {
	Hostname string   `json:"hostname"`
	Instance string   `json:"instance"`
	Ipv4     []net.IP `json:"ip"`
	Port     int      `json:"port"`
}

func (node *Node) Advertise(ctx context.Context, deviceAlias string) error {
	infs, err := anet.Interfaces()
	if err != nil {
		return err
	}

	var interfaces []net.Interface

	for _, inf := range infs {
		// Skip down, loopback, docker, or telephony interfaces
		if (inf.Flags&net.FlagUp) == 0 || inf.Name == "lo" || inf.Name == "docker0" || strings.Contains(inf.Name, "ccmni") {
			continue
		}

		addrs, err := inf.Addrs()
		if err != nil {
			continue
		}

		hasIpv4 := false
		for _, addr := range addrs {
			if ipNet, ok := addr.(*net.IPNet); ok {
				if ipv4 := ipNet.IP.To4(); ipv4 != nil {
					hasIpv4 = true
					break
				}
			}
		}

		if hasIpv4 {
			interfaces = append(interfaces, inf)
		}
	}

	// If no specific interfaces matched, pass nil so zeroconf defaults cleanly
	var targetIfs []net.Interface
	if len(interfaces) > 0 {
		targetIfs = interfaces
	}

	server, err := zeroconf.Register(
		deviceAlias,
		fmt.Sprintf("_%s._tcp", node.Name),
		"local.",
		node.Port,
		[]string{"version=1"},
		targetIfs,
	)
	if err != nil {
		return fmt.Errorf("register mDNS service: %w", err)
	}
	defer server.Shutdown()

	<-ctx.Done()
	return nil
}

func (node *Node) Discover() ([]Instance, error) {
	resolver, err := zeroconf.NewResolver(nil)
	if err != nil {
		return nil, fmt.Errorf("create mDNS resolver: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	entries := make(chan *zeroconf.ServiceEntry)
	if err := resolver.Browse(ctx, fmt.Sprintf("_%s._tcp", node.Name), "local.", entries); err != nil {
		return nil, fmt.Errorf("browse for services: %w", err)
	}

	found := make(map[string]struct{})
	var devices []Instance

	for {
		select {
		case <-ctx.Done():
			go func() {
				for range entries {
				}
			}()
			return devices, nil

		case entry, ok := <-entries:
			if !ok {
				return devices, nil
			}
			if entry == nil {
				continue
			}

			var ips []net.IP
			for _, ip := range entry.AddrIPv4 {
				ipv4 := ip.To4()
				if ipv4 != nil {
					ips = append(ips, ipv4)
				}
			}

			key := fmt.Sprintf("%s|%s|%d", entry.Instance, entry.HostName, entry.Port)
			if _, exists := found[key]; exists {
				continue
			}
			found[key] = struct{}{}

			devices = append(devices, Instance{
				Instance: entry.Instance,
				Hostname: entry.HostName,
				Port:     entry.Port,
				Ipv4:     ips, // Only contains reachable IPs
			})
		}
	}
}

type FilteredInstance struct {
	Hostname string `json:"hostname"`
	Instance string `json:"instance"`
	Ipv4     net.IP `json:"ip"`
	Port     int    `json:"port"`
}

func (node *Node) DiscoverWithFiltered() ([]FilteredInstance, error) {
	resp, err := node.Discover()
	if err != nil {
		return nil, err
	}

	instances := make(chan FilteredInstance)
	var insWg sync.WaitGroup

	for _, instance := range resp {
		insWg.Add(1)
		go func(ins Instance) {
			defer insWg.Done()

			var fastestIp net.IP
			var fastestLatency time.Duration
			found := false

			var mu sync.Mutex
			var ipWg sync.WaitGroup

			for _, ipaddr := range ins.Ipv4 {
				ipWg.Add(1)
				go func(ip net.IP) {
					defer ipWg.Done()
					latency, err := Ping(ip.String())
					if err != nil {
						return
					}

					mu.Lock()
					if !found || latency < fastestLatency {
						fastestIp = ip
						fastestLatency = latency
						found = true
					}
					mu.Unlock()
				}(ipaddr)
			}

			ipWg.Wait()
			if !found {
				return
			}

			instances <- FilteredInstance{
				Hostname: ins.Hostname,
				Instance: ins.Instance,
				Ipv4:     fastestIp,
				Port:     ins.Port,
			}
		}(instance)
	}

	go func() {
		insWg.Wait()
		close(instances)
	}()

	var filteredInstances []FilteredInstance
	for ins := range instances {
		filteredInstances = append(filteredInstances, ins)
	}

	return filteredInstances, nil
}

func init() {
	anet.SetAndroidVersion(14)
}

func getLocalNetworks() ([]*net.IPNet, error) {
	infs, err := anet.Interfaces()
	if err != nil {
		return nil, err
	}

	var nets []*net.IPNet

	for _, inf := range infs {
		if (inf.Flags&net.FlagUp) == 0 || inf.Name == "lo" || inf.Name == "docker0" || strings.Contains(inf.Name, "dummy") || strings.Contains(inf.Name, "ccmni") {
			continue
		}
		addrs, err := inf.Addrs()
		if err != nil {
			continue
		}
		for _, addr := range addrs {
			if ipNet, ok := addr.(*net.IPNet); ok {
				if ipNet.IP.To4() != nil {
					nets = append(nets, ipNet)
				}
			}
		}
	}

	return nets, nil
}
