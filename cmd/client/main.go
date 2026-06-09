package main

import (
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

	sshHost := extractHost(cfg.Server)
	sshUser := cfg.SSHUser
	if sshUser == "" {
		sshUser = "nicco"
	}
	tm := client.NewTunnelManager(sshHost, sshUser)

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
					log.Printf("starting SSH tunnel for port %d", action.Port)
					port := tm.StartTunnel(action.Port)
					if port > 0 {
						time.Sleep(500 * time.Millisecond)
						localURL := fmt.Sprintf("http://127.0.0.1:%d", port)
						exec.Command("open", localURL).Start()
					}
				case client.ActionExternal:
					exec.Command("open", m.URL).Start()
				}
			}
		},
		func(writeFn func([]byte) error) {
			log.Printf("connection ready")
		},
	)

	conn.Run()
}

func extractHost(wsURL string) string {
	u, _ := url.Parse(wsURL)
	return u.Hostname()
}
