package llm

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestFromEnvDisabledByDefault(t *testing.T) {
	t.Setenv("SCP2317_LLM_API_KEY", "")
	if _, ok := FromEnv(); ok {
		t.Fatal("analyzer must be disabled without an API key")
	}
}

func TestAnalyzeAcrossFormats(t *testing.T) {
	for _, f := range []Format{FormatOpenAI, FormatGrok, FormatMistral} {
		var gotAuth, gotPath, gotModel string
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			gotAuth = r.Header.Get("Authorization")
			gotPath = r.URL.Path
			body, _ := io.ReadAll(r.Body)
			var cr chatRequest
			_ = json.Unmarshal(body, &cr)
			gotModel = cr.Model
			w.Header().Set("Content-Type", "application/json")
			// content is a JSON string whose value itself contains JSON.
			inner := `{"risk":"high","is_bec":true,"reason":"urgent wire"}`
			out := map[string]any{"choices": []any{map[string]any{"message": map[string]any{"role": "assistant", "content": inner}}}}
			_ = json.NewEncoder(w).Encode(out)
		}))
		defer srv.Close()

		a := New(Config{Format: f, BaseURL: srv.URL, APIKey: "k-" + string(f), Model: "test-model"})
		as, err := a.Analyze(context.Background(), "From: ceo\nSubject: urgent\n\nwire now")
		if err != nil {
			t.Fatalf("%s: %v", f, err)
		}
		if as.Risk != "high" || !as.IsBEC {
			t.Fatalf("%s: unexpected assessment %+v", f, as)
		}
		if !strings.HasPrefix(gotAuth, "Bearer k-") {
			t.Fatalf("%s: missing bearer auth: %q", f, gotAuth)
		}
		if !strings.HasSuffix(gotPath, "/chat/completions") {
			t.Fatalf("%s: wrong path %q", f, gotPath)
		}
		if gotModel != "test-model" {
			t.Fatalf("%s: wrong model %q", f, gotModel)
		}
	}
}

func TestParseAssessmentPlainJSON(t *testing.T) {
	as, err := parseAssessment([]byte(`{"choices":[{"message":{"content":"{\"risk\":\"LOW\",\"is_bec\":false,\"reason\":\"ok\"}"}}]}`))
	if err != nil {
		t.Fatal(err)
	}
	if as.Risk != "low" || as.IsBEC {
		t.Fatalf("got %+v", as)
	}
}
