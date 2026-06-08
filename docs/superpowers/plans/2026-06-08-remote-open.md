# remote-open Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Build a client/server app that forwards URLs from a Linux host (via xdg-open) to a macOS client, tunneling localhost URLs over WebSocket.

**Architecture:** Three Go binaries from one module — `cmd/shim` (xdg-open replacement), `cmd/server` (Docker container on port 20080), `cmd/client` (macOS daemon). Shared `internal/config` and `internal/protocol` packages. TCP tunnels multiplexed over a single WebSocket.

**Tech Stack:** Go 1.22+, gorilla/websocket, Docker multi-stage build

---

## File Map

| File | Purpose |
|---|---|
| `go.mod` | Module `github.com/nicco/remote-open` |
| `internal/config/config.go` | Read `~/.remote-open/config.json` |
| `internal/protocol/protocol.go` | Message types: OpenURL, Ping, Pong, ProxyData |
| `cmd/shim/main.go` | xdg-open shim: POST URL to server |
| `cmd/server/main.go` | Server entry: parse --port, start HTTP |
| `internal/server/hub.go` | WebSocket client registry + broadcast |
| `internal/server/handler.go` | HTTP handlers + ping/pong + proxy |
| `cmd/client/main.go` | Client entry: wire everything together |
| `internal/client/ws.go` | WebSocket connect + reconnect |
| `internal/client/router.go` | URL parsing, localhost detection |
| `internal/client/tunnel.go` | Tunnel lifecycle, ping loop, proxy forwarding |
| `Dockerfile` | Multi-stage build for server |

---

### Task 1: Project Scaffolding

- [ ] **Step 1: Init module and dirs**

```bash
cd /home/nicco/projects/remote-open
go mod init github.com/nicco/remote-open
mkdir -p cmd/shim cmd/server cmd/client
mkdir -p internal/config internal/protocol internal/server internal/client
```

- [ ] **Step 2: Create stubs**

```bash
for dir in cmd/shim cmd/server cmd/client; do
  echo 'package main; func main() {}' > $dir/main.go
done

echo 'package config
type Config struct { Server string `json:"server"` }' > internal/config/config.go

echo 'package protocol' > internal/protocol/protocol.go
echo 'package server' > internal/server/hub.go
echo 'package server' > internal/server/handler.go
echo 'package client' > internal/client/ws.go
echo 'package client' > internal/client/router.go
echo 'package client' > internal/client/tunnel.go
```

- [ ] **Step 3: Verify**

```bash
go build ./...
```

- [ ] **Step 4: Commit**

```bash
git add -A && git commit -m "chore: scaffold project structure"
```

---

### Task 2: Config Package

**Files:** create `internal/config/config_test.go`, modify `internal/config/config.go`

- [ ] **Step 1: Write test**

```bash
cat > internal/config/config_test.go << 'GOEOF'
package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadConfig(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.json")
	os.WriteFile(cfgPath, []byte(`{"server":"http://192.168.1.50:20080"}`), 0644)
	cfg, err := Load(cfgPath)
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}
	if cfg.Server != "http://192.168.1.50:20080" {
		t.Errorf("Server = %q", cfg.Server)
	}
}

func TestLoadConfigMissingFile(t *testing.T) {
	_, err := Load("/nonexistent/config.json")
	if err == nil {
		t.Error("expected error")
	}
}

func TestLoadConfigInvalidJSON(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.json")
	os.WriteFile(cfgPath, []byte("not json"), 0644)
	_, err := Load(cfgPath)
	if err == nil {
		t.Error("expected error")
	}
}

func TestDefaultPath(t *testing.T) {
	if DefaultPath() == "" {
		t.Error("DefaultPath should not be empty")
	}
}
GOEOF
```

- [ ] **Step 2: Run — expect FAIL**

```bash
go test ./internal/config/ -v
```

- [ ] **Step 3: Implement**

```bash
cat > internal/config/config.go << 'GOEOF'
package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

type Config struct {
	Server string `json:"server"`
}

func DefaultPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".remote-open", "config.json")
}

func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading config: %w", err)
	}
	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parsing config: %w", err)
	}
	return &cfg, nil
}
GOEOF
```

