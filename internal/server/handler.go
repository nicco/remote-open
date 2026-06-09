package server

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"sync"

	"github.com/gorilla/websocket"
	"github.com/nicco/remote-open/internal/protocol"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

type Handler struct {
	hub     *Hub
	proxies map[int]*proxy
	mu      sync.Mutex
}

type proxy struct {
	port     int
	listener net.Listener
	stopCh   chan struct{}
}

func NewHandler(hub *Hub) *Handler {
	return &Handler{
		hub:     hub,
		proxies: make(map[int]*proxy),
	}
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.URL.Path {
	case "/open":
		h.handleOpen(w, r)
	case "/ws":
		h.handleWS(w, r)
	default:
		http.NotFound(w, r)
	}
}

func (h *Handler) handleOpen(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	body, _ := io.ReadAll(r.Body)
	url := string(body)
	log.Printf("open-url: %s", url)
	msg := protocol.OpenURL{Type: protocol.TypeOpenURL, URL: url}
	data, _ := json.Marshal(msg)
	h.hub.Broadcast <- data
	w.WriteHeader(http.StatusOK)
}

func (h *Handler) handleWS(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("ws upgrade error: %v", err)
		return
	}
	log.Printf("client connected")
	h.hub.Register <- conn

	go func() {
		defer func() {
			conn.Close()
			h.cleanupClientProxies(conn)
		}()
		for {
			_, raw, err := conn.ReadMessage()
			if err != nil {
				return
			}
			h.handleWSMessage(conn, raw)
		}
	}()
}

func (h *Handler) handleWSMessage(conn *websocket.Conn, raw []byte) {
	msg, err := protocol.Unmarshal(raw)
	if err != nil {
		return
	}
	switch m := msg.(type) {
	case protocol.StartProxy:
		h.startProxy(conn, m.Port)
	case protocol.ProxyStop:
		h.stopProxy(m.ProxyPort)
	}
}

func (h *Handler) startProxy(conn *websocket.Conn, targetPort int) {
	h.mu.Lock()
	defer h.mu.Unlock()

	// Find a free port
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		log.Printf("proxy: cannot listen: %v", err)
		return
	}
	proxyPort := l.Addr().(*net.TCPAddr).Port

	p := &proxy{
		port:     proxyPort,
		listener: l,
		stopCh:   make(chan struct{}),
	}
	h.proxies[proxyPort] = p

	go h.proxyLoop(p, targetPort)

	log.Printf("proxy: port %d -> localhost:%d", proxyPort, targetPort)

	// Tell the client
	resp := protocol.ProxyStarted{
		Type:      protocol.TypeProxyStarted,
		Port:      targetPort,
		ProxyPort: proxyPort,
	}
	data, _ := json.Marshal(resp)
	conn.WriteMessage(websocket.TextMessage, data)
}

func (h *Handler) proxyLoop(p *proxy, targetPort int) {
	defer func() {
		p.listener.Close()
		h.mu.Lock()
		delete(h.proxies, p.port)
		h.mu.Unlock()
	}()

	for {
		select {
		case <-p.stopCh:
			return
		default:
		}

		client, err := p.listener.Accept()
		if err != nil {
			return
		}

		go func() {
			defer client.Close()
			target, err := net.Dial("tcp", fmt.Sprintf("127.0.0.1:%d", targetPort))
			if err != nil {
				return
			}
			defer target.Close()

			go io.Copy(target, client)
			io.Copy(client, target)
		}()
	}
}

func (h *Handler) stopProxy(proxyPort int) {
	h.mu.Lock()
	p, ok := h.proxies[proxyPort]
	h.mu.Unlock()
	if ok {
		close(p.stopCh)
		log.Printf("proxy: stopped port %d", proxyPort)
	}
}

func (h *Handler) cleanupClientProxies(conn *websocket.Conn) {
	// Could track per-client proxies, for now just keep them
}
