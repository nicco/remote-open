package client

import (
	"encoding/base64"
	"fmt"
	"io"
	"log"
	"net"
	"sync"
	"time"
)

type TunnelManager struct {
	tunnels  map[int]*tunnel
	mu       sync.Mutex
	sendData func(port int, data []byte)
	sendPing func(port int)
	conns    map[int]map[net.Conn]bool
}

type tunnel struct {
	port     int
	listener net.Listener
	misses   int
	stopCh   chan struct{}
}

const pingInterval = 5 * time.Second
const maxMisses = 3

func NewTunnelManager(sendData func(port int, data []byte), sendPing func(port int)) *TunnelManager {
	return &TunnelManager{
		tunnels:  make(map[int]*tunnel),
		conns:    make(map[int]map[net.Conn]bool),
		sendData: sendData,
		sendPing: sendPing,
	}
}

func (tm *TunnelManager) StartTunnel(port int) {
	tm.mu.Lock()
	defer tm.mu.Unlock()
	if _, exists := tm.tunnels[port]; exists {
		return
	}
	addr := fmt.Sprintf("127.0.0.1:%d", port)
	l, err := net.Listen("tcp", addr)
	if err != nil {
		log.Printf("cannot listen on %s: %v", addr, err)
		return
	}
	t := &tunnel{port: port, listener: l, stopCh: make(chan struct{})}
	tm.tunnels[port] = t
	tm.conns[port] = make(map[net.Conn]bool)
	go tm.acceptLoop(t)
	go tm.pingLoop(t)
	log.Printf("tunnel started on 127.0.0.1:%d", port)
}

func (tm *TunnelManager) stopTunnel(port int) {
	tm.mu.Lock()
	t, ok := tm.tunnels[port]
	conns := tm.conns[port]
	delete(tm.tunnels, port)
	delete(tm.conns, port)
	tm.mu.Unlock()
	if ok {
		close(t.stopCh)
		t.listener.Close()
		for c := range conns {
			c.Close()
		}
		log.Printf("tunnel stopped for port %d", port)
	}
}

func (tm *TunnelManager) HandlePong(port int, alive bool) {
	tm.mu.Lock()
	t, exists := tm.tunnels[port]
	tm.mu.Unlock()
	if !exists {
		return
	}
	if alive {
		t.misses = 0
	} else {
		t.misses++
		if t.misses >= maxMisses {
			log.Printf("port %d: %d misses, tearing down", port, t.misses)
			tm.stopTunnel(port)
		}
	}
}

func (tm *TunnelManager) ForwardData(port int, data []byte) {
	tm.mu.Lock()
	conns := tm.conns[port]
	tm.mu.Unlock()
	for c := range conns {
		c.Write(data)
	}
}

func (tm *TunnelManager) acceptLoop(t *tunnel) {
	for {
		select {
		case <-t.stopCh:
			return
		default:
		}
		conn, err := t.listener.Accept()
		if err != nil {
			select {
			case <-t.stopCh:
				return
			default:
				return
			}
		}
		tm.mu.Lock()
		tm.conns[t.port][conn] = true
		tm.mu.Unlock()
		go tm.handleConn(t.port, conn)
	}
}

func (tm *TunnelManager) handleConn(port int, conn net.Conn) {
	defer func() {
		conn.Close()
		tm.mu.Lock()
		delete(tm.conns[port], conn)
		tm.mu.Unlock()
	}()
	buf := make([]byte, 32*1024)
	for {
		n, err := conn.Read(buf)
		if err != nil {
			if err != io.EOF {
				log.Printf("read error port %d: %v", port, err)
			}
			return
		}
		enc := base64.StdEncoding.EncodeToString(buf[:n])
		tm.sendData(port, []byte(enc))
	}
}

func (tm *TunnelManager) pingLoop(t *tunnel) {
	ticker := time.NewTicker(pingInterval)
	defer ticker.Stop()
	for {
		select {
		case <-t.stopCh:
			return
		case <-ticker.C:
			tm.sendPing(t.port)
		}
	}
}