- [ ] **Step 4: Run — expect PASS**

```bash
go test ./internal/config/ -v
```

- [ ] **Step 5: Commit**

```bash
git add internal/config/ && git commit -m "feat: config package"
```

---

### Task 3: Protocol Package

**Files:** create `internal/protocol/protocol_test.go`, modify `internal/protocol/protocol.go`

- [ ] **Step 1: Write test**

```bash
cat > internal/protocol/protocol_test.go << 'GOEOF'
package protocol

import (
	"encoding/json"
	"testing"
)

func TestOpenURLMarshal(t *testing.T) {
	m := OpenURL{Type: "open-url", URL: "http://localhost:3000/path"}
	data, _ := json.Marshal(m)
	expected := `{"type":"open-url","url":"http://localhost:3000/path"}`
	if string(data) != expected {
		t.Errorf("got %s, want %s", data, expected)
	}
}

func TestOpenURLUnmarshal(t *testing.T) {
	m, _ := Unmarshal([]byte(`{"type":"open-url","url":"https://github.com"}`))
	o := m.(OpenURL)
	if o.URL != "https://github.com" {
		t.Errorf("URL = %q", o.URL)
	}
}

func TestPingMarshal(t *testing.T) {
	data, _ := json.Marshal(Ping{Type: "ping", Port: 3000})
	expected := `{"type":"ping","port":3000}`
	if string(data) != expected {
		t.Errorf("got %s", data, expected)
	}
}

func TestPongMarshal(t *testing.T) {
	data, _ := json.Marshal(Pong{Type: "pong", Port: 3000, Alive: true})
	expected := `{"type":"pong","port":3000,"alive":true}`
	if string(data) != expected {
		t.Errorf("got %s", data, expected)
	}
}

func TestPongUnmarshal(t *testing.T) {
	m, _ := Unmarshal([]byte(`{"type":"pong","port":3000,"alive":false}`))
	p := m.(Pong)
	if p.Alive {
		t.Error("expected Alive=false")
	}
}

func TestProxyDataMarshal(t *testing.T) {
	data, _ := json.Marshal(ProxyData{Type: "proxy-data", Port: 8080, Data: "aGVsbG8="})
	expected := `{"type":"proxy-data","port":8080,"data":"aGVsbG8="}`
	if string(data) != expected {
		t.Errorf("got %s", data, expected)
	}
}

func TestRoundTrip(t *testing.T) {
	orig := OpenURL{Type: "open-url", URL: "http://localhost:8000/docs"}
	data, _ := json.Marshal(orig)
	m, _ := Unmarshal(data)
	o := m.(OpenURL)
	if o.URL != orig.URL {
		t.Errorf("round trip: %q != %q", o.URL, orig.URL)
	}
}
GOEOF
```

- [ ] **Step 2: Run — expect FAIL**

```bash
go test ./internal/protocol/ -v
```

- [ ] **Step 3: Implement**

```bash
cat > internal/protocol/protocol.go << 'GOEOF'
package protocol

import "encoding/json"

const (
	TypeOpenURL   = "open-url"
	TypePing      = "ping"
	TypePong      = "pong"
	TypeProxyData = "proxy-data"
)

type OpenURL struct {
	Type string `json:"type"`
	URL  string `json:"url"`
}

type Ping struct {
	Type string `json:"type"`
	Port int    `json:"port"`
}

type Pong struct {
	Type  string `json:"type"`
	Port  int    `json:"port"`
	Alive bool   `json:"alive"`
}

type ProxyData struct {
	Type string `json:"type"`
	Port int    `json:"port"`
	Data string `json:"data"`
}

type generic struct{ Type string `json:"type"` }

func Unmarshal(data []byte) (interface{}, error) {
	var g generic
	if err := json.Unmarshal(data, &g); err != nil {
		return nil, err
	}
	switch g.Type {
	case TypeOpenURL:
		var m OpenURL; json.Unmarshal(data, &m); return m, nil
	case TypePing:
		var m Ping; json.Unmarshal(data, &m); return m, nil
	case TypePong:
		var m Pong; json.Unmarshal(data, &m); return m, nil
	case TypeProxyData:
		var m ProxyData; json.Unmarshal(data, &m); return m, nil
	default:
		return g, nil
	}
}
GOEOF
```

