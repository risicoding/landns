package landns_test

import (
	"fmt"
	"testing"

	"github.com/risicoding/landns"
)

func TestPing(t *testing.T) {
	latency, err := landns.Ping("192.168.43.1")
	if err != nil {
		t.Fatalf("Ping err %s", err.Error())
	}

	fmt.Printf("Latency: %v", latency)
}
