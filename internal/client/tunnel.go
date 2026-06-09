package client

import (
	"fmt"
	"log"
	"os/exec"
	"sync"
)

// TunnelManager manages chisel tunnels per port.
type TunnelManager struct {
	chiselServer string
	processes    map[int]*exec.Cmd
	mu           sync.Mutex
}

// NewTunnelManager creates a TunnelManager. chiselServer is host:port of the chisel server.
func NewTunnelManager(chiselServer string) *TunnelManager {
	return &TunnelManager{
		chiselServer: chiselServer,
		processes:    make(map[int]*exec.Cmd),
	}
}

// StartTunnel runs a chisel client to tunnel the given port.
// chisel client <server> <local-port>:localhost:<remote-port>
func (tm *TunnelManager) StartTunnel(port int) {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	if _, exists := tm.processes[port]; exists {
		log.Printf("chisel tunnel for port %d already running", port)
		return
	}

	remote := fmt.Sprintf("R:%d", port)
	cmd := exec.Command("chisel", "client", tm.chiselServer, remote)
	cmd.Stdout = nil
	cmd.Stderr = nil

	if err := cmd.Start(); err != nil {
		log.Printf("chisel start error port %d: %v", port, err)
		return
	}

	tm.processes[port] = cmd
	log.Printf("chisel tunnel started: 127.0.0.1:%d -> server:%d", port, port)
}

// StopTunnel kills the chisel process for the given port.
func (tm *TunnelManager) StopTunnel(port int) {
	tm.mu.Lock()
	cmd, exists := tm.processes[port]
	if exists {
		delete(tm.processes, port)
	}
	tm.mu.Unlock()

	if exists && cmd.Process != nil {
		cmd.Process.Kill()
		log.Printf("chisel tunnel stopped for port %d", port)
	}
}
