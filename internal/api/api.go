package api

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"strconv"

	"github.com/example/scp2317-agent/internal/easm"
	"github.com/example/scp2317-agent/internal/events"
	"github.com/example/scp2317-agent/internal/model"
	"github.com/example/scp2317-agent/internal/sandbox"
	"github.com/example/scp2317-agent/internal/scanner"
	"github.com/example/scp2317-agent/internal/store"
)

type Handler struct {
	pipeline *scanner.Pipeline
	sandbox  sandbox.Worker
	store    *store.Store
	sink     *events.Sink
}

func New(p *scanner.Pipeline) *Handler {
	if p == nil {
		p = scanner.NewPipeline()
	}
	st, _ := store.Open(os.Getenv("SCP2317_DATA"))
	return &Handler{
		pipeline: p,
		sandbox:  sandbox.NewMemoryWorker(),
		store:    st,
		sink:     events.NewSink(os.Getenv("SCP2317_SIEM_URL")),
	}
}

func (h *Handler) Routes() http.Handler {
	m := http.NewServeMux()
	m.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) { w.Write([]byte("ok")) })
	m.HandleFunc("POST /v1/email/scan", h.scan)
	m.HandleFunc("POST /v1/sandbox/submit", h.sandboxSubmit)
	m.HandleFunc("GET /v1/sandbox/status/{jobID}", h.sandboxStatus)
	m.HandleFunc("GET /v1/sandbox/report/{jobID}", h.sandboxReport)
	// Companion-app endpoints consumed by the tray UI.
	m.HandleFunc("GET /v1/events", h.events)
	m.HandleFunc("GET /v1/stats", h.stats)
	m.HandleFunc("POST /v1/urlcheck", h.urlCheck)
	m.HandleFunc("GET /v1/easm", h.easm)
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

	verdict := h.pipeline.Scan(req)
	h.record(store.Event{
		Kind:      "scan",
		MessageID: req.MessageID,
		Status:    verdict.Status,
		Score:     verdict.Score,
		Summary:   summarize(verdict),
	})
	writeJSON(w, http.StatusOK, verdict)
}

// record persists an event and forwards it to the SIEM sink (best effort).
func (h *Handler) record(e store.Event) {
	if h.store != nil {
		h.store.Append(e)
	}
	if h.sink != nil {
		h.sink.Send(e)
	}
}

func summarize(v model.Verdict) string {
	if len(v.Findings) == 0 {
		return "no findings"
	}
	return v.Findings[0].Summary
}

func (h *Handler) events(w http.ResponseWriter, r *http.Request) {
	n := 100
	if v := r.URL.Query().Get("n"); v != "" {
		if parsed, err := strconv.Atoi(v); err == nil && parsed > 0 {
			n = parsed
		}
	}
	writeJSON(w, http.StatusOK, h.store.Recent(n))
}

func (h *Handler) stats(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, h.store.Stats())
}

// urlCheck is the time-of-click endpoint: it re-scores a single URL on demand
// (e.g. when the user is about to open a link) using the same heuristics.
func (h *Handler) urlCheck(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	var req struct {
		URL string `json:"url"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.URL == "" {
		writeAPIError(w, http.StatusBadRequest, "invalid request", "url is required")
		return
	}
	verdict := h.pipeline.Scan(model.EmailScanRequest{
		MessageID: "urlcheck",
		URLs:      []string{req.URL},
	})
	h.record(store.Event{Kind: "urlcheck", Status: verdict.Status, Score: verdict.Score, Summary: req.URL})
	writeJSON(w, http.StatusOK, verdict)
}

func (h *Handler) easm(w http.ResponseWriter, r *http.Request) {
	domain := r.URL.Query().Get("domain")
	if domain == "" {
		writeAPIError(w, http.StatusBadRequest, "invalid request", "domain query parameter is required")
		return
	}
	res := easm.Check(context.Background(), easm.DefaultResolver(), domain)
	summary := "auth OK"
	if len(res.Findings) > 0 {
		summary = res.Findings[0]
	}
	h.record(store.Event{Kind: "easm", MessageID: domain, Summary: summary})
	writeJSON(w, http.StatusOK, res)
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
