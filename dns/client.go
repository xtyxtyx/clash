package dns

import (
	"context"
	"net"
	"time"

	"github.com/Dreamacro/clash/network"
	D "github.com/miekg/dns"
)

type client struct {
	*D.Client
	Address string
}

func (c *client) Exchange(m *D.Msg) (msg *D.Msg, err error) {
	return c.ExchangeContext(context.Background(), m)
}

func (c *client) ExchangeContext(ctx context.Context, m *D.Msg) (msg *D.Msg, err error) {
	host, port, _ := net.SplitHostPort(c.Address)

	ip, err := BootstrapResolver.ResolveIP(host)
	if err != nil {
		return nil, err
	}

	host = ip.String()

	var timeout time.Duration
	if deadline, ok := ctx.Deadline(); !ok {
		timeout = 0
	} else {
		timeout = time.Until(deadline)
	}

	// Clone DefaultDialer and set timeout
	dialer := network.DefaultDialer
	c.Dialer = &dialer
	c.Dialer.Timeout = timeout

	msg, _, err = c.Client.Exchange(m, net.JoinHostPort(host, port))
	return
}
