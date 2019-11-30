// +build linux android

package dev

import (
	"bytes"
	"errors"
	"fmt"
	"net/url"
	"os"
	"strconv"
	"sync"
	"syscall"
	"unsafe"

	"github.com/google/netstack/tcpip/link/channel"
	"github.com/google/netstack/tcpip/link/fdbased"
	"github.com/google/netstack/tcpip/stack"
	"golang.org/x/sys/unix"
)

const (
	cloneDevicePath = "/dev/net/tun"
	ifReqSize       = unix.IFNAMSIZ + 64
)

type tunLinux struct {
	url         string
	name        string
	tunFile     *os.File
	linkCache   *channel.Endpoint
	mtuOverride uint32

	closed   bool
	stopW    chan struct{}
	stopOnce sync.Once
	wg       sync.WaitGroup // wait for goroutines to stop
}

// OpenTunDevice return a TunDevice according a URL
func OpenTunDevice(deviceURL url.URL) (TunDevice, error) {
	mtuOverride, _ := strconv.ParseInt(deviceURL.Query().Get("mtu"), 0, 32)

	t := &tunLinux{
		mtuOverride: uint32(mtuOverride),
		url:         deviceURL.String(),
		stopW:       make(chan struct{}),
	}
	switch deviceURL.Scheme {
	case "dev":
		return t.openDeviceByName(deviceURL.Host)
	case "fd":
		fd, err := strconv.ParseInt(deviceURL.Host, 10, 32)
		if err != nil {
			return nil, err
		}
		return t.openDeviceByFd(int(fd))
	}
	return nil, fmt.Errorf("Unsupported device type `%s`", deviceURL.Scheme)
}

func (t *tunLinux) Name() string {
	return t.name
}

func (t *tunLinux) URL() string {
	return t.url
}

func (t *tunLinux) AsLinkEndpoint() (result stack.LinkEndpoint, err error) {
	if t.linkCache != nil {
		return t.linkCache, nil
	}

	mtu, err := t.getInterfaceMtu()

	if err != nil {
		return nil, errors.New("Unable to get device mtu")
	}

	result, err = fdbased.New(&fdbased.Options{
		FDs:            []int{int(t.tunFile.Fd())},
		MTU:            mtu,
		EthernetHeader: false,
	})

	//t.linkCache = &result

	return result, nil
}

func (t *tunLinux) Write(buff []byte) (int, error) {
	return t.tunFile.Write(buff)
}

func (t *tunLinux) Read(buff []byte) (int, error) {
	return t.tunFile.Read(buff)
}

func (t *tunLinux) Close() {
	t.stopOnce.Do(func() {
		t.closed = true
		close(t.stopW)
		t.tunFile.Close()
	})
}

func (t *tunLinux) openDeviceByName(name string) (TunDevice, error) {
	nfd, err := unix.Open(cloneDevicePath, os.O_RDWR, 0)
	if err != nil {
		return nil, err
	}

	var ifr [ifReqSize]byte
	var flags uint16 = unix.IFF_TUN | unix.IFF_NO_PI
	nameBytes := []byte(name)
	if len(nameBytes) >= unix.IFNAMSIZ {
		return nil, errors.New("interface name too long")
	}
	copy(ifr[:], nameBytes)
	*(*uint16)(unsafe.Pointer(&ifr[unix.IFNAMSIZ])) = flags

	_, _, errno := unix.Syscall(
		unix.SYS_IOCTL,
		uintptr(nfd),
		uintptr(unix.TUNSETIFF),
		uintptr(unsafe.Pointer(&ifr[0])),
	)
	if errno != 0 {
		return nil, errno
	}
	err = unix.SetNonblock(nfd, true)

	// Note that the above -- open,ioctl,nonblock -- must happen prior to handing it to netpoll as below this line.

	t.tunFile = os.NewFile(uintptr(nfd), cloneDevicePath)
	t.name, err = t.getName()
	if err != nil {
		t.tunFile.Close()
		return nil, err
	}

	return t, nil
}

func (t *tunLinux) openDeviceByFd(fd int) (TunDevice, error) {
	var ifr struct {
		name  [16]byte
		flags uint16
		_     [22]byte
	}

	fd, err := syscall.Dup(fd)
	if err != nil {
		return nil, err
	}

	_, _, errno := syscall.Syscall(syscall.SYS_IOCTL, uintptr(fd), syscall.TUNGETIFF, uintptr(unsafe.Pointer(&ifr)))
	if errno != 0 {
		return nil, errno
	}

	if ifr.flags&syscall.IFF_TUN == 0 || ifr.flags&syscall.IFF_NO_PI == 0 {
		return nil, errors.New("Only tun device and no pi mode supported")
	}

	nullStr := ifr.name[:]
	i := bytes.IndexByte(nullStr, 0)
	if i != -1 {
		nullStr = nullStr[:i]
	}
	t.name = string(nullStr)
	t.tunFile = os.NewFile(uintptr(fd), "/dev/tun")

	return t, nil
}

func (t *tunLinux) getInterfaceMtu() (uint32, error) {
	if t.mtuOverride > 0 {
		return t.mtuOverride, nil
	}

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

func (t *tunLinux) getName() (string, error) {
	sysconn, err := t.tunFile.SyscallConn()
	if err != nil {
		return "", err
	}
	var ifr [ifReqSize]byte
	var errno syscall.Errno
	err = sysconn.Control(func(fd uintptr) {
		_, _, errno = unix.Syscall(
			unix.SYS_IOCTL,
			fd,
			uintptr(unix.TUNGETIFF),
			uintptr(unsafe.Pointer(&ifr[0])),
		)
	})
	if err != nil {
		return "", errors.New("failed to get name of TUN device: " + err.Error())
	}
	if errno != 0 {
		return "", errors.New("failed to get name of TUN device: " + errno.Error())
	}
	nullStr := ifr[:]
	i := bytes.IndexByte(nullStr, 0)
	if i != -1 {
		nullStr = nullStr[:i]
	}
	t.name = string(nullStr)
	return t.name, nil
}
