package landns

import (
	"errors"
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

	pinger.Run()

	stats := pinger.Statistics()
	if stats.PacketsRecv == 0 {
		return 0, ErrNoResponse
	}

	return stats.AvgRtt, nil
}
