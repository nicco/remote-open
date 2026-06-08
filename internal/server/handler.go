package server

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"time"

	"github.com/gorilla/websocket"
	"github.com/nicco/remote-open/internal/protocol"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

type Handler struct{ hub *Hub }

func NewHandler(hub *Hub) *Handler { return &Handler{hub: hub} }

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
		defer conn.Close()
		for {
			_, raw, err := conn.ReadMessage()
			if err != nil {
				log.Printf("client read error: %v", err)
				return
			}
			h.handleWSMessage(conn, raw)
		}
	}()
}

func (h *Handler) handleWSMessage(conn *websocket.Conn, raw []byte) {
	msg, err := protocol.Unmarshal(raw)
	if err != nil {
		log.Printf("unmarshal error: %v", err)
		return
	}
	switch m := msg.(type) {
	case protocol.Ping:
		alive := isPortAlive(m.Port)
		pong := protocol.Pong{Type: protocol.TypePong, Port: m.Port, Alive: alive}
		data, _ := json.Marshal(pong)
		conn.WriteMessage(websocket.TextMessage, data)
	case protocol.ProxyData:
		addr := fmt.Sprintf("127.0.0.1:%d", m.Port)
		c, err := net.DialTimeout("tcp", addr, 5*time.Second)
		if err != nil {
			log.Printf("proxy dial error port %d: %v", m.Port, err)
			return
		}
		decoded, err := base64.StdEncoding.DecodeString(m.Data)
		if err != nil {
			c.Close()
			log.Printf("proxy base64 decode error: %v", err)
			return
		}
		if _, err := c.Write(decoded); err != nil {
			c.Close()
			return
		}
		buf := make([]byte, 32*1024)
		n, err := c.Read(buf)
		c.Close()
		if err != nil && err.Error() != "EOF" {
			return
		}
		if n > 0 {
			resp := protocol.ProxyData{
				Type: protocol.TypeProxyData,
				Port: m.Port,
				Data: base64.StdEncoding.EncodeToString(buf[:n]),
			}
			data, _ := json.Marshal(resp)
			conn.WriteMessage(websocket.TextMessage, data)
		}
	default:
		log.Printf("unhandled message type: %T", msg)
	}
}

func isPortAlive(port int) bool {
	addr := fmt.Sprintf("127.0.0.1:%d", port)
	c, err := net.DialTimeout("tcp", addr, 2*time.Second)
	if err != nil {
		return false
	}
	c.Close()
	return true
}