- [ ] **Step 4: Run — expect PASS**

```bash
go test ./internal/protocol/ -v
```

- [ ] **Step 5: Commit**

```bash
git add internal/protocol/ && git commit -m "feat: protocol message types"
```

---

### Task 4: Server Hub

**Files:** create `internal/server/hub_test.go`, modify `internal/server/hub.go`

- [ ] **Step 1: Add gorilla/websocket dep**

```bash
go get github.com/gorilla/websocket
```

- [ ] **Step 2: Write test**

```bash
cat > internal/server/hub_test.go << 'GOEOF'
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
GOEOF
```

- [ ] **Step 3: Run — expect FAIL**

```bash
go test ./internal/server/ -v -run TestHub
```

- [ ] **Step 4: Implement**

```bash
cat > internal/server/hub.go << 'GOEOF'
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
GOEOF
```

- [ ] **Step 5: Run — expect PASS**

```bash
go test ./internal/server/ -v -run TestHub
```

- [ ] **Step 6: Commit**

```bash
git add internal/server/ go.mod go.sum && git commit -m "feat: server hub"
```

---

### Task 5: Server Handlers

**Files:** create `internal/server/handler_test.go`, modify `internal/server/handler.go`

- [ ] **Step 1: Write test**

```bash
cat > internal/server/handler_test.go << 'GOEOF'
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
GOEOF
```

- [ ] **Step 2: Run — expect FAIL**

```bash
go test ./internal/server/ -v -run TestHandle
```

- [ ] **Step 3: Implement**

```bash
cat > internal/server/handler.go << 'GOEOF'
package server

import (
	"encoding/json"
	"io"
	"log"
	"net/http"

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
}
GOEOF
```

- [ ] **Step 4: Run — expect PASS**

```bash
go test ./internal/server/ -v -run TestHandle
```

- [ ] **Step 5: Commit**

```bash
git add internal/server/ && git commit -m "feat: server HTTP/WS handlers"
```

---

### Task 6: Server Ping/Pong

**Files:** modify `internal/server/handler.go`, create `internal/server/ping_test.go`

- [ ] **Step 1: Write test**

```bash
cat > internal/server/ping_test.go << 'GOEOF'
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
GOEOF
```

- [ ] **Step 2: Run — expect FAIL**

```bash
go test ./internal/server/ -v -run TestHandlePing
```

- [ ] **Step 3: Update handler.go** — replace `handleWS` and add `handleWSMessage` + `isPortAlive`

The `handler.go` file needs to be updated. Replace `handleWS` with the version that starts a read loop, and add the two new functions. The updated `handler.go` becomes:

```go
package server

import (
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
		// TODO: Task 7 (proxy forwarding from client → linux localhost)
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
```

- [ ] **Step 4: Run — expect PASS**

```bash
go test ./internal/server/ -v -run TestHandlePing
```

- [ ] **Step 5: Commit**

```bash
git add internal/server/ && git commit -m "feat: server ping/pong handler"
```

---

### Task 7: Server Proxy-Data Forwarding

**Files:** modify `internal/server/handler.go`

- [ ] **Step 1: Add proxy handling to handleWSMessage**

Replace the `case protocol.ProxyData:` section in `handleWSMessage` with:

```go
	case protocol.ProxyData:
		// Forward data to the localhost service on Linux
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
		// Read response and send back
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
```

Also add `"encoding/base64"` to the import block of `handler.go`.

- [ ] **Step 2: Verify compilation**

```bash
go build ./internal/server/
```

- [ ] **Step 3: Commit**

```bash
git add internal/server/ && git commit -m "feat: server proxy-data forwarding"
```

> **Note on proxy:** This is a simple request-response model. For true bidirectional TCP proxying (HTTP requests with large responses, streaming), a production version would need per-tunnel goroutines that continuously pipe in both directions. The current implementation handles single-shot TCP requests (like HTTP health checks or small API calls). This is sufficient for the initial version.

---

### Task 8: Server Entry Point

**Files:** modify `cmd/server/main.go`

