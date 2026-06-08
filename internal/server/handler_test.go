package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/nicco/remote-open/internal/protocol"
)

func TestHandleOpen(t *testing.T) {
	hub := NewHub()
	go hub.Run()
	defer hub.Stop()
	handler := NewHandler(hub)
	upgrader := websocket.Upgrader{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/ws" {
			conn, _ := upgrader.Upgrade(w, r, nil)
			hub.Register <- conn
		} else {
			handler.ServeHTTP(w, r)
		}
	}))
	defer srv.Close()

	wsURL := "ws" + strings.TrimPrefix(srv.URL, "http")
	ws, _, _ := websocket.DefaultDialer.Dial(wsURL+"/ws", nil)
	defer ws.Close()
	time.Sleep(50 * time.Millisecond)

	resp, _ := http.Post(srv.URL+"/open", "text/plain", strings.NewReader("http://localhost:3000/path"))
	if resp.StatusCode != 200 {
		t.Errorf("status = %d", resp.StatusCode)
	}

	_, raw, _ := ws.ReadMessage()
	var msg protocol.OpenURL
	json.Unmarshal(raw, &msg)
	if msg.URL != "http://localhost:3000/path" {
		t.Errorf("URL = %q", msg.URL)
	}
}

func TestHandleOpenNoClients(t *testing.T) {
	hub := NewHub()
	go hub.Run()
	defer hub.Stop()
	srv := httptest.NewServer(NewHandler(hub))
	defer srv.Close()
	resp, _ := http.Post(srv.URL+"/open", "text/plain", strings.NewReader("http://localhost:3000"))
	if resp.StatusCode != 200 {
		t.Errorf("status = %d", resp.StatusCode)
	}
}
