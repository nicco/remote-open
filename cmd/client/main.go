package main

import (
	"fmt"
	"log"
	"net/url"
	"os/exec"
	"strconv"
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

	chiselServer := deriveChiselServer(cfg.Server)
	tm := client.NewTunnelManager(chiselServer)

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
					time.Sleep(500 * time.Millisecond)
					localURL := fmt.Sprintf("http://127.0.0.1:%d", action.Port)
					exec.Command("open", localURL).Start()
				case client.ActionExternal:
					exec.Command("open", m.URL).Start()
				}
			default:
				log.Printf("unhandled message: %T", msg)
			}
		},
		func(writeFn func([]byte) error) {
			log.Printf("connection ready")
		},
	)

	conn.Run()
}

func deriveChiselServer(wsURL string) string {
	u, err := url.Parse(wsURL)
	if err != nil {
		return ""
	}
	host := u.Hostname()
	portStr := u.Port()
	if portStr == "" {
		portStr = "80"
	}
	port, _ := strconv.Atoi(portStr)
	return fmt.Sprintf("%s:%d", host, port+1)
}
