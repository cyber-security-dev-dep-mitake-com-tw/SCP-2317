// Package easm is a small External Attack Surface Monitor for domains you own.
// It checks that a domain still publishes SPF and an enforcing DMARC policy —
// the "authentication drift" monitoring described in docs/ref.txt. It uses live
// DNS (the companion app is allowed outbound), and its parsing is pure/testable.
package easm

import (
	"context"
	"net"
	"strings"
	"time"
)

type Result struct {
	Domain     string   `json:"domain"`
	HasSPF     bool     `json:"has_spf"`
	SPFRecord  string   `json:"spf_record,omitempty"`
	DMARCPolicy string  `json:"dmarc_policy,omitempty"` // none | quarantine | reject | ""
	HasMX      bool     `json:"has_mx"`
	Findings   []string `json:"findings"`
}

// ParseSPF returns the SPF record from a set of TXT records, or "".
func ParseSPF(txts []string) string {
	for _, t := range txts {
		if strings.HasPrefix(strings.ToLower(strings.TrimSpace(t)), "v=spf1") {
			return strings.TrimSpace(t)
		}
	}
	return ""
}

// ParseDMARCPolicy extracts the p= policy from _dmarc TXT records, or "".
func ParseDMARCPolicy(txts []string) string {
	for _, t := range txts {
		lt := strings.ToLower(t)
		if !strings.Contains(lt, "v=dmarc1") {
			continue
		}
		for _, part := range strings.Split(lt, ";") {
			part = strings.TrimSpace(part)
			if strings.HasPrefix(part, "p=") {
				return strings.TrimSpace(strings.TrimPrefix(part, "p="))
			}
		}
	}
	return ""
}

// EvaluateFindings derives drift warnings from parsed records.
func EvaluateFindings(r Result) []string {
	var f []string
	if !r.HasSPF {
		f = append(f, "no SPF record published")
	}
	switch r.DMARCPolicy {
	case "":
		f = append(f, "no DMARC record published")
	case "none":
		f = append(f, "DMARC policy is p=none (monitor only, not enforcing)")
	}
	return f
}

type Resolver interface {
	LookupTXT(ctx context.Context, name string) ([]string, error)
	LookupMX(ctx context.Context, name string) ([]*net.MX, error)
}

// Check runs a live DNS assessment of a domain.
func Check(ctx context.Context, r Resolver, domain string) Result {
	domain = strings.TrimSpace(strings.ToLower(domain))
	res := Result{Domain: domain}
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	if txts, err := r.LookupTXT(ctx, domain); err == nil {
		if spf := ParseSPF(txts); spf != "" {
			res.HasSPF, res.SPFRecord = true, spf
		}
	}
	if txts, err := r.LookupTXT(ctx, "_dmarc."+domain); err == nil {
		res.DMARCPolicy = ParseDMARCPolicy(txts)
	}
	if mxs, err := r.LookupMX(ctx, domain); err == nil {
		res.HasMX = len(mxs) > 0
	}
	res.Findings = EvaluateFindings(res)
	return res
}

// DefaultResolver uses the system resolver.
func DefaultResolver() Resolver { return netResolver{} }

type netResolver struct{}

func (netResolver) LookupTXT(ctx context.Context, name string) ([]string, error) {
	return net.DefaultResolver.LookupTXT(ctx, name)
}
func (netResolver) LookupMX(ctx context.Context, name string) ([]*net.MX, error) {
	return net.DefaultResolver.LookupMX(ctx, name)
}
