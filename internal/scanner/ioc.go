package scanner

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"

	"github.com/example/scp2317-agent/internal/model"
)

// IOCList is a local, offline threat-intelligence feed. It stands in for the
// signature / IP-reputation / STIX-TAXII feeds described in docs/ref.txt: an
// endpoint cannot reach a live global intel network, but it can match against a
// list distributed to it. Extend or replace rules/ioc.json operationally.
type IOCList struct {
	BadSHA256       []string `json:"bad_sha256"`
	BadDomains      []string `json:"bad_domains"`
	BadURLSubstring []string `json:"bad_url_substrings"`
}

var (
	loadedIOC IOCList
	iocOnce   sync.Once
)

func loadIOC() IOCList {
	iocOnce.Do(func() {
		_, file, _, ok := runtime.Caller(0)
		candidates := []string{"rules/ioc.json"}
		if ok {
			candidates = append([]string{filepath.Join(filepath.Dir(file), "..", "..", "rules", "ioc.json")}, candidates...)
		}
		for _, path := range candidates {
			data, err := os.ReadFile(path)
			if err != nil {
				continue
			}
			var l IOCList
			if json.Unmarshal(data, &l) == nil {
				loadedIOC = l
				return
			}
		}
	})
	return loadedIOC
}

// iocEngine matches attachment hashes, sender/URL domains, and URL substrings
// against the local IOC list.
func (p *Pipeline) iocEngine(r model.EmailScanRequest, urls []string) []model.Finding {
	ioc := loadIOC()
	var out []model.Finding

	badHash := map[string]bool{}
	for _, h := range ioc.BadSHA256 {
		badHash[strings.ToLower(strings.TrimSpace(h))] = true
	}
	for _, a := range r.Attachments {
		if a.SHA256 != "" && badHash[strings.ToLower(a.SHA256)] {
			out = append(out, model.Finding{
				Engine: "threat-intel", Category: "threat-intel", Severity: "high",
				Summary: "Attachment hash matches a known-bad IOC: " + a.Name,
			})
		}
	}

	badDom := map[string]bool{}
	for _, d := range ioc.BadDomains {
		badDom[strings.ToLower(strings.TrimSpace(d))] = true
	}
	if d := domainOf(r.Sender); d != "" && (badDom[d] || badDom[registrable(d)]) {
		out = append(out, model.Finding{
			Engine: "threat-intel", Category: "threat-intel", Severity: "high",
			Summary: "Sender domain is on the IOC blocklist: " + d,
		})
	}
	for _, u := range urls {
		lu := strings.ToLower(u)
		host := urlHost(lu)
		if host != "" && (badDom[host] || badDom[registrable(host)]) {
			out = append(out, model.Finding{
				Engine: "threat-intel", Category: "threat-intel", Severity: "high",
				Summary: "URL host is on the IOC blocklist: " + host,
			})
			continue
		}
		for _, sub := range ioc.BadURLSubstring {
			if sub != "" && strings.Contains(lu, strings.ToLower(sub)) {
				out = append(out, model.Finding{
					Engine: "threat-intel", Category: "threat-intel", Severity: "high",
					Summary: "URL matches a known phishing-kit pattern: " + u,
				})
				break
			}
		}
	}
	return out
}

// urlHost extracts the host portion of an http(s) URL, lowercased, without port.
func urlHost(u string) string {
	u = strings.ToLower(strings.TrimSpace(u))
	for _, pfx := range []string{"http://", "https://"} {
		if strings.HasPrefix(u, pfx) {
			u = u[len(pfx):]
			break
		}
	}
	if i := strings.IndexAny(u, "/?#"); i >= 0 {
		u = u[:i]
	}
	if i := strings.LastIndex(u, "@"); i >= 0 {
		u = u[i+1:]
	}
	if i := strings.IndexByte(u, ':'); i >= 0 {
		u = u[:i]
	}
	return u
}