- [ ] **Step 1: Implement**

```bash
cat > cmd/server/main.go << 'GOEOF'
package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"

	"github.com/nicco/remote-open/internal/server"
)

func main() {
	port := flag.Int("port", 20080, "port to listen on")
	flag.Parse()

	hub := server.NewHub()
	go hub.Run()

	handler := server.NewHandler(hub)
	addr := fmt.Sprintf(":%d", *port)
	log.Printf("remote-open-server listening on %s", addr)
	if err := http.ListenAndServe(addr, handler); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
GOEOF
```

- [ ] **Step 2: Verify**

```bash
go build ./cmd/server/
```

- [ ] **Step 3: Commit**

```bash
git add cmd/server/ && git commit -m "feat: server entry point"
```

---

### Task 9: Dockerfile

**Files:** create `Dockerfile`

- [ ] **Step 1: Write Dockerfile**

```bash
cat > Dockerfile << 'DOCEOF'
FROM golang:1.22-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /remote-open-server ./cmd/server/

FROM alpine:3.20
RUN apk --no-cache add ca-certificates
COPY --from=builder /remote-open-server /usr/local/bin/remote-open-server
EXPOSE 20080
ENTRYPOINT ["remote-open-server"]
DOCEOF
```

- [ ] **Step 2: Build and test**

```bash
docker build -t remote-open-server .
docker run --rm -d --network host --name test-server remote-open-server --port 20080
sleep 1
curl http://localhost:20080/open -X POST -d "http://example.com" && echo "OK"
docker stop test-server
```

- [ ] **Step 3: Commit**

```bash
git add Dockerfile && git commit -m "feat: Dockerfile"
```

---

### Task 10: Client URL Router

**Files:** create `internal/client/router_test.go`, modify `internal/client/router.go`

- [ ] **Step 1: Write test**

```bash
cat > internal/client/router_test.go << 'GOEOF'
package client

import "testing"

func TestRouteURLLocalhost(t *testing.T) {
	tests := []struct {
		url      string
		external bool
		port     int
	}{
		{"http://localhost:3000/path", false, 3000},
		{"http://127.0.0.1:8080", false, 8080},
		{"http://localhost:9999/foo", false, 9999},
		{"http://localhost", false, 80},
		{"https://localhost", false, 443},
		{"https://github.com", true, 0},
		{"http://example.com:3000", true, 0},
		{"http://192.168.1.1:3000", true, 0},
	}

	for _, tc := range tests {
		action := RouteURL(tc.url)
		if tc.external {
			if action.Kind != ActionExternal || action.URL != tc.url {
				t.Errorf("RouteURL(%q): expected external, got %v", tc.url, action)
			}
		} else {
			if action.Kind != ActionTunnel || action.Port != tc.port {
				t.Errorf("RouteURL(%q): expected tunnel port %d, got %v",
					tc.url, tc.port, action)
			}
		}
	}
}
GOEOF
```

- [ ] **Step 2: Run — expect FAIL**

```bash
go test ./internal/client/ -v -run TestRouteURL
```

- [ ] **Step 3: Implement**

```bash
cat > internal/client/router.go << 'GOEOF'
package client

import (
	"net/url"
	"strconv"
	"strings"
)

type ActionKind int

const (
	ActionExternal ActionKind = iota
	ActionTunnel
)

type RouteAction struct {
	Kind ActionKind
	URL  string
	Port int
}

func RouteURL(rawURL string) RouteAction {
	if isLocalhost(rawURL) {
		port, _ := extractPort(rawURL)
		return RouteAction{Kind: ActionTunnel, URL: rawURL, Port: port}
	}
	return RouteAction{Kind: ActionExternal, URL: rawURL}
}

func isLocalhost(rawURL string) bool {
	u, err := url.Parse(rawURL)
	if err != nil {
		return false
	}
	host := strings.Split(u.Host, ":")[0]
	return host == "localhost" || host == "127.0.0.1"
}

func extractPort(rawURL string) (int, bool) {
	u, err := url.Parse(rawURL)
	if err != nil {
		return 0, false
	}
	if strings.Contains(u.Host, ":") {
		parts := strings.Split(u.Host, ":")
		port, err := strconv.Atoi(parts[1])
		if err != nil {
			return 0, false
		}
		return port, true
	}
	if u.Scheme == "https" {
		return 443, true
	}
	return 80, true
}
GOEOF
```

