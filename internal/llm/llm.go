// Package llm is an OPTIONAL, disabled-by-default analyzer that can escalate
// ambiguous email to an external LLM for BEC / phishing intent scoring. It is
// never invoked unless explicitly configured, so the default scanner stays
// fully offline. OpenAI, Grok (xAI), and Mistral all expose an OpenAI-style
// /chat/completions endpoint; this client targets that shared shape.
package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"
)

// Format selects provider defaults (base URL + model) for the shared
// OpenAI-compatible chat-completions API.
type Format string

const (
	FormatOpenAI  Format = "openai"
	FormatGrok    Format = "grok"
	FormatMistral Format = "mistral"
)

type Config struct {
	Format  Format
	BaseURL string
	APIKey  string
	Model   string
	Timeout time.Duration
}

func defaultsFor(f Format) (baseURL, model string) {
	switch f {
	case FormatGrok:
		return "https://api.x.ai/v1", "grok-2-latest"
	case FormatMistral:
		return "https://api.mistral.ai/v1", "mistral-small-latest"
	default:
		return "https://api.openai.com/v1", "gpt-4o-mini"
	}
}

// FromEnv builds a Config from SCP2317_LLM_* variables, or returns ok=false
// when analysis is not configured (the normal, offline case).
//
//	SCP2317_LLM_FORMAT   openai | grok | mistral
//	SCP2317_LLM_API_KEY  provider API key (required to enable)
//	SCP2317_LLM_BASE_URL optional base URL override
//	SCP2317_LLM_MODEL    optional model override
func FromEnv() (Config, bool) {
	key := os.Getenv("SCP2317_LLM_API_KEY")
	if strings.TrimSpace(key) == "" {
		return Config{}, false
	}
	f := Format(strings.ToLower(strings.TrimSpace(os.Getenv("SCP2317_LLM_FORMAT"))))
	if f != FormatGrok && f != FormatMistral {
		f = FormatOpenAI
	}
	baseURL, model := defaultsFor(f)
	if v := os.Getenv("SCP2317_LLM_BASE_URL"); v != "" {
		baseURL = v
	}
	if v := os.Getenv("SCP2317_LLM_MODEL"); v != "" {
		model = v
	}
	return Config{Format: f, BaseURL: baseURL, APIKey: key, Model: model, Timeout: 8 * time.Second}, true
}

// Assessment is the normalized verdict returned by the model.
type Assessment struct {
	Risk   string `json:"risk"` // low | medium | high
	Reason string `json:"reason"`
	IsBEC  bool   `json:"is_bec"`
}

type Analyzer struct {
	cfg    Config
	client *http.Client
}

func New(cfg Config) *Analyzer {
	if cfg.Timeout == 0 {
		cfg.Timeout = 8 * time.Second
	}
	if cfg.BaseURL == "" || cfg.Model == "" {
		b, m := defaultsFor(cfg.Format)
		if cfg.BaseURL == "" {
			cfg.BaseURL = b
		}
		if cfg.Model == "" {
			cfg.Model = m
		}
	}
	return &Analyzer{cfg: cfg, client: &http.Client{Timeout: cfg.Timeout}}
}

const systemPrompt = "You are an email-security classifier. Given an email, respond ONLY with compact JSON " +
	`{"risk":"low|medium|high","is_bec":true|false,"reason":"..."} assessing business-email-compromise / phishing intent.`

type chatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type chatRequest struct {
	Model       string        `json:"model"`
	Messages    []chatMessage `json:"messages"`
	Temperature float64       `json:"temperature"`
}

type chatResponse struct {
	Choices []struct {
		Message chatMessage `json:"message"`
	} `json:"choices"`
}

// BuildRequest constructs the HTTP request (exposed for testing without network).
func (a *Analyzer) BuildRequest(ctx context.Context, email string) (*http.Request, error) {
	body, err := json.Marshal(chatRequest{
		Model:       a.cfg.Model,
		Temperature: 0,
		Messages: []chatMessage{
			{Role: "system", Content: systemPrompt},
			{Role: "user", Content: email},
		},
	})
	if err != nil {
		return nil, err
	}
	url := strings.TrimRight(a.cfg.BaseURL, "/") + "/chat/completions"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+a.cfg.APIKey)
	return req, nil
}

// parseAssessment extracts the normalized verdict from a chat response.
func parseAssessment(raw []byte) (Assessment, error) {
	var cr chatResponse
	if err := json.Unmarshal(raw, &cr); err != nil {
		return Assessment{}, err
	}
	if len(cr.Choices) == 0 {
		return Assessment{}, fmt.Errorf("llm: empty choices")
	}
	content := cr.Choices[0].Message.Content
	// The model may wrap JSON in prose or code fences; extract the object.
	if i := strings.Index(content, "{"); i >= 0 {
		if j := strings.LastIndex(content, "}"); j >= i {
			content = content[i : j+1]
		}
	}
	var as Assessment
	if err := json.Unmarshal([]byte(content), &as); err != nil {
		return Assessment{}, err
	}
	as.Risk = strings.ToLower(strings.TrimSpace(as.Risk))
	return as, nil
}

// Analyze calls the configured provider and returns a normalized assessment.
func (a *Analyzer) Analyze(ctx context.Context, email string) (Assessment, error) {
	req, err := a.BuildRequest(ctx, email)
	if err != nil {
		return Assessment{}, err
	}
	resp, err := a.client.Do(req)
	if err != nil {
		return Assessment{}, err
	}
	defer resp.Body.Close()
	buf := new(bytes.Buffer)
	if _, err := buf.ReadFrom(resp.Body); err != nil {
		return Assessment{}, err
	}
	if resp.StatusCode != http.StatusOK {
		return Assessment{}, fmt.Errorf("llm: provider status %d", resp.StatusCode)
	}
	return parseAssessment(buf.Bytes())
}
