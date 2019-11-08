// +build darwin

package dev

import (
	"errors"
	"fmt"
	"net/url"
	"strconv"
	"syscall"
	"unsafe"

	"golang.org/x/sys/unix"

	"github.com/google/netstack/tcpip/stack"
)

const utunControlName = "com.apple.net.utun_control"

// _CTLIOCGINFO value derived from /usr/include/sys/{kern_control,ioccom}.h
const _CTLIOCGINFO = (0x40000000 | 0x80000000) | ((100 & 0x1fff) << 16) | uint32(byte('N'))<<8 | 3

var sockaddrCtlSize uintptr = 32

type tun struct {
	url       string
	name      string
	fd        int
	linkCache *stack.LinkEndpoint
}

func OpenTunDevice(deviceURL url.URL) (TunDevice, error) {
	switch deviceURL.Scheme {
	case "dev":
		mtu, err := strconv.ParseInt(deviceURL.Query().Get("mtu"), 10, 32)
		if err != nil {
			mtu = -1
		}

		return tun{
			url: deviceURL.String(),
		}.openDeviceByName(deviceURL.Host, int(mtu))
		// case "fd":
		// 	fd, err := strconv.ParseInt(deviceURL.Host, 10, 32)
		// 	if err != nil {
		// 		return nil, err
		// 	}
		// 	return tun{
		// 		url: deviceURL.String(),
		// 	}.openDeviceByFd(int(fd))
	}

	return nil, errors.New("Unsupported device type " + deviceURL.Scheme)
}

func (t tun) Name() string {
	return t.name
}

func (t tun) URL() string {
	return t.url
}

func (t tun) AsLinkEndpoint() (result stack.LinkEndpoint, err error) {
	// TODO
	return nil, errors.New("Stub!")
}

func (t tun) Close() {
	syscall.Close(t.fd)
}

// from https://git.zx2c4.com/wireguard-go/tree/tun/tun_darwin.go
func (t tun) openDeviceByName(name string, mtu int) (TunDevice, error) {
	ifIndex := -1
	if name != "utun" {
		_, err := fmt.Sscanf(name, "utun%d", &ifIndex)
		if err != nil || ifIndex < 0 {
			return nil, fmt.Errorf("Interface name must be utun[0-9]*")
		}
	}

	fd, err := unix.Socket(unix.AF_SYSTEM, unix.SOCK_DGRAM, 2)

	if err != nil {
		return nil, err
	}

	var ctlInfo = &struct {
		ctlID   uint32
		ctlName [96]byte
	}{}

	copy(ctlInfo.ctlName[:], []byte(utunControlName))

	_, _, errno := unix.Syscall(
		unix.SYS_IOCTL,
		uintptr(fd),
		uintptr(_CTLIOCGINFO),
		uintptr(unsafe.Pointer(ctlInfo)),
	)

	if errno != 0 {
		return nil, fmt.Errorf("_CTLIOCGINFO: %v", errno)
	}

	// sockaddr_ctl specifeid in /usr/include/sys/kern_control.h
	var sockaddr struct {
		scLen      uint8
		scFamily   uint8
		ssSysaddr  uint16
		scID       uint32
		scUnit     uint32
		scReserved [5]uint32
	}

	sockaddr.scLen = uint8(sockaddrCtlSize)
	sockaddr.scFamily = unix.AF_SYSTEM
	sockaddr.ssSysaddr = 2
	sockaddr.scID = ctlInfo.ctlID
	sockaddr.scUnit = uint32(ifIndex) + 1

	scPointer := unsafe.Pointer(&sockaddr)

	_, _, errno = unix.RawSyscall(
		unix.SYS_CONNECT,
		uintptr(fd),
		uintptr(scPointer),
		uintptr(sockaddrCtlSize),
	)

	if errno != 0 {
		return nil, fmt.Errorf("SYS_CONNECT: %v", errno)
	}

	if mtu > 0 {
		if err := setInterfaceMtu(name, mtu); err != nil {
			return nil, err
		}
	}

	err = syscall.SetNonblock(fd, true)
	if err != nil {
		return nil, err
	}

	t.fd = fd
	t.name = name

	return t, nil
}

func (t tun) getInterfaceMtu() (uint32, error) {
	fd, err := syscall.Socket(syscall.AF_UNIX, syscall.SOCK_DGRAM, 0)
	if err != nil {
		return 0, err
	}

	defer syscall.Close(fd)

	var ifreq struct {
		name [16]byte
		mtu  int32
		_    [20]byte
	}

	copy(ifreq.name[:], t.name)
	_, _, errno := syscall.Syscall(syscall.SYS_IOCTL, uintptr(fd), syscall.SIOCGIFMTU, uintptr(unsafe.Pointer(&ifreq)))
	if errno != 0 {
		return 0, errno
	}

	return uint32(ifreq.mtu), nil
}

func setInterfaceMtu(name string, mtu int) error {
	fd, err := syscall.Socket(syscall.AF_UNIX, syscall.SOCK_DGRAM, 0)
	if err != nil {
		return err
	}

	defer syscall.Close(fd)

	var ifreq struct {
		name [16]byte
		mtu  int32
		_    [20]byte
	}

	copy(ifreq.name[:], name)
	ifreq.mtu = int32(mtu)

	_, _, errno := syscall.Syscall(syscall.SYS_IOCTL, uintptr(fd), syscall.SIOCSIFMTU, uintptr(unsafe.Pointer(&ifreq)))
	if errno != 0 {
		return errno
	}

	return nil
}
