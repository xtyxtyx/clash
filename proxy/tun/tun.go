package tun

import "github.com/Dreamacro/clash/dns"

// TunAdapter hold the state of tun/tap interface
type TunAdapter interface {
	Close()
	DeviceURL() string
	// Create creates dns server on tun device
	CreateDNSServer(resolver *dns.Resolver, addr string) error
	DestroyDNSSerrvice()
	DNSListen() string
}
