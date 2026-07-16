package scanner

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/example/scp2317-agent/internal/llm"
	"github.com/example/scp2317-agent/internal/model"
)

var urlInText = regexp.MustCompile(`(?i)\bhttps?://[^\s<>"')]+`)

type Policy struct {
	HighRiskExtensions   []string `json:"high_risk_extensions"`
	InlineSizeLimitBytes int      `json:"inline_size_limit_bytes"`
	MaxURLs              int      `json:"max_urls"`
	MaxAttachments       int      `json:"max_attachments"`
	// TrustedDomains are the org's own / known-good sending domains. They anchor
	// look-alike and display-name-spoof detection.
	TrustedDomains []string `json:"trusted_domains,omitempty"`
}

func DefaultPolicy() Policy {
	return Policy{
		HighRiskExtensions:   []string{".exe", ".dll", ".js", ".vbs", ".lnk", ".iso", ".scr", ".bat", ".cmd"},
		InlineSizeLimitBytes: 10 * 1024 * 1024,
		MaxURLs:              100,
		MaxAttachments:       50,
		TrustedDomains:       []string{},
	}
}

type Pipeline struct {
	policy Policy
	// analyzer is nil unless an LLM provider is explicitly configured via env.
	// When set, it augments (never replaces) the offline heuristics.
	analyzer *llm.Analyzer
}

// SetAnalyzer attaches an optional LLM analyzer.
func (p *Pipeline) SetAnalyzer(a *llm.Analyzer) { p.analyzer = a }

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
	p := &Pipeline{policy: loadPolicy()}
	if cfg, ok := llm.FromEnv(); ok {
		p.analyzer = llm.New(cfg)
	}
	return p
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

// severityScore maps a severity label to its default score weight.
func severityScore(sev string) int {
	switch sev {
	case "high":
		return 55
	case "medium":
		return 25
	case "low":
		return 10
	default:
		return 0
	}
}

// Scan performs bounded, non-executing checks across every heuristic engine.
// A production dynamic analyzer must be placed behind the separate, disposable
// VM worker contract; nothing here executes attachment content.
func (p *Pipeline) Scan(r model.EmailScanRequest) model.Verdict {
	findings := []model.Finding{}
	urls := ExtractURLs(r.URLs, r.BodyPreview)

	// Inbound-facing engines.
	findings = append(findings, p.urlEngine(urls)...)
	findings = append(findings, p.attachmentEngine(r)...)
	findings = append(findings, p.authEngine(r)...)
	findings = append(findings, p.spoofingEngine(r)...)
	findings = append(findings, p.socialEngine(r)...)
	findings = append(findings, p.quishingEngine(r)...)

	// Outbound-only engine (data loss prevention).
	if strings.EqualFold(r.Direction, "outbound") {
		findings = append(findings, p.dlpEngine(r)...)
	}

	// Optional LLM augmentation (only when explicitly configured).
	findings = append(findings, p.llmEngine(r, findings)...)

	score := 0
	for _, f := range findings {
		score += severityScore(f.Severity)
	}
	if score > 100 {
		score = 100
	}
	status := "clean"
	if score >= 50 {
		status = "malicious"
	} else if score > 0 {
		status = "suspicious"
	}
	return model.Verdict{Status: status, Score: score, Findings: findings}
}

// llmEngine augments the heuristics with an optional external LLM assessment.
// It is a no-op unless an analyzer was configured, and it never downgrades an
// existing verdict — it can only add a finding.
func (p *Pipeline) llmEngine(r model.EmailScanRequest, existing []model.Finding) []model.Finding {
	if p.analyzer == nil {
		return nil
	}
	email := "From: " + r.SenderDisplay + " <" + r.Sender + ">\nSubject: " + r.Subject + "\n\n" + r.BodyPreview
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()
	as, err := p.analyzer.Analyze(ctx, email)
	if err != nil {
		return nil // fail open: offline heuristics already ran
	}
	if as.Risk != "high" && !as.IsBEC {
		return nil
	}
	sev := "medium"
	if as.Risk == "high" {
		sev = "high"
	}
	summary := "LLM flagged likely BEC/phishing intent"
	if as.Reason != "" {
		summary += ": " + as.Reason
	}
	return []model.Finding{{
		Engine: "llm-analyzer", Category: "social-engineering", Severity: sev, Summary: summary,
	}}
}

func (p *Pipeline) urlEngine(urls []string) []model.Finding {
	var out []model.Finding
	for _, u := range urls {
		lu := strings.ToLower(u)
		if strings.HasPrefix(lu, "http://") || strings.Contains(lu, "xn--") {
			out = append(out, model.Finding{
				Engine:   "url-heuristics",
				Category: "url",
				Severity: "medium",
				Summary:  "URL uses cleartext HTTP or punycode: " + u,
			})
		}
	}
	return out
}

func (p *Pipeline) attachmentEngine(r model.EmailScanRequest) []model.Finding {
	var out []model.Finding
	for _, a := range r.Attachments {
		name := strings.ToLower(a.Name)
		for _, ext := range p.policy.HighRiskExtensions {
			if strings.HasSuffix(name, strings.ToLower(ext)) {
				out = append(out, model.Finding{
					Engine:   "attachment-policy",
					Category: "attachment",
					Severity: "high",
					Summary:  "High-risk attachment type: " + ext,
				})
			}
		}
		if a.DataBase64 != "" {
			if b, err := base64.StdEncoding.DecodeString(a.DataBase64); err == nil && len(b) > p.policy.InlineSizeLimitBytes {
				out = append(out, model.Finding{
					Engine:   "size-policy",
					Category: "attachment",
					Severity: "low",
					Summary:  "Attachment exceeds inline scan limit",
				})
			}
		}
	}
	return out
}
