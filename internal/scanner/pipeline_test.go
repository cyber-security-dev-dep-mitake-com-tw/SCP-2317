package scanner

import (
	"encoding/base64"
	"strings"
	"testing"

	"github.com/example/scp2317-agent/internal/model"
)

func TestScanCleanHTTPS(t *testing.T) {
	v := NewPipeline().Scan(model.EmailScanRequest{
		MessageID: "t1",
		URLs:      []string{"https://example.com"},
	})
	if v.Status != "clean" || v.Score != 0 {
		t.Fatalf("%+v", v)
	}
}

func TestScanSuspiciousPunycodeHTTP(t *testing.T) {
	v := NewPipeline().Scan(model.EmailScanRequest{
		MessageID: "t2",
		URLs:      []string{"http://xn--demo.test"},
	})
	if v.Status != "suspicious" || v.Score != 25 {
		t.Fatalf("%+v", v)
	}
}

func TestScanMaliciousExecutable(t *testing.T) {
	v := NewPipeline().Scan(model.EmailScanRequest{
		MessageID: "t3",
		Attachments: []model.Artifact{{
			Name:        "invoice.exe",
			ContentType: "application/octet-stream",
		}},
	})
	if v.Status != "malicious" || v.Score != 55 {
		t.Fatalf("%+v", v)
	}
}

func TestScanOversizedInlineAttachment(t *testing.T) {
	data := base64.StdEncoding.EncodeToString(make([]byte, 10*1024*1024+1))
	v := NewPipeline().Scan(model.EmailScanRequest{
		MessageID: "t4",
		Attachments: []model.Artifact{{
			Name:       "big.bin",
			DataBase64: data,
		}},
	})
	if v.Status != "suspicious" || v.Score != 10 {
		t.Fatalf("%+v", v)
	}
}

func TestExtractURLsFromBody(t *testing.T) {
	urls := ExtractURLs(nil, "See http://xn--phish.test/login and https://safe.example/")
	if len(urls) != 2 {
		t.Fatalf("got %v", urls)
	}
	v := NewPipeline().Scan(model.EmailScanRequest{
		MessageID:   "t5",
		BodyPreview: "Click http://evil.test now",
	})
	if v.Status != "suspicious" {
		t.Fatalf("%+v", v)
	}
}

func TestValidateRequiresMessageID(t *testing.T) {
	err := NewPipeline().Validate(model.EmailScanRequest{})
	if err == nil || !strings.Contains(err.Error(), "message_id") {
		t.Fatalf("err=%v", err)
	}
}

func TestValidateRejectsBadBase64(t *testing.T) {
	err := NewPipeline().Validate(model.EmailScanRequest{
		MessageID: "t6",
		Attachments: []model.Artifact{{
			Name:       "x.bin",
			DataBase64: "%%%",
		}},
	})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestValidateMaxURLs(t *testing.T) {
	p := NewPipelineWithPolicy(Policy{MaxURLs: 2, MaxAttachments: 10, HighRiskExtensions: []string{".exe"}, InlineSizeLimitBytes: 1})
	urls := make([]string, 3)
	err := p.Validate(model.EmailScanRequest{MessageID: "t7", URLs: urls})
	if err == nil {
		t.Fatal("expected error")
	}
}
