package main

import (
	"log"
	"net/http"
	"os"

	"github.com/example/scp2317-agent/internal/api"
	"github.com/example/scp2317-agent/internal/scanner"
)

func main() {
	addr := os.Getenv("SCP2317_ADDR")
	if addr == "" {
		addr = "127.0.0.1:8080"
	}
	h := api.New(scanner.NewPipeline())
	log.Printf("SCP-2317 API listening on %s", addr)
	log.Fatal(http.ListenAndServe(addr, h.Routes()))
}
