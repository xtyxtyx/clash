// +build amd64 arm64 mips64

package tun

import (
	"fmt"
	"net"
	"time"

	"github.com/Dreamacro/clash/common/pool"
	"github.com/Dreamacro/clash/dns"
	"github.com/Dreamacro/clash/log"
	"github.com/google/netstack/tcpip"
	"github.com/google/netstack/tcpip/adapters/gonet"
	"github.com/google/netstack/tcpip/stack"
	D "github.com/miekg/dns"
)

const (
	defaultTimeout = 10
)

var (
	ipv4Zero = tcpip.Address(net.IPv4zero.To4())
)

// DNSServer is DNS Server listening on tun devcice
type DNSServer struct {
	handler       func(w D.ResponseWriter, r *D.Msg)
	dnsEndpointID stack.TransportEndpointID
}

type connResponseWriter struct {
	*gonet.Conn
}

func (s *DNSServer) HandleConn(id stack.TransportEndpointID, conn *gonet.Conn) bool {
	if id.LocalPort != s.dnsEndpointID.LocalPort {
		return false
	}

	if s.dnsEndpointID.LocalAddress != id.LocalAddress &&
		s.dnsEndpointID.LocalAddress != ipv4Zero {
		return false
	}

	go func() {
		buffer := pool.BufPool.Get().([]byte)
		defer pool.BufPool.Put(buffer[:cap(buffer)])
		defer conn.Close()
		w := &connResponseWriter{Conn: conn}
		var msg D.Msg
		for {
			conn.SetDeadline(time.Now().Add(defaultTimeout * time.Second))
			// TODO: handle request larger than MTU
			n, err := conn.Read(buffer[:])
			if err != nil {
				break
			}
			msg.Unpack(buffer[:n])
			go s.handler(w, &msg)
		}
	}()

	return true
}

func (w *connResponseWriter) WriteMsg(msg *D.Msg) error {
	b, err := msg.Pack()
	if err != nil {
		return err
	}
	_, err = w.Write(b)

	return err
}

func (w *connResponseWriter) TsigStatus() error {
	return nil
}
func (w *connResponseWriter) TsigTimersOnly(bool) {
	// Unsupported
}
func (w *connResponseWriter) Hijack() {
	// Unsupported
}

// CreateDNSServer create a dns server on given netstack
func CreateDNSServer(resolver *dns.Resolver, ip net.IP, port int) *DNSServer {
	handler := dns.NewHandler(resolver)

	if v4 := ip.To4(); v4 != nil {
		ip = v4
	}

	server := &DNSServer{
		handler: handler,
		dnsEndpointID: stack.TransportEndpointID{
			LocalAddress: tcpip.Address(ip),
			LocalPort:    uint16(port),
		},
	}

	return server
}

// DNSListen return the listening address of DNS Server
func (t *tunAdapter) DNSListen() string {
	if t.dnsserver != nil {
		id := t.dnsserver.dnsEndpointID
		return fmt.Sprintf("%s:%d", id.LocalAddress.String(), id.LocalPort)
	}
	return ""
}

func (t *tunAdapter) DestroyDNSSerrvice() {
	t.dnsserver = nil
}

// Stop stop the DNS Server on tun
func (t *tunAdapter) CreateDNSServer(resolver *dns.Resolver, addr string) error {
	if resolver == nil {
		return fmt.Errorf("Failed to create DNS server on tun: resolver not provided")
	}
	var err error
	_, port, err := net.SplitHostPort(addr)
	if port == "0" || port == "" || err != nil {
		return err
	}

	udpAddr, err := net.ResolveUDPAddr("udp", addr)
	if err != nil {
		return err
	}

	server := CreateDNSServer(resolver, udpAddr.IP, udpAddr.Port)

	t.dnsserver = server
	log.Infoln("Tun DNS server listening at: %s", addr)
	return nil
}
