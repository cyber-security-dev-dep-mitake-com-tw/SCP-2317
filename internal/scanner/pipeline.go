package scanner

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"sync"

	"github.com/example/scp2317-agent/internal/model"
)

var urlInText = regexp.MustCompile(`(?i)\bhttps?://[^\s<>"')]+`)

type Policy struct {
	HighRiskExtensions   []string `json:"high_risk_extensions"`
	InlineSizeLimitBytes int      `json:"inline_size_limit_bytes"`
	MaxURLs              int      `json:"max_urls"`
	MaxAttachments       int      `json:"max_attachments"`
}

func DefaultPolicy() Policy {
	return Policy{
		HighRiskExtensions:   []string{".exe", ".dll", ".js", ".vbs", ".lnk", ".iso", ".scr", ".bat", ".cmd"},
		InlineSizeLimitBytes: 10 * 1024 * 1024,
		MaxURLs:              100,
		MaxAttachments:       50,
	}
}

type Pipeline struct {
	policy Policy
}

var (
	loadedPolicy Policy
	policyOnce   sync.Once
)

func loadPolicy() Policy {
	policyOnce.Do(func() {
		loadedPolicy = DefaultPolicy()
		_, file, _, ok := runtime.Caller(0)
		if !ok {
			return
		}
		candidates := []string{
			filepath.Join(filepath.Dir(file), "..", "..", "rules", "attachment_policy.json"),
			"rules/attachment_policy.json",
		}
		for _, path := range candidates {
			data, err := os.ReadFile(path)
			if err != nil {
				continue
			}
			var p Policy
			if json.Unmarshal(data, &p) == nil && len(p.HighRiskExtensions) > 0 {
				if p.InlineSizeLimitBytes <= 0 {
					p.InlineSizeLimitBytes = loadedPolicy.InlineSizeLimitBytes
				}
				if p.MaxURLs <= 0 {
					p.MaxURLs = loadedPolicy.MaxURLs
				}
				if p.MaxAttachments <= 0 {
					p.MaxAttachments = loadedPolicy.MaxAttachments
				}
				loadedPolicy = p
				return
			}
		}
	})
	return loadedPolicy
}

func NewPipeline() *Pipeline {
	return &Pipeline{policy: loadPolicy()}
}

func NewPipelineWithPolicy(p Policy) *Pipeline {
	return &Pipeline{policy: p}
}

// Validate checks request bounds before scanning. Invalid base64 is reported here.
func (p *Pipeline) Validate(r model.EmailScanRequest) error {
	if strings.TrimSpace(r.MessageID) == "" {
		return fmt.Errorf("message_id is required")
	}
	if len(r.URLs) > p.policy.MaxURLs {
		return fmt.Errorf("urls exceeds max of %d", p.policy.MaxURLs)
	}
	if len(r.Attachments) > p.policy.MaxAttachments {
		return fmt.Errorf("attachments exceeds max of %d", p.policy.MaxAttachments)
	}
	for _, a := range r.Attachments {
		if a.DataBase64 == "" {
			continue
		}
		if _, err := base64.StdEncoding.DecodeString(a.DataBase64); err != nil {
			return fmt.Errorf("attachment %q has invalid data_base64", a.Name)
		}
	}
	return nil
}

// ExtractURLs returns unique URLs from the explicit list and body_preview text.
func ExtractURLs(explicit []string, body string) []string {
	seen := map[string]struct{}{}
	var out []string
	add := func(u string) {
		u = strings.TrimRight(u, ".,;:)")
		if u == "" {
			return
		}
		key := strings.ToLower(u)
		if _, ok := seen[key]; ok {
			return
		}
		seen[key] = struct{}{}
		out = append(out, u)
	}
	for _, u := range explicit {
		add(u)
	}
	for _, m := range urlInText.FindAllString(body, -1) {
		add(m)
	}
	return out
}

// Scan performs bounded, non-executing checks. A production dynamic analyzer
// must be placed behind the separate, disposable VM worker contract.
func (p *Pipeline) Scan(r model.EmailScanRequest) model.Verdict {
	findings, score := []model.Finding{}, 0
	urls := ExtractURLs(r.URLs, r.BodyPreview)
	for _, u := range urls {
		lu := strings.ToLower(u)
		if strings.HasPrefix(lu, "http://") || strings.Contains(lu, "xn--") {
			findings = append(findings, model.Finding{
				Engine:   "url-heuristics",
				Severity: "medium",
				Summary:  "URL uses cleartext HTTP or punycode",
			})
			score += 25
		}
	}
	for _, a := range r.Attachments {
		name := strings.ToLower(a.Name)
		for _, ext := range p.policy.HighRiskExtensions {
			if strings.HasSuffix(name, strings.ToLower(ext)) {
				findings = append(findings, model.Finding{
					Engine:   "attachment-policy",
					Severity: "high",
					Summary:  "High-risk attachment type: " + ext,
				})
				score += 55
			}
		}
		if a.DataBase64 != "" {
			if b, err := base64.StdEncoding.DecodeString(a.DataBase64); err == nil && len(b) > p.policy.InlineSizeLimitBytes {
				findings = append(findings, model.Finding{
					Engine:   "size-policy",
					Severity: "low",
					Summary:  "Attachment exceeds inline scan limit",
				})
				score += 10
			}
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
