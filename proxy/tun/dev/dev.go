package dev

import "github.com/google/netstack/tcpip/stack"

type TunDevice interface {
	URL() string
	AsLinkEndpoint() (stack.LinkEndpoint, error)
	Close()
}
