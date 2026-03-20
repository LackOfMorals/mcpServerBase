// async.go — async tool execution via JobRegistry.
//
// When execute-tool is called with async=true the handler submits the tool
// to a JobRegistry goroutine and returns a job_id immediately.  The caller
// then polls get-tool-status until the job reaches "completed" or "failed",
// at which point the raw *mcp.CallToolResult is returned via GetResult.

package tools

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/mark3labs/mcp-go/mcp"
)

// JobStatus represents the lifecycle state of an async tool execution.
type JobStatus string

const (
	JobStatusPending   JobStatus = "pending"
	JobStatusRunning   JobStatus = "running"
	JobStatusCompleted JobStatus = "completed"
	JobStatusFailed    JobStatus = "failed"
)

// AsyncJob tracks a single async tool execution in-memory.
type AsyncJob struct {
	ID        string              `json:"id"`
	ToolID    string              `json:"tool_id"`
	Status    JobStatus           `json:"status"`
	Progress  float64             `json:"progress"`
	Message   string              `json:"message,omitempty"`
	Error     string              `json:"error,omitempty"`
	Result    *mcp.CallToolResult `json:"-"` // not serialised directly
	CreatedAt time.Time           `json:"created_at"`
	UpdatedAt time.Time           `json:"updated_at"`
}

// AsyncJobView is the JSON-serialisable snapshot returned to callers.
// It represents job state for pending, running, and failed jobs.
// Completed jobs are returned as raw *mcp.CallToolResult via GetResult.
type AsyncJobView struct {
	ID        string    `json:"id"`
	ToolID    string    `json:"tool_id"`
	Status    JobStatus `json:"status"`
	Progress  float64   `json:"progress"`
	Message   string    `json:"message,omitempty"`
	Error     string    `json:"error,omitempty"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// JobRegistry stores and manages all async jobs.
// All methods are safe for concurrent use.
type JobRegistry struct {
	mu   sync.RWMutex
	jobs map[string]*AsyncJob
}

// NewJobRegistry creates an empty JobRegistry.
func NewJobRegistry() *JobRegistry {
	return &JobRegistry{jobs: make(map[string]*AsyncJob)}
}

// Submit creates a new job, runs the handler in a goroutine, and returns the
// job ID immediately. The supplied context is forwarded to the handler — cancel
// it to interrupt a long-running tool.
func (jr *JobRegistry) Submit(
	ctx context.Context,
	toolID string,
	handler ToolHandler,
	parameters map[string]interface{},
	deps *Dependencies,
) string {
	id := uuid.New().String()

	job := &AsyncJob{
		ID:        id,
		ToolID:    toolID,
		Status:    JobStatusPending,
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	}

	jr.mu.Lock()
	jr.jobs[id] = job
	jr.mu.Unlock()

	go func() {
		jr.update(id, func(j *AsyncJob) {
			j.Status = JobStatusRunning
		})

		result, err := handler(ctx, parameters, deps)

		if err != nil {
			jr.update(id, func(j *AsyncJob) {
				j.Status = JobStatusFailed
				j.Error = err.Error()
			})
			return
		}

		jr.update(id, func(j *AsyncJob) {
			j.Status = JobStatusCompleted
			j.Progress = 1.0
			j.Result = result
		})
	}()

	return id
}

// Get returns a read-only JSON-serialisable snapshot of the job, or an error
// if the ID is not found.
func (jr *JobRegistry) Get(id string) (*AsyncJobView, error) {
	jr.mu.RLock()
	defer jr.mu.RUnlock()

	job, exists := jr.jobs[id]
	if !exists {
		return nil, fmt.Errorf("job %q not found", id)
	}

	return &AsyncJobView{
		ID:        job.ID,
		ToolID:    job.ToolID,
		Status:    job.Status,
		Progress:  job.Progress,
		Message:   job.Message,
		Error:     job.Error,
		CreatedAt: job.CreatedAt,
		UpdatedAt: job.UpdatedAt,
	}, nil
}

// GetResult returns the raw *mcp.CallToolResult for a completed job.
// Returns nil, nil when the job exists but has not yet completed.
// Returns nil, err when the job ID is unknown.
func (jr *JobRegistry) GetResult(id string) (*mcp.CallToolResult, error) {
	jr.mu.RLock()
	defer jr.mu.RUnlock()

	job, exists := jr.jobs[id]
	if !exists {
		return nil, fmt.Errorf("job %q not found", id)
	}
	if job.Status == JobStatusCompleted && job.Result != nil {
		return job.Result, nil
	}
	return nil, nil
}

// Len returns the number of jobs currently tracked (useful in tests).
func (jr *JobRegistry) Len() int {
	jr.mu.RLock()
	defer jr.mu.RUnlock()
	return len(jr.jobs)
}

// update applies a mutation function under the write lock.
func (jr *JobRegistry) update(id string, fn func(*AsyncJob)) {
	jr.mu.Lock()
	defer jr.mu.Unlock()
	if job, ok := jr.jobs[id]; ok {
		fn(job)
		job.UpdatedAt = time.Now().UTC()
	}
}
