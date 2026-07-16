package store

import (
	"path/filepath"
	"testing"
)

func TestStorePersistsAndReloads(t *testing.T) {
	path := filepath.Join(t.TempDir(), "events.jsonl")
	s, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	s.Append(Event{Kind: "scan", Status: "malicious", Score: 90})
	s.Append(Event{Kind: "scan", Status: "clean", Score: 0})

	reopened, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	if got := reopened.Stats().Total; got != 2 {
		t.Fatalf("total = %d, want 2", got)
	}
	if reopened.Stats().ByStatus["malicious"] != 1 {
		t.Fatalf("malicious count wrong: %+v", reopened.Stats())
	}
	recent := reopened.Recent(1)
	if len(recent) != 1 || recent[0].Status != "clean" {
		t.Fatalf("recent newest-first wrong: %+v", recent)
	}
}
