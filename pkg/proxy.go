package pkg

import (
	"context"
	"log/slog"
	"net"
	"net/netip"
	"sync/atomic"
	"time"
)

type Proxy struct {
	l *slog.Logger

	listenPort uint16
	listenConn *net.UDPConn

	destAddr string
	destConn *net.UDPConn

	statsInterval    time.Duration
	recvPackets      atomic.Int64
	deliveredPackets atomic.Int64
}

func NewProxy(cfg *ProxyConfig) *Proxy {
	return &Proxy{
		l: slog.Default(),

		listenPort: cfg.ListenerPort,

		destAddr: cfg.DestinationAddr,

		statsInterval: 5 * time.Second,
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

func (p *Proxy) runStats(ctx context.Context) {
	ticker := time.NewTicker(p.statsInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			p.l.Info("proxy stats",
				"listen_port", p.listenPort, "dest_addr", p.destAddr,
				"recv_packets", p.recvPackets.Load(), "delivered_packets", p.deliveredPackets.Load(),
			)
		}
	}
}

func (p *Proxy) Run(ctx context.Context) {
	go func() {
		<-ctx.Done()
		p.listenConn.Close()
		p.destConn.Close()
	}()

	go p.runStats(ctx)

	p.l.Info("proxy running", "listen_port", p.listenPort, "dest_addr", p.destAddr)

	buf := make([]byte, 2048)
	for {
		n, err := p.listenConn.Read(buf)
		if err != nil {
			p.l.Warn("failed to read from listener",
				"listen_port", p.listenPort, "dest_addr", p.destAddr,
				"error", err,
			)
			continue
		}

		p.recvPackets.Add(1)

		_, err = p.destConn.Write(buf[:n])
		if err != nil {
			p.l.Warn("failed to write to destination",
				"listen_port", p.listenPort, "dest_addr", p.destAddr,
				"error", err,
			)
			continue
		}

		p.deliveredPackets.Add(1)
	}
}
