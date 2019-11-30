package tun

import "github.com/google/netstack/tcpip/stack"
import "github.com/Dreamacro/clash/dns"

// TunAdapter hold the state of tun/tap interface
type TunAdapter interface {
	Close()
	DeviceURL() string
	Stack() *stack.Stack
	// Create creates dns server on tun device
	CreateDNSServer(resolver *dns.Resolver, addr string) error
	DNSListen() string
}
