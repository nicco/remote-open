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
