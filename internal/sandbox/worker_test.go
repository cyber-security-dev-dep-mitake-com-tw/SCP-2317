package sandbox

import "testing"

func TestMemoryWorkerNeverExecutes(t *testing.T) {
	w := NewMemoryWorker()
	sub, err := w.Submit(SubmitRequest{Hash: "abc", ObjectRef: "obj://abc", Policy: "default"})
	if err != nil {
		t.Fatal(err)
	}
	st, err := w.Status(sub.JobID)
	if err != nil {
		t.Fatal(err)
	}
	if st.Status != "not_executed" {
		t.Fatalf("status=%s", st.Status)
	}
	rep, err := w.Report(sub.JobID)
	if err != nil {
		t.Fatal(err)
	}
	if len(rep.Findings) == 0 || rep.Findings[0].Engine != "sandbox-stub" {
		t.Fatalf("unexpected report: %+v", rep)
	}
}
