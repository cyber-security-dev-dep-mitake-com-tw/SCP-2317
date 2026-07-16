package scanner

import (
	"strings"

	"github.com/example/scp2317-agent/internal/model"
)

// knownSaaSHosts are legitimate platforms frequently abused to host phishing
// forms/files ("living off trusted SaaS"). A link to one of these combined with
// credential-harvest language is suspicious even though the domain is valid.
var knownSaaSHosts = []string{
	"forms.gle", "docs.google.com/forms", "forms.office.com",
	"sharepoint.com", "notion.so", "airtable.com", "typeform.com",
	"formstack.com", "jotform.com", "wufoo.com", "surveymonkey.com",
}

// urlContextEngine flags two things the plain URL engine misses:
//   - a link to a legitimate SaaS form/file host used as a credential-harvest
//     landing page (paired with lure language);
//   - an in-body URL whose host is a look-alike/typosquat of a trusted domain.
func (p *Pipeline) urlContextEngine(r model.EmailScanRequest, urls []string) []model.Finding {
	var out []model.Finding
	lure := hasCredentialLure(r)

	for _, u := range urls {
		host := urlHost(u)
		if host == "" {
			continue
		}
		lu := strings.ToLower(u)

		if lure {
			for _, s := range knownSaaSHosts {
				if strings.Contains(lu, s) {
					out = append(out, model.Finding{
						Engine: "url-context", Category: "url", Severity: "medium",
						Summary: "Credential lure links to a form on trusted SaaS host: " + host,
					})
					break
				}
			}
		}

		folded := registrable(foldConfusables(host))
		for _, td := range p.policy.TrustedDomains {
			rt := registrable(strings.ToLower(strings.TrimSpace(td)))
			if rt == "" || rt == folded {
				continue
			}
			if d := levenshtein(folded, rt); d > 0 && d <= 2 {
				out = append(out, model.Finding{
					Engine: "url-context", Category: "spoofing", Severity: "high",
					Summary: "Link host " + host + " closely resembles trusted domain " + rt,
				})
				break
			}
		}
	}
	return out
}

func hasCredentialLure(r model.EmailScanRequest) bool {
	text := strings.ToLower(r.Subject + "\n" + r.BodyPreview)
	for _, ph := range intentLexicon["credential"] {
		if strings.Contains(text, ph) {
			return true
		}
	}
	return false
}

// routeForensicsEngine performs lightweight Received-chain analysis. Without a
// live ASN/geo database it cannot geolocate relays, but it can flag an
// implausible number of hops and relays that advertise no reverse DNS.
func (p *Pipeline) routeForensicsEngine(r model.EmailScanRequest) []model.Finding {
	if len(r.ReceivedChain) == 0 {
		return nil
	}
	var out []model.Finding
	if len(r.ReceivedChain) > 8 {
		out = append(out, model.Finding{
			Engine: "route-forensics", Category: "route-forensics", Severity: "low",
			Summary: "Message traversed an unusually long relay chain",
		})
	}
	for _, hop := range r.ReceivedChain {
		l := strings.ToLower(hop)
		// "from [1.2.3.4]" with no hostname before it is a bare-IP relay.
		if strings.Contains(l, "from [") || strings.Contains(l, "unknown [") {
			out = append(out, model.Finding{
				Engine: "route-forensics", Category: "route-forensics", Severity: "low",
				Summary: "Relay hop with no reverse-DNS hostname (bare IP)",
			})
			break
		}
	}
	return out
}
