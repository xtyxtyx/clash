package tun

import (
	"github.com/Dreamacro/clash/log"
	"github.com/Dreamacro/clash/tunnel"
	"github.com/songgao/water"
)

var (
	tun = tunnel.Instance()
)

type TunAdapter struct {
	Interface *water.Interface
}

func NewTunProxy(linuxIfName string) (*TunAdapter, error) {

	config := water.Config{
		DeviceType: water.TUN,
	}
	if linuxIfName != "" {
		config.PlatformSpecificParams = water.PlatformSpecificParams{
			Name: linuxIfName,
		}
	}

	ifce, err := water.New(config)

	if err != nil {
		return nil, err
	}

	tl := &TunAdapter{
		Interface: ifce,
	}
	log.Infoln("Tun adapter have interface name: %s", tl.IfName())

	return tl, nil

}

func (t *TunAdapter) Close() {
	t.Close()
	t.Interface = nil
}

func (t *TunAdapter) IfName() string {
	if t.Interface != nil {
		return t.Interface.Name()
	}
	return ""
}
