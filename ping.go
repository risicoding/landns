package landns

import (
	"errors"
	"fmt"
	"time"

	"github.com/influxdata/go-ping"
)

var ErrNoResponse = errors.New("no ping response")

func Ping(ip string) (time.Duration, error) {
	pinger, err := ping.NewPinger(ip)
	if err != nil {
		return 0, err
	}

	pinger.Count = 4
	pinger.Timeout = 3 * time.Second

	fmt.Printf("Starting pinger run\n")
	pinger.Run()

	stats := pinger.Statistics()
	fmt.Printf("%+v\n", stats)
	if stats.PacketsRecv == 0 {
		fmt.Printf("0 Packets recieved\n")
		return 0, ErrNoResponse
	}

	return stats.AvgRtt, nil
}
