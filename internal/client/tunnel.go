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

// StartTunnel runs a chisel client to tunnel to the server's port.
// Returns the local Mac port to connect to (0 if failed).
func (tm *TunnelManager) StartTunnel(remotePort int) int {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	if _, exists := tm.processes[remotePort]; exists {
		return 50000 + (remotePort % 10000)
	}

	localPort := 50000 + (remotePort % 10000)
	remote := fmt.Sprintf("R:%d:127.0.0.1:%d", localPort, remotePort)
	cmd := exec.Command("chisel", "client", tm.chiselServer, remote)

	if err := cmd.Start(); err != nil {
		log.Printf("chisel start error port %d: %v", remotePort, err)
		return 0
	}

	tm.processes[remotePort] = cmd
	log.Printf("chisel tunnel: 127.0.0.1:%d -> server:%d", localPort, remotePort)
	return localPort
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