- [ ] **Step 4: Run — expect PASS**

```bash
go test ./internal/client/ -v -run TestRouteURL
```

- [ ] **Step 5: Commit**

```bash
git add internal/client/ && git commit -m "feat: client URL router"
```

---

### Task 11: Client WebSocket Connection

**Files:** modify `internal/client/ws.go`

- [ ] **Step 1: Implement**

```bash
cat > internal/client/ws.go << 'GOEOF'
package client

import (
	"log"
	"time"

	"github.com/gorilla/websocket"
)

type Conn struct {
	serverURL string
	onMessage func([]byte)
	onReady   func(writeFn func([]byte) error)
}

func NewConn(serverURL string, onMessage func([]byte), onReady func(writeFn func([]byte) error)) *Conn {
	return &Conn{serverURL: serverURL, onMessage: onMessage, onReady: onReady}
}

func (c *Conn) Run() {
	backoff := 1 * time.Second
	const maxBackoff = 30 * time.Second

	for {
		log.Printf("connecting to %s", c.serverURL)
		conn, _, err := websocket.DefaultDialer.Dial(c.serverURL, nil)
		if err != nil {
			log.Printf("connect failed: %v (retry in %v)", err, backoff)
			time.Sleep(backoff)
			backoff *= 2
			if backoff > maxBackoff {
				backoff = maxBackoff
			}
			continue
		}

		log.Printf("connected to %s", c.serverURL)
		backoff = 1 * time.Second

		c.onReady(func(data []byte) error {
			return conn.WriteMessage(websocket.TextMessage, data)
		})

		for {
			_, raw, err := conn.ReadMessage()
			if err != nil {
				log.Printf("read error: %v (reconnecting)", err)
				conn.Close()
				break
			}
			c.onMessage(raw)
		}

		log.Printf("disconnected, reconnecting in %v", backoff)
		time.Sleep(backoff)
		backoff *= 2
		if backoff > maxBackoff {
			backoff = maxBackoff
		}
	}
}
GOEOF
```

- [ ] **Step 2: Verify**

```bash
go build ./internal/client/
```

- [ ] **Step 3: Commit**

```bash
git add internal/client/ && git commit -m "feat: client WebSocket with reconnect"
```

---

### Task 12: Client Tunnel Manager

**Files:** modify `internal/client/tunnel.go`

- [ ] **Step 1: Implement**

