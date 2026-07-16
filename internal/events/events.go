// Package events forwards companion-app events to an external SIEM/SOAR sink.
// This is the endpoint half of the "ship logs to SIEM" capability: it POSTs
// JSON events to a configurable HTTP collector (e.g. a Splunk HEC, Elastic, or
// a webhook). It is a no-op when no sink URL is configured, keeping the default
// build offline.
package events

import (
	"bytes"
	"encoding/json"
	"net/http"
	"time"

	"github.com/example/scp2317-agent/internal/store"
)

type Sink struct {
	url    string
	client *http.Client
}

// NewSink returns a sink posting to url, or an inert sink when url is empty.
func NewSink(url string) *Sink {
	return &Sink{url: url, client: &http.Client{Timeout: 4 * time.Second}}
}

// Send forwards an event asynchronously; failures are swallowed (best effort).
func (s *Sink) Send(e store.Event) {
	if s == nil || s.url == "" {
		return
	}
	go func() {
		body, err := json.Marshal(e)
		if err != nil {
			return
		}
		req, err := http.NewRequest(http.MethodPost, s.url, bytes.NewReader(body))
		if err != nil {
			return
		}
		req.Header.Set("Content-Type", "application/json")
		if resp, err := s.client.Do(req); err == nil {
			resp.Body.Close()
		}
	}()
}
