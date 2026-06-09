package client

import (
	"fmt"
	"log"
	"os/exec"
	"sync"
)

// TunnelManager manages SSH tunnels to the server.
type TunnelManager struct {
	sshHost  string
	sshUser  string
	processes map[int]*exec.Cmd
	mu       sync.Mutex
}

func NewTunnelManager(sshHost, sshUser string) *TunnelManager {
	return &TunnelManager{
		sshHost:   sshHost,
		sshUser:   sshUser,
		processes: make(map[int]*exec.Cmd),
	}
}

// StartTunnel opens an SSH tunnel for the given port. Returns the local port.
func (tm *TunnelManager) StartTunnel(remotePort int) int {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	if _, exists := tm.processes[remotePort]; exists {
		return remotePort
	}

	target := fmt.Sprintf("%s@%s", tm.sshUser, tm.sshHost)
	spec := fmt.Sprintf("%d:localhost:%d", remotePort, remotePort)
	cmd := exec.Command("ssh", "-N", "-L", spec, target)

	if err := cmd.Start(); err != nil {
		log.Printf("ssh tunnel error port %d: %v", remotePort, err)
		return 0
	}

	tm.processes[remotePort] = cmd
	log.Printf("ssh tunnel: 127.0.0.1:%d -> %s:%d", remotePort, tm.sshHost, remotePort)
	return remotePort
}

func (tm *TunnelManager) StopTunnel(port int) {
	tm.mu.Lock()
	cmd, exists := tm.processes[port]
	if exists {
		delete(tm.processes, port)
	}
	tm.mu.Unlock()

	if exists && cmd.Process != nil {
		cmd.Process.Kill()
	}
}
