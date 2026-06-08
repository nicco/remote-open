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
