package client

import (
	"sync"
)

// ProxyManager tracks active server-side proxies keyed by remote port.
type ProxyManager struct {
	proxies map[int]int // remote port -> proxy port
	mu      sync.Mutex
}

func NewProxyManager() *ProxyManager {
	return &ProxyManager{proxies: make(map[int]int)}
}

func (pm *ProxyManager) Add(remotePort, proxyPort int) {
	pm.mu.Lock()
	pm.proxies[remotePort] = proxyPort
	pm.mu.Unlock()
}

func (pm *ProxyManager) Remove(remotePort int) {
	pm.mu.Lock()
	delete(pm.proxies, remotePort)
	pm.mu.Unlock()
}
