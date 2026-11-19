package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/risicoding/landns"
	// Replace "mymodule" with the actual module name defined in your go.mod file
)

func main() {
	if len(os.Args) <= 1 {
		fmt.Printf("Device Alias required\n\nUsage : command [deviceAlias]\n")
		return
	}

	fmt.Println("=== Starting landns Android ARM64 Test Binary ===")

	serviceName := "termux-landns-bin"
	deviceAlias := os.Args[1]
	bindPort := 9099

	// 1. Use your actual package types and constructor
	node := landns.NewNode(serviceName, bindPort)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 2. Fire up the package's advertiser in the background
	go func() {
		if err := node.Advertise(ctx, deviceAlias); err != nil {
			log.Printf("Advertiser stopped: %v", err)
		}
	}()

	fmt.Printf("Broadcasting service _%s._tcp via landns package...\n", serviceName)
	fmt.Printf("Broadcasting device %s\n", deviceAlias)
	time.Sleep(2000 * time.Millisecond)

	ticker := time.NewTicker(1 * time.Second)

	for {
		select {
		case <-ctx.Done():
			fmt.Printf("Shutting down test\n")
		case <-ticker.C:
			{
				fmt.Println("Scanning for matching network devices via package...")
				instances, err := node.DiscoverWithFiltered()
				if err != nil {
					log.Fatalf("Discovery process failed: %v", err)
				}

				// 4. Output the results returned directly by your package logic
				fmt.Printf("Scan completed. Found %d matching instances:\n", len(instances))
				for _, inst := range instances {
					fmt.Printf(" -> Name: %s | Host: %s | Port: %d | Reachable IPs: %v | Latency: %v\n",
						inst.Instance, inst.Hostname, inst.Port, inst.Ipv4, inst.Latency)
				}
			}
		}
	}
}
