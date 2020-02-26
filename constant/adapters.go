package constant

import (
	"context"
	"fmt"
	"net"
	"time"
)

// Adapter Type
const (
	Direct AdapterType = iota
	Fallback
	Reject
	Selector
	Shadowsocks
	ShadowsocksR
	Snell
	Socks5
	Http
	URLTest
	Vmess
	LoadBalance
)

type ServerAdapter interface {
	net.Conn
	Metadata() *Metadata
}

type Connection interface {
	Chains() Chain
	AppendToChains(adapter ProxyAdapter)
}

type Chain []string

func (c Chain) String() string {
	switch len(c) {
	case 0:
		return ""
	case 1:
		return c[0]
	default:
		return fmt.Sprintf("%s[%s]", c[len(c)-1], c[0])
	}
}

type Conn interface {
	net.Conn
	Connection
}

type PacketConn interface {
	net.PacketConn
	Connection
	WriteWithMetadata(p []byte, metadata *Metadata) (n int, err error)
}

type ProxyAdapter interface {
	Name() string
	Type() AdapterType
	DialContext(ctx context.Context, metadata *Metadata) (Conn, error)
	DialUDP(metadata *Metadata) (PacketConn, error)
	SupportUDP() bool
	MarshalJSON() ([]byte, error)
}

type DelayHistory struct {
	Time  time.Time `json:"time"`
	Delay uint16    `json:"delay"`
}

type Proxy interface {
	ProxyAdapter
	Alive() bool
	DelayHistory() []DelayHistory
	Dial(metadata *Metadata) (Conn, error)
	LastDelay() uint16
	URLTest(ctx context.Context, url string) (uint16, error)
}

// AdapterType is enum of adapter type
type AdapterType int

func (at AdapterType) String() string {
	switch at {
	case Direct:
		return "Direct"
	case Fallback:
		return "Fallback"
	case Reject:
		return "Reject"
	case Selector:
		return "Selector"
	case Shadowsocks:
		return "Shadowsocks"
	case ShadowsocksR:
		return "ShadowsocksR"
	case Snell:
		return "Snell"
	case Socks5:
		return "Socks5"
	case Http:
		return "Http"
	case URLTest:
		return "URLTest"
	case Vmess:
		return "Vmess"
	case LoadBalance:
		return "LoadBalance"
	default:
		return "Unknown"
	}
}

// UDPPacket contains the data of UDP packet, and offers control/info of UDP packet's source
type UDPPacket interface {
	// Data get the payload of UDP Packet
	Data() []byte

	// WriteBack writes the payload with source IP/Port equals addr
	// - variable source IP/Port is important to STUN
	// - if addr is not provided, WriteBack will wirte out UDP packet with SourceIP/Prot equals to origional Target,
	//   this is important when using Fake-IP.
	WriteBack(b []byte, addr net.Addr) (n int, err error)

	// Close closes the underlaying connection.
	Close() error

	// LocalAddr returns the source IP/Port of packet
	LocalAddr() net.Addr
}
