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
