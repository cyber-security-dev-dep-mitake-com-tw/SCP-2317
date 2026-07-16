package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
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
