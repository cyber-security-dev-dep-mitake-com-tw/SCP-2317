package scanner

import (
	"testing"

	"github.com/example/scp2317-agent/internal/model"
)

func TestIOCDomainBlocklist(t *testing.T) {
	// Uses the default rules/ioc.json placeholder domain.
	v := NewPipeline().Scan(model.EmailScanRequest{
		MessageID: "i1",
		Sender:    "x@known-c2-drop.invalid",
	})
	if !hasCategory(v, "threat-intel") {
		t.Fatalf("expected threat-intel finding, got %+v", v)
	}
}

func TestIOCHashBlocklist(t *testing.T) {
	v := NewPipeline().Scan(model.EmailScanRequest{
		MessageID: "i2",
		Attachments: []model.Artifact{{
			Name:   "payload.bin",
			SHA256: "275a021bbfb6489e54d471899f7db9d1663fc695ec2fe2a2c4538aabf651fd0f",
		}},
	})
	if !hasCategory(v, "threat-intel") {
		t.Fatalf("expected hash IOC finding, got %+v", v)
	}
}

func TestDoubleExtensionMasquerade(t *testing.T) {
	v := trustedPipeline().Scan(model.EmailScanRequest{
		MessageID: "x1",
		Attachments: []model.Artifact{{
			Name:        "invoice.pdf.png",
			ContentType: "image/png",
		}},
	})
	found := false
	for _, f := range v.Findings {
		if f.Engine == "attachment-name" {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected double-extension finding, got %+v", v)
	}
}

func TestSaaSAbuseWithLure(t *testing.T) {
	v := trustedPipeline().Scan(model.EmailScanRequest{
		MessageID:   "s1",
		Subject:     "verify your account",
		BodyPreview: "Please sign in here: https://docs.google.com/forms/d/e/abc/viewform",
	})
	found := false
	for _, f := range v.Findings {
		if f.Engine == "url-context" {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected SaaS-abuse finding, got %+v", v)
	}
}

func TestLookalikeLinkInBody(t *testing.T) {
	v := trustedPipeline().Scan(model.EmailScanRequest{
		MessageID:   "l1",
		BodyPreview: "Update at https://paypa1.com/account now",
	})
	if !hasCategory(v, "spoofing") {
		t.Fatalf("expected look-alike link spoofing finding, got %+v", v)
	}
}

func TestRouteForensicsLongChain(t *testing.T) {
	chain := make([]string, 10)
	for i := range chain {
		chain[i] = "from relay.example.com by mx.example.com"
	}
	v := trustedPipeline().Scan(model.EmailScanRequest{
		MessageID:     "r1",
		ReceivedChain: chain,
	})
	if !hasCategory(v, "route-forensics") {
		t.Fatalf("expected route-forensics finding, got %+v", v)
	}
}

func TestSocialSympathyGiftCard(t *testing.T) {
	v := trustedPipeline().Scan(model.EmailScanRequest{
		MessageID:   "g1",
		Subject:     "quick favor",
		BodyPreview: "Are you available? I need your help to buy gift card urgently.",
	})
	if !hasCategory(v, "social-engineering") {
		t.Fatalf("expected social-engineering finding, got %+v", v)
	}
}
