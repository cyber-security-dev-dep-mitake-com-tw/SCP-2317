package api

import (
	"encoding/json"
	"net/http"

	"github.com/example/scp2317-agent/internal/model"
	"github.com/example/scp2317-agent/internal/sandbox"
	"github.com/example/scp2317-agent/internal/scanner"
)

type Handler struct {
	pipeline *scanner.Pipeline
	sandbox  sandbox.Worker
}

func New(p *scanner.Pipeline) *Handler {
	if p == nil {
		p = scanner.NewPipeline()
	}
	return &Handler{pipeline: p, sandbox: sandbox.NewMemoryWorker()}
}

func (h *Handler) Routes() http.Handler {
	m := http.NewServeMux()
	m.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) { w.Write([]byte("ok")) })
	m.HandleFunc("POST /v1/email/scan", h.scan)
	m.HandleFunc("POST /v1/sandbox/submit", h.sandboxSubmit)
	m.HandleFunc("GET /v1/sandbox/status/{jobID}", h.sandboxStatus)
	m.HandleFunc("GET /v1/sandbox/report/{jobID}", h.sandboxReport)
	return m
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeAPIError(w http.ResponseWriter, status int, msg, details string) {
	writeJSON(w, status, model.APIError{Error: msg, Details: details})
}

func (h *Handler) scan(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 25<<20)
	var req model.EmailScanRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeAPIError(w, http.StatusBadRequest, "invalid request", "body must be valid JSON")
		return
	}
	if err := h.pipeline.Validate(req); err != nil {
		writeAPIError(w, http.StatusBadRequest, "validation failed", err.Error())
		return
	}

	// Dynamic analysis is disabled by default. Clients may pass ?dynamic=true
	// to obtain a not_executed sandbox job acknowledgement only.
	if r.URL.Query().Get("dynamic") == "true" {
		_, _ = h.sandbox.Submit(sandbox.SubmitRequest{
			Hash:      req.MessageID,
			ObjectRef: "inline:" + req.MessageID,
			Policy:    "policy-only",
		})
	}

	writeJSON(w, http.StatusOK, h.pipeline.Scan(req))
}

func (h *Handler) sandboxSubmit(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	var req sandbox.SubmitRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeAPIError(w, http.StatusBadRequest, "invalid request", "body must be valid JSON")
		return
	}
	if req.Hash == "" || req.ObjectRef == "" {
		writeAPIError(w, http.StatusBadRequest, "validation failed", "hash and objectRef are required")
		return
	}
	resp, err := h.sandbox.Submit(req)
	if err != nil {
		writeAPIError(w, http.StatusInternalServerError, "submit failed", err.Error())
		return
	}
	writeJSON(w, http.StatusAccepted, resp)
}

func (h *Handler) sandboxStatus(w http.ResponseWriter, r *http.Request) {
	jobID := r.PathValue("jobID")
	resp, err := h.sandbox.Status(jobID)
	if err != nil {
		writeAPIError(w, http.StatusNotFound, "job not found", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

func (h *Handler) sandboxReport(w http.ResponseWriter, r *http.Request) {
	jobID := r.PathValue("jobID")
	resp, err := h.sandbox.Report(jobID)
	if err != nil {
		writeAPIError(w, http.StatusNotFound, "job not found", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, resp)
}
