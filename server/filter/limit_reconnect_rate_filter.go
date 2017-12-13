package filter

import (
	"fmt"
	"net"
	"time"

	"github.com/millken/tcpwder/config"
	"github.com/millken/tcpwder/core"
	"github.com/millken/tcpwder/utils"
)

type LimitReconnectRateFilter struct {
	reconnects int
	interval   time.Duration
	clients    map[string]int
	stop       chan bool
}

func (this *LimitReconnectRateFilter) Init(cfg config.Server) bool {
	if cfg.LimitReconnectRate != nil {
		this.reconnects = cfg.LimitReconnectRate.Reconnects
		this.interval = utils.ParseDurationOrDefault(cfg.LimitReconnectRate.Interval, time.Second*2)
		this.clients = make(map[string]int)

		ticker := time.NewTicker(this.interval)
		go func() {
			for {
				select {
				case <-ticker.C:
					this.clients = make(map[string]int)
				case <-this.stop:
					ticker.Stop()
					return
				}
			}
		}()
		return true
	}
	return false
}

func (this *LimitReconnectRateFilter) Connect(client net.Conn) error {
	host, _, _ := net.SplitHostPort(client.RemoteAddr().String())
	if _, ok := this.clients[host]; ok {
		if this.clients[host] > this.reconnects {
			return fmt.Errorf("limit reconnet rate %s, limit %d", host, this.reconnects)
		}
	}
	return nil
}

func (this *LimitReconnectRateFilter) Disconnect(client net.Conn) {
	host, _, _ := net.SplitHostPort(client.RemoteAddr().String())
	this.clients[host] += 1
}

func (this *LimitReconnectRateFilter) Read(client net.Conn, rwc core.ReadWriteCount) {
}

func (this *LimitReconnectRateFilter) Write(client net.Conn, rwc core.ReadWriteCount) {
}

func (this *LimitReconnectRateFilter) Request(buf []byte) error {
	return nil
}

func (this *LimitReconnectRateFilter) Stop() {
	close(this.stop)
}

func init() {
	RegisterFilter("limit_reconnects_rate", func() interface{} {
		return new(LimitReconnectRateFilter)
	})
}
