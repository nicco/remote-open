package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/url"
	"os/exec"
	"time"

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

	serverHost := extractHost(cfg.Server)
	pm := client.NewProxyManager()

	var wsWrite func([]byte) error

	conn := client.NewConn(cfg.Server,
		func(raw []byte) {
			msg, err := protocol.Unmarshal(raw)
			if err != nil {
				return
			}
			switch m := msg.(type) {
			case protocol.OpenURL:
				log.Printf("open-url: %s", m.URL)
				action := client.RouteURL(m.URL)
				switch action.Kind {
				case client.ActionTunnel:
					// Request a server-side proxy
					req := protocol.StartProxy{
						Type: protocol.TypeStartProxy,
						Port: action.Port,
					}
					data, _ := json.Marshal(req)
					if wsWrite != nil {
						wsWrite(data)
					}
				case client.ActionExternal:
					exec.Command("open", m.URL).Start()
				}

			case protocol.ProxyStarted:
				log.Printf("proxy: port %d -> proxy port %d", m.Port, m.ProxyPort)
				pm.Add(m.Port, m.ProxyPort)
				time.Sleep(300 * time.Millisecond)
				localURL := fmt.Sprintf("http://%s:%d", serverHost, m.ProxyPort)
				exec.Command("open", localURL).Start()
			}
		},
		func(writeFn func([]byte) error) {
			wsWrite = writeFn
			log.Printf("connection ready")
		},
	)

	conn.Run()
}

func extractHost(wsURL string) string {
	u, _ := url.Parse(wsURL)
	return u.Hostname()
}
