package server

import (
	"encoding/json"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/nicco/remote-open/internal/protocol"
)

func TestHandlePingAlive(t *testing.T) {
	l, _ := net.Listen("tcp", "127.0.0.1:0")
	port := l.Addr().(*net.TCPAddr).Port
	defer l.Close()

	hub := NewHub()
	h := NewHandler(hub)
	go hub.Run()
	defer hub.Stop()

	upgrader := websocket.Upgrader{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/ws" {
			conn, _ := upgrader.Upgrade(w, r, nil)
			hub.Register <- conn
			for {
				_, raw, err := conn.ReadMessage()
				if err != nil {
					return
				}
				h.handleWSMessage(conn, raw)
			}
		} else {
			h.ServeHTTP(w, r)
		}
	}))
	defer srv.Close()

	wsURL := "ws" + strings.TrimPrefix(srv.URL, "http")
	ws, _, _ := websocket.DefaultDialer.Dial(wsURL+"/ws", nil)
	defer ws.Close()
	time.Sleep(50 * time.Millisecond)

	ping, _ := json.Marshal(protocol.Ping{Type: "ping", Port: port})
	ws.WriteMessage(websocket.TextMessage, ping)

	_, raw, _ := ws.ReadMessage()
	var pong protocol.Pong
	json.Unmarshal(raw, &pong)
	if !pong.Alive {
		t.Error("expected Alive=true")
	}
}

func TestHandlePingDead(t *testing.T) {
	hub := NewHub()
	h := NewHandler(hub)
	go hub.Run()
	defer hub.Stop()

	upgrader := websocket.Upgrader{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/ws" {
			conn, _ := upgrader.Upgrade(w, r, nil)
			hub.Register <- conn
			for {
				_, raw, err := conn.ReadMessage()
				if err != nil {
					return
				}
				h.handleWSMessage(conn, raw)
			}
		} else {
			h.ServeHTTP(w, r)
		}
	}))
	defer srv.Close()

	wsURL := "ws" + strings.TrimPrefix(srv.URL, "http")
	ws, _, _ := websocket.DefaultDialer.Dial(wsURL+"/ws", nil)
	defer ws.Close()
	time.Sleep(50 * time.Millisecond)

	ping, _ := json.Marshal(protocol.Ping{Type: "ping", Port: 19999})
	ws.WriteMessage(websocket.TextMessage, ping)

	_, raw, _ := ws.ReadMessage()
	var pong protocol.Pong
	json.Unmarshal(raw, &pong)
	if pong.Alive {
		t.Error("expected Alive=false")
	}
}