```bash
cat > internal/client/tunnel.go << 'GOEOF'
package client

import (
	"encoding/base64"
	"fmt"
	"io"
	"log"
	"net"
	"sync"
	"time"
)

type TunnelManager struct {
	tunnels  map[int]*tunnel
	mu       sync.Mutex
	sendData func(port int, data []byte)
	sendPing func(port int)
	conns    map[int]map[net.Conn]bool
}

type tunnel struct {
	port     int
	listener net.Listener
	misses   int
	stopCh   chan struct{}
}

const pingInterval = 5 * time.Second
const maxMisses = 3

func NewTunnelManager(sendData func(port int, data []byte), sendPing func(port int)) *TunnelManager {
	return &TunnelManager{
		tunnels:  make(map[int]*tunnel),
		conns:    make(map[int]map[net.Conn]bool),
		sendData: sendData,
		sendPing: sendPing,
	}
}

func (tm *TunnelManager) StartTunnel(port int) {
	tm.mu.Lock()
	defer tm.mu.Unlock()
	if _, exists := tm.tunnels[port]; exists {
		return
	}
	addr := fmt.Sprintf("127.0.0.1:%d", port)
	l, err := net.Listen("tcp", addr)
	if err != nil {
		log.Printf("cannot listen on %s: %v", addr, err)
		return
	}
	t := &tunnel{port: port, listener: l, stopCh: make(chan struct{})}
	tm.tunnels[port] = t
	tm.conns[port] = make(map[net.Conn]bool)
	go tm.acceptLoop(t)
	go tm.pingLoop(t)
	log.Printf("tunnel started on 127.0.0.1:%d", port)
}

func (tm *TunnelManager) stopTunnel(port int) {
	tm.mu.Lock()
	t, ok := tm.tunnels[port]
	conns := tm.conns[port]
	delete(tm.tunnels, port)
	delete(tm.conns, port)
	tm.mu.Unlock()
	if ok {
		close(t.stopCh)
		t.listener.Close()
		for c := range conns {
			c.Close()
		}
		log.Printf("tunnel stopped for port %d", port)
	}
}

func (tm *TunnelManager) HandlePong(port int, alive bool) {
	tm.mu.Lock()
	t, exists := tm.tunnels[port]
	tm.mu.Unlock()
	if !exists {
		return
	}
	if alive {
		t.misses = 0
	} else {
		t.misses++
		if t.misses >= maxMisses {
			log.Printf("port %d: %d misses, tearing down", port, t.misses)
			tm.stopTunnel(port)
		}
	}
}

func (tm *TunnelManager) ForwardData(port int, data []byte) {
	tm.mu.Lock()
	conns := tm.conns[port]
	tm.mu.Unlock()
	for c := range conns {
		c.Write(data)
	}
}

func (tm *TunnelManager) acceptLoop(t *tunnel) {
	for {
		select {
		case <-t.stopCh:
			return
		default:
		}
		conn, err := t.listener.Accept()
		if err != nil {
			select {
			case <-t.stopCh:
				return
			default:
				return
			}
		}
		tm.mu.Lock()
		tm.conns[t.port][conn] = true
		tm.mu.Unlock()
		go tm.handleConn(t.port, conn)
	}
}

func (tm *TunnelManager) handleConn(port int, conn net.Conn) {
	defer func() {
		conn.Close()
		tm.mu.Lock()
		delete(tm.conns[port], conn)
		tm.mu.Unlock()
	}()
	buf := make([]byte, 32*1024)
	for {
		n, err := conn.Read(buf)
		if err != nil {
			if err != io.EOF {
				log.Printf("read error port %d: %v", port, err)
			}
			return
		}
		enc := base64.StdEncoding.EncodeToString(buf[:n])
		tm.sendData(port, []byte(enc))
	}
}

func (tm *TunnelManager) pingLoop(t *tunnel) {
	ticker := time.NewTicker(pingInterval)
	defer ticker.Stop()
	for {
		select {
		case <-t.stopCh:
			return
		case <-ticker.C:
			tm.sendPing(t.port)
		}
	}
}
GOEOF
```

- [ ] **Step 2: Verify**

```bash
go build ./internal/client/
```

- [ ] **Step 3: Commit**

```bash
git add internal/client/ && git commit -m "feat: client tunnel manager"
```

---

### Task 13: Client Entry Point

**Files:** modify `cmd/client/main.go`

- [ ] **Step 1: Implement**

```bash
cat > cmd/client/main.go << 'GOEOF'
package main

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"os/exec"

	"github.com/nicco/remote-open/internal/client"
	"github.com/nicco/remote-open/internal/config"
	"github.com/nicco/remote-open/internal/protocol"
)

func main() {
	cfg, err := config.Load(config.DefaultPath())
	if err != nil {
		log.Fatalf("config: %v", err)
	}
	if cfg.Server == "" {
		log.Fatal("config: server URL not set")
	}

	var wsWrite func([]byte) error

	tm := client.NewTunnelManager(
		func(port int, data []byte) {
			if wsWrite == nil {
				return
			}
			msg := protocol.ProxyData{
				Type: protocol.TypeProxyData,
				Port: port,
				Data: string(data),
			}
			raw, _ := json.Marshal(msg)
			wsWrite(raw)
		},
		func(port int) {
			if wsWrite == nil {
				return
			}
			msg := protocol.Ping{Type: protocol.TypePing, Port: port}
			raw, _ := json.Marshal(msg)
			wsWrite(raw)
		},
	)

	conn := client.NewConn(cfg.Server,
		func(raw []byte) {
			msg, err := protocol.Unmarshal(raw)
			if err != nil {
				log.Printf("unmarshal: %v", err)
				return
			}
			switch m := msg.(type) {
			case protocol.OpenURL:
				log.Printf("open-url: %s", m.URL)
				action := client.RouteURL(m.URL)
				switch action.Kind {
				case client.ActionTunnel:
					tm.StartTunnel(action.Port)
					localURL := fmt.Sprintf("http://127.0.0.1:%d", action.Port)
					exec.Command("open", localURL).Start()
				case client.ActionExternal:
					exec.Command("open", m.URL).Start()
				}
			case protocol.Pong:
				tm.HandlePong(m.Port, m.Alive)
			case protocol.ProxyData:
				decoded, err := base64.StdEncoding.DecodeString(m.Data)
				if err != nil {
					log.Printf("proxy base64 decode: %v", err)
					return
				}
				tm.ForwardData(m.Port, decoded)
			default:
				log.Printf("unhandled message: %T", msg)
			}
		},
		func(writeFn func([]byte) error) {
			wsWrite = writeFn
			log.Printf("connection ready")
		},
	)

	conn.Run()
}
GOEOF
```

