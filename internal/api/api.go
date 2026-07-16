package api

import (
	"encoding/json"
	"net/http"

	"github.com/example/scp2317-agent/internal/model"
	"github.com/example/scp2317-agent/internal/scanner"
)

type Handler struct{ pipeline *scanner.Pipeline }

func New(p *scanner.Pipeline) *Handler {
	if p == nil {
		p = scanner.NewPipeline()
	}
	return &Handler{pipeline: p}
}
func (h *Handler) Routes() http.Handler {
	m := http.NewServeMux()
	m.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) { w.Write([]byte("ok")) })
	m.HandleFunc("POST /v1/email/scan", h.scan)
	return m
}
func (h *Handler) scan(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 25<<20)
	var req model.EmailScanRequest
	if json.NewDecoder(r.Body).Decode(&req) != nil {
		http.Error(w, "invalid request", 400)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(h.pipeline.Scan(req))
}
