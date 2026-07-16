package scanner

import (
	"encoding/base64"
	"strings"

	"github.com/example/scp2317-agent/internal/model"
	"github.com/example/scp2317-agent/internal/qr"
)

func isImageAttachment(a model.Artifact) bool {
	ct := strings.ToLower(a.ContentType)
	if strings.HasPrefix(ct, "image/") {
		return true
	}
	name := strings.ToLower(a.Name)
	for _, ext := range []string{".png", ".jpg", ".jpeg", ".gif"} {
		if strings.HasSuffix(name, ext) {
			return true
		}
	}
	return false
}

// quishingEngine decodes QR codes hidden in inline images and flags the URLs
// they carry (quishing). It decodes images the server received (data_base64),
// honors URLs a client decoded client-side (QRURLs), and flags an image + QR
// lure-language combination even when no URL could be extracted.
func (p *Pipeline) quishingEngine(r model.EmailScanRequest) []model.Finding {
	var out []model.Finding
	seen := map[string]bool{}

	emitURL := func(u, source string) {
		if u == "" || seen[u] {
			return
		}
		seen[u] = true
		lu := strings.ToLower(u)
		sev := "medium"
		if strings.HasPrefix(lu, "http://") || strings.Contains(lu, "xn--") || hasNonASCII(u) {
			sev = "high"
		}
		out = append(out, model.Finding{
			Engine:   "quishing",
			Category: "quishing",
			Severity: sev,
			Summary:  "QR code (" + source + ") encodes URL: " + u,
		})
	}

	// URLs a client already decoded (e.g. from a rasterized PDF page).
	for _, u := range r.QRURLs {
		emitURL(strings.TrimSpace(u), "client-decoded")
	}

	// Server-side decode of inline image attachments.
	decodedAny := false
	haveImage := false
	for _, a := range r.Attachments {
		if !isImageAttachment(a) {
			continue
		}
		haveImage = true
		if a.DataBase64 == "" {
			continue
		}
		data, err := base64.StdEncoding.DecodeString(a.DataBase64)
		if err != nil {
			continue
		}
		if text, err := qr.DecodeImageBytes(data); err == nil {
			decodedAny = true
			for _, u := range ExtractURLs(nil, text) {
				emitURL(u, "image "+a.Name)
			}
			// A QR that encodes a URL directly (no scheme prefix captured) still
			// matters; surface the raw payload if it looks like a host.
			if len(ExtractURLs(nil, text)) == 0 && looksLikeHost(text) {
				emitURL("http://"+strings.TrimSpace(text), "image "+a.Name)
			}
		}
	}

	// Lure language + an image present, but no decodable URL: still suspicious.
	if haveImage && !decodedAny && len(out) == 0 && hasQRLure(r) {
		out = append(out, model.Finding{
			Engine:   "quishing",
			Category: "quishing",
			Severity: "medium",
			Summary:  "Image attachment plus QR-scan lure language (possible quishing)",
		})
	}
	return out
}

func looksLikeHost(s string) bool {
	s = strings.TrimSpace(strings.ToLower(s))
	return strings.Contains(s, ".") && !strings.ContainsAny(s, " \t\n") && len(s) <= 253
}

func hasQRLure(r model.EmailScanRequest) bool {
	text := strings.ToLower(r.Subject + "\n" + r.BodyPreview)
	for _, ph := range []string{"qr", "scan", "掃描", "下載圖片", "掃碼"} {
		if strings.Contains(text, ph) {
			return true
		}
	}
	return false
}
