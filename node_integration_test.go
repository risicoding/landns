package landns

import (
	"context"
	"testing"
	"time"
)

func TestNode_AdvertiseAndDiscover_Integration(t *testing.T) {
	// 1. Check if the testing environment has active non-loopback network interfaces.
	// Your package filters out loopback "lo" and virtual bridges, so a physical
	// or active virtual network adapter must be active for this test to pass.
	localNets, err := getLocalNetworks()
	if err != nil {
		t.Fatalf("Failed to analyze local interfaces: %v", err)
	}
	if len(localNets) == 0 {
		t.Skip("Skipping integration test: No active non-loopback IPv4 network interfaces found.")
	}

	// 2. Define unique configurations for our integration cluster
	serviceName := "landns-integration-test"
	deviceAlias := "test-node-peer"
	bindPort := 8899

	node := NewNode(serviceName, bindPort)

	// Create a cancelable context to control the lifecycle of our background broadcaster
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	errChan := make(chan error, 1)

	// 3. Start advertising the service in a background goroutine
	go func() {
		if err := node.Advertise(ctx, deviceAlias); err != nil {
			errChan <- err
		}
	}()

	// Allow a brief propagation window for multicast routers/interfaces to bind the group
	time.Sleep(1500 * time.Millisecond)

	// 4. Trigger the discovery phase
	discoveredInstances, err := node.Discover()
	if err != nil {
		t.Fatalf("Discovery process failed critically: %v", err)
	}

	// 5. Assertions: Inspect if our background node was correctly captured and validated
	var targetFound bool
	for _, instance := range discoveredInstances {
		if instance.Instance == deviceAlias {
			targetFound = true

			if instance.Port != bindPort {
				t.Errorf("Port mismatch: expected %d, got %d", bindPort, instance.Port)
			}

			if len(instance.Ipv4) == 0 {
				t.Errorf("Expected at least one matching common subnet IP, found none")
			}
			break
		}
	}

	// Check if any errors occurred inside our background advertiser thread
	select {
	case advertiseErr := <-errChan:
		t.Fatalf("Background broadcaster crashed: %v", advertiseErr)
	default:
	}

	if !targetFound {
		t.Errorf("Integration failure: Discovered node list did not contain instance %q", deviceAlias)
	}
}
