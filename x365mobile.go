// Package x365mobile provides gomobile-compatible bindings for the X365 protocol.
//
// It exposes a simple API: parse a node URI, start a SOCKS5 proxy, and stop it.
// The Android app uses this via VpnService: the Go SOCKS5 server listens on a local
// port, and the VpnService routes device traffic through a tun2socks layer that
// connects to that SOCKS5 port.
package x365mobile

import (
	"context"
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/365vpn/x365/core"
)

// LogCallback is a gomobile-compatible log callback type.
// Android sets this via SetLogCallback to receive Go-side logs.
type LogCallback interface {
	OnLog(msg string)
}

// Proxy is the gomobile-compatible proxy manager.
// It is safe to call Start/Stop/Status from any goroutine.
type Proxy struct {
	mu       sync.Mutex
	listener net.Listener
	cancel   context.CancelFunc
	cfg      *x365.X365Config
	running  bool
}

// SetLogCallback installs a log callback that receives Go-side SOCKS5/tunnel
// log messages. Pass nil to disable. The callback is invoked from goroutines,
// so the receiver must be thread-safe (e.g. forward to the main thread).
func SetLogCallback(cb LogCallback) {
	if cb == nil {
		x365.SetLogger(nil)
		return
	}
	x365.SetLogger(func(format string, args ...interface{}) {
		cb.OnLog(fmt.Sprintf(format, args...))
	})
}

// NewProxy creates a new Proxy instance.
func NewProxy() *Proxy {
	return &Proxy{}
}

// ParseURI parses an x365:// URI and returns the server address and path for display.
// Returns an error string if parsing fails (gomobile doesn't propagate Go errors well).
func (p *Proxy) ParseURI(uri string) (string, error) {
	cfg, err := x365.ParseURI(uri)
	if err != nil {
		return "", fmt.Errorf("parse: %w", err)
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	p.cfg = cfg
	return fmt.Sprintf("%s:%d%s", cfg.Server, cfg.Port, cfg.Path), nil
}

// Start begins the SOCKS5 proxy on the given listen address (e.g. "127.0.0.1:10808").
// The nodeURI must be a valid x365:// URI.
func (p *Proxy) Start(nodeURI, listenAddr string) error {
	cfg, err := x365.ParseURI(nodeURI)
	if err != nil {
		return err
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	if p.running {
		p.stopLocked()
	}

	ln, err := net.Listen("tcp", listenAddr)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithCancel(context.Background())
	p.listener = ln
	p.cancel = cancel
	p.cfg = cfg
	p.running = true

	go p.acceptLoop(ctx, ln, cfg)

	return nil
}

// Stop shuts down the SOCKS5 proxy.
func (p *Proxy) Stop() error {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.stopLocked()
}

func (p *Proxy) stopLocked() error {
	if !p.running {
		return nil
	}
	p.running = false
	if p.cancel != nil {
		p.cancel()
		p.cancel = nil
	}
	var err error
	if p.listener != nil {
		err = p.listener.Close()
		p.listener = nil
	}
	p.cfg = nil
	return err
}

// IsRunning returns true if the proxy is active.
func (p *Proxy) IsRunning() bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.running
}

// CurrentPath returns the current node's path (e.g. "/hk"), or empty if not running.
func (p *Proxy) CurrentPath() string {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.cfg == nil {
		return ""
	}
	return p.cfg.Path
}

// CurrentServer returns the current node's server address, or empty if not running.
func (p *Proxy) CurrentServer() string {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.cfg == nil {
		return ""
	}
	return p.cfg.Server
}

// TestConnect attempts a quick connection through the node to verify reachability.
// It dials example.com:80 through the X365 tunnel and returns the first line of the response.
// Timeout is 15 seconds.
func (p *Proxy) TestConnect(nodeURI string) (string, error) {
	cfg, err := x365.ParseURI(nodeURI)
	if err != nil {
		return "", err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	conn, err := x365.Dial(ctx, cfg, "example.com", 80)
	if err != nil {
		return "", err
	}
	defer conn.Close()

	// Send a minimal HTTP request
	_, err = conn.Write([]byte("GET / HTTP/1.0\r\nHost: example.com\r\n\r\n"))
	if err != nil {
		return "", err
	}

	// Read response (first 512 bytes)
	buf := make([]byte, 512)
	conn.SetReadDeadline(time.Now().Add(10 * time.Second))
	n, err := conn.Read(buf)
	if err != nil && n == 0 {
		return "", err
	}
	return string(buf[:n]), nil
}

func (p *Proxy) acceptLoop(ctx context.Context, ln net.Listener, cfg *x365.X365Config) {
	for {
		client, err := ln.Accept()
		if err != nil {
			select {
			case <-ctx.Done():
				return
			default:
				continue
			}
		}
		go x365.HandleSOCKS5(client, cfg)
	}
}
