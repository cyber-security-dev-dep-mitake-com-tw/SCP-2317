package scanner

import (
	"regexp"
	"strings"

	"github.com/example/scp2317-agent/internal/model"
)

var (
	// Candidate 13-16 digit sequences (spaces/hyphens allowed) for card checks.
	cardCandidate = regexp.MustCompile(`\b(?:\d[ -]?){13,16}\b`)
	// Taiwan national ID: 1 letter + [1-2] + 8 digits.
	twNationalID = regexp.MustCompile(`\b[A-Za-z][12]\d{8}\b`)
	privateKey   = regexp.MustCompile(`-----BEGIN (?:RSA |EC |OPENSSH )?PRIVATE KEY-----`)
	awsKey       = regexp.MustCompile(`\bAKIA[0-9A-Z]{16}\b`)
	passwordLine = regexp.MustCompile(`(?i)\b(?:password|passwd|pwd|密碼)\b\s*[:=]\s*\S+`)
)

func luhnValid(digits string) bool {
	sum, alt := 0, false
	for i := len(digits) - 1; i >= 0; i-- {
		c := digits[i]
		if c < '0' || c > '9' {
			return false
		}
		d := int(c - '0')
		if alt {
			d *= 2
			if d > 9 {
				d -= 9
			}
		}
		sum += d
		alt = !alt
	}
	return sum%10 == 0 && len(digits) >= 13
}

// dlpEngine runs only on outbound mail and flags likely data exfiltration:
// Luhn-valid card numbers, national IDs, private keys, cloud keys, and
// plaintext passwords. A hit is a send-block advisory for the client.
func (p *Pipeline) dlpEngine(r model.EmailScanRequest) []model.Finding {
	text := r.Subject + "\n" + r.BodyPreview
	var out []model.Finding
	add := func(sev, msg string) {
		out = append(out, model.Finding{
			Engine: "dlp", Category: "dlp", Severity: sev, Summary: msg,
		})
	}

	for _, m := range cardCandidate.FindAllString(text, -1) {
		digits := strings.NewReplacer(" ", "", "-", "").Replace(m)
		if luhnValid(digits) {
			add("high", "Outbound message contains a Luhn-valid card number")
			break
		}
	}
	if twNationalID.MatchString(text) {
		add("high", "Outbound message contains a Taiwan national ID number")
	}
	if privateKey.MatchString(text) {
		add("high", "Outbound message contains a private key block")
	}
	if awsKey.MatchString(text) {
		add("high", "Outbound message contains an AWS access key ID")
	}
	if passwordLine.MatchString(text) {
		add("medium", "Outbound message contains a plaintext password")
	}
	return out
}
