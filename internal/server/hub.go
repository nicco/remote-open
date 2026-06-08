package server

import (
	"log"
	"sync"

	"github.com/gorilla/websocket"
)

type Hub struct {
	Register  chan *websocket.Conn
	Broadcast chan []byte
	clients   map[*websocket.Conn]bool
	mu        sync.RWMutex
	done      chan struct{}
}

func NewHub() *Hub {
	return &Hub{
		Register:  make(chan *websocket.Conn),
		Broadcast: make(chan []byte, 256),
		clients:   make(map[*websocket.Conn]bool),
		done:      make(chan struct{}),
	}
}

func (h *Hub) Run() {
	for {
		select {
		case conn := <-h.Register:
			h.mu.Lock()
			h.clients[conn] = true
			h.mu.Unlock()
		case msg := <-h.Broadcast:
			h.mu.RLock()
			for conn := range h.clients {
				if err := conn.WriteMessage(websocket.TextMessage, msg); err != nil {
					log.Printf("write error, removing client: %v", err)
					conn.Close()
					h.mu.RUnlock()
					h.mu.Lock()
					delete(h.clients, conn)
					h.mu.Unlock()
					h.mu.RLock()
				}
			}
			h.mu.RUnlock()
		case <-h.done:
			h.mu.Lock()
			for conn := range h.clients {
				conn.Close()
			}
			h.clients = make(map[*websocket.Conn]bool)
			h.mu.Unlock()
			return
		}
	}
}

func (h *Hub) Stop() { close(h.done) }