- [ ] **Step 2: Build for macOS**

```bash
GOOS=darwin GOARCH=amd64 go build -o remote-open-client ./cmd/client/
```

- [ ] **Step 3: Commit**

```bash
git add cmd/client/ && git commit -m "feat: client entry point"
```

---

### Task 14: Shim

**Files:** modify `cmd/shim/main.go`

- [ ] **Step 1: Implement**

```bash
cat > cmd/shim/main.go << 'GOEOF'
package main

import (
	"io"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/nicco/remote-open/internal/config"
)

func main() {
	if len(os.Args) < 2 {
		return
	}
	url := os.Args[1]
	if !strings.HasPrefix(url, "http://") && !strings.HasPrefix(url, "https://") {
		return
	}

	cfg, err := config.Load(config.DefaultPath())
	if err != nil {
		log.Printf("config error: %v", err)
		return
	}

	resp, err := http.Post(cfg.Server+"/open", "text/plain", strings.NewReader(url))
	if err != nil {
		log.Printf("post error: %v", err)
		return
	}
	defer resp.Body.Close()
	io.Copy(io.Discard, resp.Body)
}
GOEOF
```

- [ ] **Step 2: Build for Linux**

```bash
GOOS=linux GOARCH=amd64 go build -o remote-open-shim ./cmd/shim/
```

- [ ] **Step 3: Commit**

```bash
git add cmd/shim/ && git commit -m "feat: xdg-open shim"
```

---

### Task 15: End-to-End Validation

- [ ] **Step 1: Run all tests**

```bash
go test ./... -v
```

Expected: all tests pass.

- [ ] **Step 2: Verify all binaries compile for target platforms**

```bash
GOOS=linux GOARCH=amd64 go build -o /dev/null ./cmd/shim/
GOOS=linux GOARCH=amd64 go build -o /dev/null ./cmd/server/
GOOS=darwin GOARCH=amd64 go build -o /dev/null ./cmd/client/
```

Expected: all compile without errors.

- [ ] **Step 3: Manual smoke test with Docker server + local client (on Linux)**

```bash
# Terminal 1: Start server
docker run --rm --network host remote-open-server --port 20080

# Terminal 2: Start client (connecting to localhost for testing)
# Note: on Linux the client can't call macOS `open`, but WS flow can be verified
echo '{"server":"ws://localhost:20080/ws"}' > ~/.remote-open/config.json
go run ./cmd/client/ &
sleep 1

# Terminal 3: Trigger an external URL
curl http://localhost:20080/open -X POST -d "https://example.com"
# Client should log: "open-url: https://example.com"

# Trigger a localhost URL (start a dummy listener first)
nc -l 9999 &
curl http://localhost:20080/open -X POST -d "http://localhost:9999"
# Client should log: "tunnel started on 127.0.0.1:9999"
# Ping should succeed (nc is listening), ping loop starts
kill %1  # Kill nc
# After ~15s (3 misses * 5s), client should log: "tunnel stopped for port 9999"
```

- [ ] **Step 4: Commit if any fixes were made**

```bash
git add -A && git commit -m "chore: final validation"
```
