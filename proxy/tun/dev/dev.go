package dev

import "github.com/google/netstack/tcpip/stack"

type TunDevice interface {
	Name() string
	AsLinkEndpoint() (stack.LinkEndpoint, error)
	Close()
}
