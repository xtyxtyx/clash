package adapters

import (
	"context"
	"encoding/json"
	"errors"
	"net"

	C "github.com/Dreamacro/clash/constant"
)

type Redirect struct {
	*Base
	ToHost    string
	ToAddress net.IP
	ToPort    string
	forward   C.Proxy
}

type RedirectOption struct {
	Name      string   `proxy:"name"`
	ToHost    string   `proxy:"to-host,omitempty"`
	ToAddress string   `proxy:"to-address,omitempty"`
	ToPort    string   `proxy:"to-port,omitempty"`
	Proxies   []string `proxy:"proxies"`
}

func (s *Redirect) DialContext(ctx context.Context, metadata *C.Metadata) (C.Conn, error) {
	s.redirectMetadata(metadata)

	c, err := s.forward.DialContext(ctx, metadata)
	if err == nil {
		c.AppendToChains(s)
	}
	return c, err
}

func (s *Redirect) DialUDP(metadata *C.Metadata) (C.PacketConn, net.Addr, error) {
	s.redirectMetadata(metadata)

	pc, addr, err := s.forward.DialUDP(metadata)
	if err == nil {
		pc.AppendToChains(s)
	}
	return pc, addr, err
}

func (s *Redirect) SupportUDP() bool {
	return s.forward.SupportUDP()
}

func (s *Redirect) MarshalJSON() ([]byte, error) {
	return json.Marshal(map[string]interface{}{
		"type":    s.Type().String(),
		"forward": s.forward,
	})
}

func (s *Redirect) redirectMetadata(metadata *C.Metadata) {
	if len(s.ToHost) != 0 {
		metadata.DstIP = nil
		metadata.Host = s.ToHost
	}

	if len(s.ToAddress) != 0 {
		metadata.DstIP = s.ToAddress
	}

	if len(s.ToPort) != 0 {
		metadata.DstPort = s.ToPort
	}
}

func NewRedirect(name string, option RedirectOption, proxies []C.Proxy) (*Redirect, error) {
	if len(proxies) != 1 {
		return nil, errors.New("Provide only one proxy")
	}

	s := &Redirect{
		Base: &Base{
			name: name,
			tp:   C.Redirect,
		},
		forward:   proxies[0],
		ToHost:    option.ToHost,
		ToAddress: net.ParseIP(option.ToAddress),
		ToPort:    option.ToPort,
	}
	return s, nil
}
