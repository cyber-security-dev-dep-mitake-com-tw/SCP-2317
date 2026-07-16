package sandbox

import (
	"fmt"
	"sync"
	"time"

	"github.com/example/scp2317-agent/internal/model"
)

// Worker is the disposable-VM analysis contract. Production workers must run
// outside the API process with network-denied ephemeral VMs.
type Worker interface {
	Submit(req SubmitRequest) (SubmitResponse, error)
	Status(jobID string) (StatusResponse, error)
	Report(jobID string) (ReportResponse, error)
}

type SubmitRequest struct {
	Hash      string `json:"hash"`
	ObjectRef string `json:"objectRef"`
	Policy    string `json:"policy"`
}

type SubmitResponse struct {
	JobID string `json:"job_id"`
}

type StatusResponse struct {
	JobID  string `json:"job_id"`
	Status string `json:"status"`
}

type ReportResponse struct {
	JobID    string          `json:"job_id"`
	Status   string          `json:"status"`
	Findings []model.Finding `json:"findings"`
}

type job struct {
	status   string
	findings []model.Finding
}

// MemoryWorker acknowledges jobs without executing samples.
type MemoryWorker struct {
	mu   sync.Mutex
	seq  int
	jobs map[string]job
}

func NewMemoryWorker() *MemoryWorker {
	return &MemoryWorker{jobs: map[string]job{}}
}

func (w *MemoryWorker) Submit(req SubmitRequest) (SubmitResponse, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.seq++
	id := fmt.Sprintf("job-%d-%d", time.Now().UnixNano(), w.seq)
	w.jobs[id] = job{
		status: "not_executed",
		findings: []model.Finding{{
			Engine:   "sandbox-stub",
			Severity: "info",
			Summary:  "Dynamic analysis is not enabled; sample was not executed (policy-only)",
		}},
	}
	_ = req
	return SubmitResponse{JobID: id}, nil
}

func (w *MemoryWorker) Status(jobID string) (StatusResponse, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	j, ok := w.jobs[jobID]
	if !ok {
		return StatusResponse{}, fmt.Errorf("unknown job %q", jobID)
	}
	return StatusResponse{JobID: jobID, Status: j.status}, nil
}

func (w *MemoryWorker) Report(jobID string) (ReportResponse, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	j, ok := w.jobs[jobID]
	if !ok {
		return ReportResponse{}, fmt.Errorf("unknown job %q", jobID)
	}
	return ReportResponse{JobID: jobID, Status: j.status, Findings: j.findings}, nil
}
