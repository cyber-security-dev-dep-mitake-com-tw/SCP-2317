package scanner

import (
	"strings"
	"unicode"

	"github.com/example/scp2317-agent/internal/model"
)

// domainOf returns the lowercased domain of an email address, or "".
func domainOf(addr string) string {
	addr = strings.TrimSpace(strings.ToLower(addr))
	addr = strings.Trim(addr, "<>")
	at := strings.LastIndex(addr, "@")
	if at < 0 || at == len(addr)-1 {
		return ""
	}
	return addr[at+1:]
}

// authEngine surfaces SPF/DKIM/DMARC results the client read from the message.
// The endpoint never performs DNS itself; it interprets Authentication-Results.
func (p *Pipeline) authEngine(r model.EmailScanRequest) []model.Finding {
	ar := strings.ToLower(r.AuthResults)
	if strings.TrimSpace(ar) == "" {
		return nil
	}
	var out []model.Finding
	check := func(mech, sev, msg string) {
		if strings.Contains(ar, mech+"=fail") || strings.Contains(ar, mech+"=softfail") {
			out = append(out, model.Finding{
				Engine: "authentication", Category: "authentication", Severity: sev, Summary: msg,
			})
		}
	}
	check("dmarc", "high", "DMARC authentication failed")
	check("spf", "medium", "SPF authentication failed")
	check("dkim", "medium", "DKIM signature failed")
	return out
}

// asciiLookalike folds common Unicode confusables to ASCII so a homograph
// domain collapses onto the brand it imitates.
var confusables = map[rune]rune{
	'а': 'a', 'е': 'e', 'о': 'o', 'р': 'p', 'с': 'c', 'х': 'x', 'ѕ': 's', // Cyrillic
	'і': 'i', 'ј': 'j', 'ԁ': 'd', 'ɡ': 'g', 'ｌ': 'l', '০': '0', 'Ο': 'O',
}

func hasNonASCII(s string) bool {
	for _, r := range s {
		if r > unicode.MaxASCII {
			return true
		}
	}
	return false
}

func foldConfusables(s string) string {
	var b strings.Builder
	for _, r := range s {
		if c, ok := confusables[r]; ok {
			b.WriteRune(c)
		} else {
			b.WriteRune(r)
		}
	}
	return b.String()
}

// levenshtein is a small edit-distance used for typosquat detection.
func levenshtein(a, b string) int {
	if a == b {
		return 0
	}
	la, lb := len(a), len(b)
	prev := make([]int, lb+1)
	for j := 0; j <= lb; j++ {
		prev[j] = j
	}
	for i := 1; i <= la; i++ {
		cur := make([]int, lb+1)
		cur[0] = i
		for j := 1; j <= lb; j++ {
			cost := 1
			if a[i-1] == b[j-1] {
				cost = 0
			}
			cur[j] = min3(prev[j]+1, cur[j-1]+1, prev[j-1]+cost)
		}
		prev = cur
	}
	return prev[lb]
}

func min3(a, b, c int) int {
	if b < a {
		a = b
	}
	if c < a {
		a = c
	}
	return a
}

// registrable trims a domain to its last two labels for comparison.
func registrable(d string) string {
	parts := strings.Split(d, ".")
	if len(parts) < 2 {
		return d
	}
	return strings.Join(parts[len(parts)-2:], ".")
}

// spoofingEngine detects sender-identity abuse: homograph domains, From vs
// Reply-To mismatch, display-name brand impersonation, and look-alike domains
// close to a trusted domain.
func (p *Pipeline) spoofingEngine(r model.EmailScanRequest) []model.Finding {
	var out []model.Finding
	from := domainOf(r.Sender)
	if from == "" {
		return out
	}

	// Homograph / punycode in the sender domain itself.
	if strings.Contains(from, "xn--") || hasNonASCII(from) {
		out = append(out, model.Finding{
			Engine: "spoofing", Category: "spoofing", Severity: "high",
			Summary: "Sender domain uses punycode or non-ASCII look-alike characters",
		})
	}

	// From vs Reply-To domain mismatch.
	if rt := domainOf(r.ReplyTo); rt != "" && rt != from {
		out = append(out, model.Finding{
			Engine: "spoofing", Category: "spoofing", Severity: "medium",
			Summary: "Reply-To domain (" + rt + ") differs from sender domain (" + from + ")",
		})
	}

	// Display-name impersonation: name claims a trusted brand but the domain is
	// not that brand.
	display := strings.ToLower(r.SenderDisplay)
	for _, td := range p.policy.TrustedDomains {
		td = strings.ToLower(strings.TrimSpace(td))
		if td == "" {
			continue
		}
		brand := strings.SplitN(td, ".", 2)[0]
		if brand != "" && strings.Contains(display, brand) && registrable(from) != registrable(td) {
			out = append(out, model.Finding{
				Engine: "spoofing", Category: "spoofing", Severity: "high",
				Summary: "Display name references \"" + brand + "\" but sender domain is " + from,
			})
			break
		}
	}

	// Look-alike / typosquat of a trusted domain (fold confusables first).
	folded := registrable(foldConfusables(from))
	for _, td := range p.policy.TrustedDomains {
		rt := registrable(strings.ToLower(strings.TrimSpace(td)))
		if rt == "" || rt == folded {
			continue
		}
		if d := levenshtein(folded, rt); d > 0 && d <= 2 {
			out = append(out, model.Finding{
				Engine: "spoofing", Category: "spoofing", Severity: "high",
				Summary: "Sender domain " + from + " closely resembles trusted domain " + rt,
			})
			break
		}
	}
	return out
}
