package scanner

import (
	"encoding/base64"
	"strings"

	"github.com/example/scp2317-agent/internal/model"
)

type Pipeline struct{}

func NewPipeline() *Pipeline { return &Pipeline{} }

// Scan performs bounded, non-executing checks. A production dynamic analyzer
// must be placed behind the separate, disposable VM worker contract.
func (p *Pipeline) Scan(r model.EmailScanRequest) model.Verdict {
	findings, score := []model.Finding{}, 0
	for _, u := range r.URLs {
		lu := strings.ToLower(u)
		if strings.HasPrefix(lu, "http://") || strings.Contains(lu, "xn--") {
			findings = append(findings, model.Finding{"url-heuristics", "medium", "URL uses cleartext HTTP or punycode"})
			score += 25
		}
	}
	for _, a := range r.Attachments {
		name := strings.ToLower(a.Name)
		for _, ext := range []string{".exe", ".dll", ".js", ".vbs", ".lnk", ".iso"} {
			if strings.HasSuffix(name, ext) {
				findings = append(findings, model.Finding{"attachment-policy", "high", "High-risk attachment type: " + ext})
				score += 55
			}
		}
		if b, err := base64.StdEncoding.DecodeString(a.DataBase64); err == nil && len(b) > 10*1024*1024 {
			findings = append(findings, model.Finding{"size-policy", "low", "Attachment exceeds inline scan limit"})
			score += 10
		}
	}
	status := "clean"
	if score >= 50 {
		status = "malicious"
	} else if score > 0 {
		status = "suspicious"
	}
	if score > 100 {
		score = 100
	}
	return model.Verdict{Status: status, Score: score, Findings: findings}
}
