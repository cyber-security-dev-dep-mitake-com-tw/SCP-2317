package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/example/scp2317-agent/internal/scanner"
)

func TestRoutesHealthz(t *testing.T) {
	h := New(scanner.NewPipeline())
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	rec := httptest.NewRecorder()

	h.Routes().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	if body := rec.Body.String(); body != "ok" {
		t.Fatalf("expected ok body, got %q", body)
	}
}

func TestScanRejectsInvalidJSON(t *testing.T) {
	h := New(scanner.NewPipeline())
	req := httptest.NewRequest(http.MethodPost, "/v1/email/scan", bytes.NewBufferString("{"))
	rec := httptest.NewRecorder()

	h.Routes().ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestScanReturnsVerdict(t *testing.T) {
	h := New(scanner.NewPipeline())
	body := map[string]any{
		"message_id": "demo",
		"sender":     "sender@example.test",
		"subject":    "invoice",
		"urls":       []string{"http://xn--demo.test"},
		"attachments": []map[string]any{
			{"name": "invoice.exe", "content_type": "application/octet-stream"},
		},
	}
	payload, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("marshal request: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/v1/email/scan", bytes.NewReader(payload))
	rec := httptest.NewRecorder()

	h.Routes().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	if contentType := rec.Header().Get("Content-Type"); contentType != "application/json" {
		t.Fatalf("expected json content type, got %q", contentType)
	}
	var verdict map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &verdict); err != nil {
		t.Fatalf("unmarshal verdict: %v", err)
	}
	if verdict["status"] != "malicious" {
		t.Fatalf("expected malicious status, got %#v", verdict["status"])
	}
}
