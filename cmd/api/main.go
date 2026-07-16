package main

import (
	"log"
	"net/http"

	"github.com/example/scp2317-agent/internal/api"
	"github.com/example/scp2317-agent/internal/scanner"
)

func main() {
	h := api.New(scanner.NewPipeline())
	log.Println("SCP-2317 API listening on :8080")
	log.Fatal(http.ListenAndServe(":8080", h.Routes()))
}
