package scanner

import (
	"encoding/base64"
	"strings"
	"testing"

	"github.com/example/scp2317-agent/internal/model"
	"github.com/example/scp2317-agent/internal/qr"
)

func hasCategory(v model.Verdict, cat string) bool {
	for _, f := range v.Findings {
		if f.Category == cat {
			return true
		}
	}
	return false
}

func trustedPipeline() *Pipeline {
	return NewPipelineWithPolicy(Policy{
		HighRiskExtensions:   []string{".exe"},
		InlineSizeLimitBytes: 10 << 20,
		MaxURLs:              100,
		MaxAttachments:       50,
		TrustedDomains:       []string{"mitake.com.tw", "paypal.com"},
	})
}

func TestAuthEngineDMARCFail(t *testing.T) {
	v := trustedPipeline().Scan(model.EmailScanRequest{
		MessageID:   "a1",
		AuthResults: "mx.google.com; spf=pass; dkim=pass; dmarc=fail (p=reject)",
	})
	if !hasCategory(v, "authentication") || v.Status == "clean" {
		t.Fatalf("expected authentication finding, got %+v", v)
	}
}

func TestSpoofingDisplayNameImpersonation(t *testing.T) {
	v := trustedPipeline().Scan(model.EmailScanRequest{
		MessageID:     "s1",
		Sender:        "billing@secure-paypal-support.test",
		SenderDisplay: "PayPal Billing",
	})
	if !hasCategory(v, "spoofing") {
		t.Fatalf("expected spoofing finding, got %+v", v)
	}
}

func TestSpoofingLookalikeDomain(t *testing.T) {
	// paypa1.com (digit one) is edit-distance 1 from trusted paypal.com.
	v := trustedPipeline().Scan(model.EmailScanRequest{
		MessageID: "s2",
		Sender:    "service@paypa1.com",
	})
	if !hasCategory(v, "spoofing") {
		t.Fatalf("expected look-alike spoofing finding, got %+v", v)
	}
}

func TestSpoofingReplyToMismatch(t *testing.T) {
	v := trustedPipeline().Scan(model.EmailScanRequest{
		MessageID: "s3",
		Sender:    "vendor@partner.example",
		ReplyTo:   "attacker@evil.test",
	})
	if !hasCategory(v, "spoofing") {
		t.Fatalf("expected reply-to spoof finding, got %+v", v)
	}
}

func TestSocialEngineeringBEC(t *testing.T) {
	v := trustedPipeline().Scan(model.EmailScanRequest{
		MessageID:   "b1",
		Subject:     "URGENT wire transfer",
		BodyPreview: "This is urgent. Please complete the change of bank account and wire transfer immediately.",
	})
	if !hasCategory(v, "social-engineering") || v.Status != "malicious" {
		t.Fatalf("expected high BEC verdict, got %+v", v)
	}
}

func TestSocialEngineeringSingleWordNoFlag(t *testing.T) {
	v := trustedPipeline().Scan(model.EmailScanRequest{
		MessageID:   "b2",
		BodyPreview: "Please review the attached notes now.",
	})
	if hasCategory(v, "social-engineering") {
		t.Fatalf("single weak word should not flag, got %+v", v)
	}
}

func TestQuishingDecodesImageQR(t *testing.T) {
	m, err := qr.Encode("http://xn--phishy.example/login", 3)
	if err != nil {
		t.Fatal(err)
	}
	png, err := qr.RenderPNG(m, 4, 4)
	if err != nil {
		t.Fatal(err)
	}
	v := trustedPipeline().Scan(model.EmailScanRequest{
		MessageID:   "q1",
		Subject:     "Don't lose your access",
		BodyPreview: "Scan the QR code to verify your account.",
		Attachments: []model.Artifact{{
			Name:        "qr.png",
			ContentType: "image/png",
			DataBase64:  base64.StdEncoding.EncodeToString(png),
		}},
	})
	if !hasCategory(v, "quishing") {
		t.Fatalf("expected quishing finding, got %+v", v)
	}
	found := false
	for _, f := range v.Findings {
		if f.Category == "quishing" && strings.Contains(f.Summary, "xn--phishy.example") {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected decoded URL in finding, got %+v", v.Findings)
	}
}

func TestQuishingLureWithoutDecodableURL(t *testing.T) {
	// A 1x1 white image: no QR, but lure language + image present.
	white := "iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR42mP8z8BQDwAEhQGAhKmMIQAAAABJRU5ErkJggg=="
	raw, _ := base64.StdEncoding.DecodeString(white)
	v := trustedPipeline().Scan(model.EmailScanRequest{
		MessageID:   "q2",
		BodyPreview: "請掃描 QR 圖片以登入",
		Attachments: []model.Artifact{{
			Name:        "image.png",
			ContentType: "image/png",
			DataBase64:  base64.StdEncoding.EncodeToString(raw),
		}},
	})
	if !hasCategory(v, "quishing") {
		t.Fatalf("expected quishing lure finding, got %+v", v)
	}
}

func TestDLPOutboundCardNumber(t *testing.T) {
	v := trustedPipeline().Scan(model.EmailScanRequest{
		MessageID:   "d1",
		Direction:   "outbound",
		BodyPreview: "Here is my card 4111 1111 1111 1111 for the payment.",
	})
	if !hasCategory(v, "dlp") || v.Status != "malicious" {
		t.Fatalf("expected DLP high finding, got %+v", v)
	}
}

func TestDLPInboundSkipped(t *testing.T) {
	v := trustedPipeline().Scan(model.EmailScanRequest{
		MessageID:   "d2",
		Direction:   "inbound",
		BodyPreview: "card 4111 1111 1111 1111",
	})
	if hasCategory(v, "dlp") {
		t.Fatalf("DLP must not run on inbound, got %+v", v)
	}
}
