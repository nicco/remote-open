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

func TestHubRegisterAndBroadcast(t *testing.T) {
	hub := NewHub()
	go hub.Run()
	defer hub.Stop()

	upgrader := websocket.Upgrader{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			t.Errorf("upgrade: %v", err)
			return
		}
		hub.Register <- conn
	}))
	defer srv.Close()

	wsURL := "ws" + strings.TrimPrefix(srv.URL, "http")
	ws, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer ws.Close()
	time.Sleep(50 * time.Millisecond)

	msg := protocol.OpenURL{Type: "open-url", URL: "http://localhost:3000"}
	data, _ := json.Marshal(msg)
	hub.Broadcast <- data

	_, raw, err := ws.ReadMessage()
	if err != nil {
		t.Fatal(err)
	}
	expected := `{"type":"open-url","url":"http://localhost:3000"}`
	if string(raw) != expected {
		t.Errorf("got %s, want %s", raw, expected)
	}
}

func TestHubBroadcastNoClients(t *testing.T) {
	hub := NewHub()
	go hub.Run()
	defer hub.Stop()
	msg := protocol.OpenURL{Type: "open-url", URL: "http://localhost:3000"}
	data, _ := json.Marshal(msg)
	hub.Broadcast <- data
	time.Sleep(50 * time.Millisecond)
}

func TestHubClientDisconnect(t *testing.T) {
	hub := NewHub()
	go hub.Run()
	defer hub.Stop()
	upgrader := websocket.Upgrader{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, _ := upgrader.Upgrade(w, r, nil)
		hub.Register <- conn
	}))
	defer srv.Close()
	wsURL := "ws" + strings.TrimPrefix(srv.URL, "http")
	ws, _, _ := websocket.DefaultDialer.Dial(wsURL, nil)
	time.Sleep(50 * time.Millisecond)
	ws.Close()
	time.Sleep(50 * time.Millisecond)
	msg := protocol.OpenURL{Type: "open-url", URL: "http://localhost:3000"}
	data, _ := json.Marshal(msg)
	hub.Broadcast <- data
	time.Sleep(50 * time.Millisecond)
}
