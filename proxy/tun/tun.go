package tun

import "github.com/google/netstack/tcpip/stack"

// TunAdapter hold the state of tun/tap interface
type TunAdapter interface {
	Close()
	DeviceURL() string
	Stack() *stack.Stack
}
