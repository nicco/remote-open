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
