package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/example/scp2317-agent/internal/model"
)

func TestHealthz(t *testing.T) {
	srv := New(nil).Routes()
	rr := httptest.NewRecorder()
	srv.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/healthz", nil))
	if rr.Code != 200 || rr.Body.String() != "ok" {
		t.Fatalf("%d %q", rr.Code, rr.Body.String())
	}
}

func TestScanInvalidJSON(t *testing.T) {
	srv := New(nil).Routes()
	rr := httptest.NewRecorder()
	srv.ServeHTTP(rr, httptest.NewRequest(http.MethodPost, "/v1/email/scan", bytes.NewBufferString("{")))
	if rr.Code != 400 {
		t.Fatalf("code=%d", rr.Code)
	}
	var ae model.APIError
	if err := json.Unmarshal(rr.Body.Bytes(), &ae); err != nil || ae.Error == "" {
		t.Fatalf("body=%s", rr.Body.String())
	}
}

func TestScanMissingMessageID(t *testing.T) {
	srv := New(nil).Routes()
	body, _ := json.Marshal(map[string]any{"sender": "a@b.test", "subject": "x"})
	rr := httptest.NewRecorder()
	srv.ServeHTTP(rr, httptest.NewRequest(http.MethodPost, "/v1/email/scan", bytes.NewReader(body)))
	if rr.Code != 400 {
		t.Fatalf("code=%d body=%s", rr.Code, rr.Body.String())
	}
}

func TestScanMaliciousVerdictJSONTags(t *testing.T) {
	srv := New(nil).Routes()
	payload := model.EmailScanRequest{
		MessageID: "demo",
		Sender:    "billing@example.test",
		Subject:   "invoice",
		URLs:      []string{"http://xn--demo.test"},
		Attachments: []model.Artifact{{
			Name:        "invoice.exe",
			ContentType: "application/octet-stream",
		}},
	}
	body, _ := json.Marshal(payload)
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/v1/email/scan", bytes.NewReader(body))
	srv.ServeHTTP(rr, req)
	if rr.Code != 200 {
		t.Fatalf("code=%d", rr.Code)
	}
	if ct := rr.Header().Get("Content-Type"); ct != "application/json" {
		t.Fatalf("ct=%s", ct)
	}
	var v model.Verdict
	if err := json.Unmarshal(rr.Body.Bytes(), &v); err != nil {
		t.Fatal(err)
	}
	if v.Status != "malicious" {
		t.Fatalf("%+v", v)
	}
	if len(v.Findings) == 0 || v.Findings[0].Engine == "" {
		t.Fatalf("findings missing snake json fields: %s", rr.Body.String())
	}
	raw := rr.Body.String()
	if !bytes.Contains(rr.Body.Bytes(), []byte(`"engine"`)) {
		t.Fatalf("expected lowercase engine tag in %s", raw)
	}
}

func TestSandboxSubmitStatusReport(t *testing.T) {
	srv := New(nil).Routes()
	body, _ := json.Marshal(map[string]string{
		"hash": "deadbeef", "objectRef": "s3://bucket/obj", "policy": "default",
	})
	rr := httptest.NewRecorder()
	srv.ServeHTTP(rr, httptest.NewRequest(http.MethodPost, "/v1/sandbox/submit", bytes.NewReader(body)))
	if rr.Code != http.StatusAccepted {
		t.Fatalf("code=%d body=%s", rr.Code, rr.Body.String())
	}
	var sub struct {
		JobID string `json:"job_id"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &sub); err != nil || sub.JobID == "" {
		t.Fatalf("body=%s", rr.Body.String())
	}
	rr2 := httptest.NewRecorder()
	srv.ServeHTTP(rr2, httptest.NewRequest(http.MethodGet, "/v1/sandbox/status/"+sub.JobID, nil))
	if rr2.Code != 200 {
		t.Fatalf("status code=%d", rr2.Code)
	}
	rr3 := httptest.NewRecorder()
	srv.ServeHTTP(rr3, httptest.NewRequest(http.MethodGet, "/v1/sandbox/report/"+sub.JobID, nil))
	if rr3.Code != 200 {
		t.Fatalf("report code=%d", rr3.Code)
	}
	var rep struct {
		Status string `json:"status"`
	}
	_ = json.Unmarshal(rr3.Body.Bytes(), &rep)
	if rep.Status != "not_executed" {
		t.Fatalf("%+v", rep)
	}
}

func TestCompanionEndpoints(t *testing.T) {
	h := New(nil)
	srv := h.Routes()

	// A scan should record an event visible via /v1/events and /v1/stats.
	body, _ := json.Marshal(model.EmailScanRequest{MessageID: "e1", URLs: []string{"http://xn--demo.test"}})
	rr := httptest.NewRecorder()
	srv.ServeHTTP(rr, httptest.NewRequest(http.MethodPost, "/v1/email/scan", bytes.NewReader(body)))
	if rr.Code != 200 {
		t.Fatalf("scan code=%d", rr.Code)
	}

	rrE := httptest.NewRecorder()
	srv.ServeHTTP(rrE, httptest.NewRequest(http.MethodGet, "/v1/events?n=10", nil))
	if rrE.Code != 200 {
		t.Fatalf("events code=%d", rrE.Code)
	}
	var evs []map[string]any
	if err := json.Unmarshal(rrE.Body.Bytes(), &evs); err != nil || len(evs) == 0 {
		t.Fatalf("events body=%s", rrE.Body.String())
	}

	rrS := httptest.NewRecorder()
	srv.ServeHTTP(rrS, httptest.NewRequest(http.MethodGet, "/v1/stats", nil))
	if rrS.Code != 200 || !bytes.Contains(rrS.Body.Bytes(), []byte(`"total"`)) {
		t.Fatalf("stats body=%s", rrS.Body.String())
	}
}

func TestURLCheckEndpoint(t *testing.T) {
	srv := New(nil).Routes()
	body, _ := json.Marshal(map[string]string{"url": "http://xn--demo.test/login"})
	rr := httptest.NewRecorder()
	srv.ServeHTTP(rr, httptest.NewRequest(http.MethodPost, "/v1/urlcheck", bytes.NewReader(body)))
	if rr.Code != 200 {
		t.Fatalf("urlcheck code=%d body=%s", rr.Code, rr.Body.String())
	}
	var v model.Verdict
	if err := json.Unmarshal(rr.Body.Bytes(), &v); err != nil || v.Status == "clean" {
		t.Fatalf("expected non-clean verdict: %s", rr.Body.String())
	}
}

func TestURLCheckRequiresURL(t *testing.T) {
	srv := New(nil).Routes()
	rr := httptest.NewRecorder()
	srv.ServeHTTP(rr, httptest.NewRequest(http.MethodPost, "/v1/urlcheck", bytes.NewBufferString("{}")))
	if rr.Code != 400 {
		t.Fatalf("code=%d", rr.Code)
	}
}

// TestMain routes the default event store to a temp file so tests never write
// into the real per-user config directory (see dataPath).
func TestMain(m *testing.M) {
	dir, err := os.MkdirTemp("", "scp2317-test")
	if err == nil {
		os.Setenv("SCP2317_DATA", filepath.Join(dir, "events.jsonl"))
	}
	code := m.Run()
	if dir != "" {
		os.RemoveAll(dir)
	}
	os.Exit(code)
}
