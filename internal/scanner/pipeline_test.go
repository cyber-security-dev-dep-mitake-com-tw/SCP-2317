package scanner

import (
	"bytes"
	"encoding/base64"
	"testing"

	"github.com/example/scp2317-agent/internal/model"
)

func TestScanCleanMessage(t *testing.T) {
	v := NewPipeline().Scan(model.EmailScanRequest{
		MessageID:   "clean",
		Sender:      "sender@example.test",
		Subject:     "Quarterly update",
		BodyPreview: "Hello",
		URLs:        []string{"https://example.com"},
	})

	if v.Status != "clean" {
		t.Fatalf("expected clean, got %s", v.Status)
	}
	if v.Score != 0 {
		t.Fatalf("expected score 0, got %d", v.Score)
	}
	if len(v.Findings) != 0 {
		t.Fatalf("expected no findings, got %d", len(v.Findings))
	}
}

func TestScanSuspiciousURL(t *testing.T) {
	v := NewPipeline().Scan(model.EmailScanRequest{
		MessageID:   "url",
		Sender:      "sender@example.test",
		Subject:     "Suspicious link",
		BodyPreview: "Click carefully",
		URLs:        []string{"http://xn--demo.test"},
	})

	if v.Status != "suspicious" {
		t.Fatalf("expected suspicious, got %s", v.Status)
	}
	if v.Score != 25 {
		t.Fatalf("expected score 25, got %d", v.Score)
	}
}

func TestScanExecutableAttachment(t *testing.T) {
	v := NewPipeline().Scan(model.EmailScanRequest{
		MessageID: "attach",
		Sender:    "sender@example.test",
		Subject:   "Attachment risk",
		Attachments: []model.Artifact{
			{Name: "invoice.exe", ContentType: "application/octet-stream"},
		},
	})

	if v.Status != "malicious" {
		t.Fatalf("expected malicious, got %s", v.Status)
	}
	if v.Score != 55 {
		t.Fatalf("expected score 55, got %d", v.Score)
	}
	if len(v.Findings) == 0 {
		t.Fatal("expected findings for executable attachment")
	}
}

func TestScanLargeInlineAttachment(t *testing.T) {
	payload := bytes.Repeat([]byte{0}, 10*1024*1024+1)
	encoded := base64.StdEncoding.EncodeToString(payload)

	v := NewPipeline().Scan(model.EmailScanRequest{
		MessageID: "large",
		Sender:    "sender@example.test",
		Subject:   "Large payload",
		Attachments: []model.Artifact{
			{Name: "archive.zip", ContentType: "application/zip", DataBase64: encoded},
		},
	})

	if v.Status != "suspicious" {
		t.Fatalf("unexpected status %s", v.Status)
	}
	if v.Score != 10 {
		t.Fatalf("expected score 10, got %d", v.Score)
	}
}
