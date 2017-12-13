package filter

import (
	"log"
	"net"

	"github.com/millken/tcpwder/config"
	"github.com/millken/tcpwder/core"
	"github.com/millken/tcpwder/firewall"
)

type FilterInterface interface {
	Init(cf config.Server) bool
	Connect(client net.Conn) error
	Read(client net.Conn, rwc core.ReadWriteCount)
	Write(client net.Conn, rwc core.ReadWriteCount)
	Request([]byte) error
	Disconnect(client net.Conn)
	Stop()
}

var filters = make(map[string]func() interface{})

type Filter struct {
	cfg     config.Server
	filters map[string]FilterInterface
	stop    chan bool
}

func RegisterFilter(name string, filter func() interface{}) {
	if filter == nil {
		return
	}

	if _, ok := filters[name]; ok {
		log.Fatalln("Register called twice for filter " + name)
	}

	filters[name] = filter
}

func New(cfg config.Server) *Filter {
	return &Filter{
		cfg:     cfg,
		filters: make(map[string]FilterInterface),
	}
}

func (this *Filter) Start() {
	log.Printf("[INFO] Starting filter")
	this.stop = make(chan bool)
	for name, filter := range filters {
		ff := filter().(FilterInterface)
		if ff.Init(this.cfg) {
			this.filters[name] = ff
		}
	}
	go func() {
		for {
			select {

			case <-this.stop:
				log.Printf("Stopping filter")
				return
			}
		}
	}()
}

func (this *Filter) Stop() {
	for _, filter := range this.filters {
		filter.Stop()
	}
	this.stop <- true
}

func (this *Filter) HandleClientConnect(client net.Conn) error {
	for _, filter := range this.filters {
		if err := filter.Connect(client); err != nil {
			host, _, _ := net.SplitHostPort(client.RemoteAddr().String())
			firewall.SetDeny(host, 3600)
			return err
		}
	}
	return nil
}

func (this *Filter) HandleClientDisconnect(client net.Conn) {
	for _, filter := range this.filters {
		filter.Disconnect(client)
	}
}

func (this *Filter) HandleClientWrite(client net.Conn, rwc core.ReadWriteCount) {
	for _, filter := range this.filters {
		filter.Write(client, rwc)
	}
}

func (this *Filter) HandleClientRead(client net.Conn, rwc core.ReadWriteCount) {
	for _, filter := range this.filters {
		filter.Read(client, rwc)
	}
}

func (this *Filter) HandleClientRequest(buf []byte) error {
	for _, filter := range this.filters {
		if err := filter.Request(buf); err != nil {
			return err
		}
	}
	return nil
}
