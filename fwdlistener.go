package fwdlistener

import (
	"net"
	"strconv"

	"github.com/NebulousLabs/go-upnp"
)

//FwdListener wraps a net.Listener adn forwards the underlying listening port and
//the Addr() method of the returned Listener returns the public IP instead of local IP
func FwdListener(l net.Listener) (net.Listener, error) {
	igd, externalIP, err := initalize()
	if err != nil {
		return nil, err
	}

	closeFn, externalAddr, err := setup(igd, externalIP, l.Addr())
	if err != nil {
		l.Close()
		return nil, err
	}

	return &fwdListener{externalAddr: externalAddr, Listener: l, closeFn: closeFn}, nil
}

//FwdPacketListener wraps a net.PacketConn, forwards the underlying listening port and
//the Addr() method of the returned PacketConn returns the public IP instead of local IP
func FwdPacketListener(l net.PacketConn) (net.PacketConn, error) {
	igd, externalIP, err := initalize()
	if err != nil {
		return nil, err
	}

	closeFn, externalAddr, err := setup(igd, externalIP, l.LocalAddr())
	if err != nil {
		l.Close()
		return nil, err
	}

	return &fwdPacketListener{externalAddr: externalAddr, PacketConn: l, closeFn: closeFn}, nil
}

//Listen works like net.Listen but forwards the port using UPnP and the Addr()
//method of the returned Listener returns the public IP instead of local IP
func Listen(network, address string) (net.Listener, error) {
	listener, err := net.Listen(network, address)
	if err != nil {
		return nil, err
	}

	return FwdListener(listener)
}

//ListenPacket works like net.ListenPacket but forwards the port using UPnP
//and the LocalAddr() method of the returned PacketConn returns the public IP
//instead of local IP
func ListenPacket(network, address string) (net.PacketConn, error) {
	listener, err := net.ListenPacket(network, address)
	if err != nil {
		return nil, err
	}

	return FwdPacketListener(listener)
}

func initalize() (*upnp.IGD, string, error) {
	igd, err := upnp.Discover()
	if err != nil {
		return nil, "", err
	}

	externalIP, err := igd.ExternalIP()
	if err != nil {
		return nil, "", err
	}
	return igd, externalIP, nil
}

func setup(igd *upnp.IGD, externalIP string, laddr net.Addr) (func(), net.Addr, error) {
	_, portString, _ := net.SplitHostPort(laddr.String())
	port, _ := strconv.ParseInt(portString, 10, 16)

	if err := igd.Forward(uint16(port), "go-fwdlistner: "+laddr.String()); err != nil {
		return nil, nil, err
	}
	return genCloseFunc(igd, uint16(port)), &addr{laddr.Network(), net.JoinHostPort(externalIP, portString)}, nil
}

type addr struct {
	network, address string
}

var _ (net.Addr) = (*addr)(nil)

func (a *addr) Network() string {
	return a.network
}

func (a *addr) String() string {
	return a.address
}

type fwdListener struct {
	externalAddr net.Addr
	closeFn      func()
	net.Listener
}

var _ net.Listener = (*fwdListener)(nil)

func (f *fwdListener) Addr() net.Addr {
	return f.externalAddr
}

func (f *fwdListener) Close() error {
	defer f.closeFn()
	return f.Close()
}

type fwdPacketListener struct {
	externalAddr net.Addr
	closeFn      func()
	net.PacketConn
}

var _ net.PacketConn = (*fwdPacketListener)(nil)

func (f *fwdPacketListener) LocalAddr() net.Addr {
	return f.externalAddr
}

func (f *fwdPacketListener) Close() error {
	defer f.closeFn()
	return f.Close()
}

func genCloseFunc(igd *upnp.IGD, port uint16) func() {
	return func() {
		igd.Clear(port)
	}
}
