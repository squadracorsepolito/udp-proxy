package pkg

import (
	"context"
	"log/slog"
	"net"
	"net/netip"
)

type Proxy struct {
	l *slog.Logger

	listenPort uint16
	listenConn *net.UDPConn

	destAddr string
	destConn *net.UDPConn
}

func NewProxy(cfg *ProxyConfig) *Proxy {
	return &Proxy{
		l: slog.Default(),

		listenPort: cfg.ListenerPort,

		destAddr: cfg.DestinationAddr,
	}
}

func (p *Proxy) Init() error {
	listenIPAddr := netip.MustParseAddr("0.0.0.0")
	listenAddr := net.UDPAddrFromAddrPort(netip.AddrPortFrom(listenIPAddr, p.listenPort))

	listenConn, err := net.ListenUDP("udp", listenAddr)
	if err != nil {
		return err
	}
	p.listenConn = listenConn

	destAddr, err := net.ResolveUDPAddr("udp", p.destAddr)
	if err != nil {
		return err
	}

	destConn, err := net.DialUDP("udp", nil, destAddr)
	if err != nil {
		return err
	}
	p.destConn = destConn

	return nil
}

func (p *Proxy) Run(ctx context.Context) {
	go func() {
		<-ctx.Done()
		p.listenConn.Close()
		p.destConn.Close()
	}()

	p.l.Info("proxy running", "listen_port", p.listenPort, "dest_addr", p.destAddr)

	buf := make([]byte, 2048)
	for {
		n, err := p.listenConn.Read(buf)
		if err != nil {
			continue
		}

		_, err = p.destConn.Write(buf[:n])
		if err != nil {
			continue
		}
	}
}
