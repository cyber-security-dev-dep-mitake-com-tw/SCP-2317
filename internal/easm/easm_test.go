package easm

import (
	"context"
	"net"
	"testing"
)

func TestParseSPF(t *testing.T) {
	if got := ParseSPF([]string{"something", "v=spf1 include:_spf.google.com ~all"}); got == "" {
		t.Fatal("expected SPF record")
	}
	if got := ParseSPF([]string{"no spf here"}); got != "" {
		t.Fatalf("unexpected SPF: %q", got)
	}
}

func TestParseDMARCPolicy(t *testing.T) {
	got := ParseDMARCPolicy([]string{"v=DMARC1; p=reject; rua=mailto:a@b"})
	if got != "reject" {
		t.Fatalf("policy = %q, want reject", got)
	}
	if ParseDMARCPolicy([]string{"unrelated"}) != "" {
		t.Fatal("expected empty policy")
	}
}

func TestEvaluateFindings(t *testing.T) {
	f := EvaluateFindings(Result{HasSPF: false, DMARCPolicy: "none"})
	if len(f) != 2 {
		t.Fatalf("expected 2 findings, got %v", f)
	}
	if len(EvaluateFindings(Result{HasSPF: true, DMARCPolicy: "reject"})) != 0 {
		t.Fatal("healthy domain should have no findings")
	}
}

// fakeResolver lets Check run offline.
type fakeResolver struct {
	txt map[string][]string
	mx  []*net.MX
}

func (f fakeResolver) LookupTXT(_ context.Context, name string) ([]string, error) {
	return f.txt[name], nil
}
func (f fakeResolver) LookupMX(_ context.Context, _ string) ([]*net.MX, error) {
	return f.mx, nil
}

func TestCheckWithFakeResolver(t *testing.T) {
	r := fakeResolver{
		txt: map[string][]string{
			"example.com":        {"v=spf1 -all"},
			"_dmarc.example.com": {"v=DMARC1; p=reject"},
		},
		mx: []*net.MX{{Host: "mx.example.com"}},
	}
	res := Check(context.Background(), r, "example.com")
	if !res.HasSPF || res.DMARCPolicy != "reject" || !res.HasMX {
		t.Fatalf("unexpected result: %+v", res)
	}
	if len(res.Findings) != 0 {
		t.Fatalf("healthy domain should have no findings: %+v", res.Findings)
	}
}
