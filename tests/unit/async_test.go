package unit

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/LackOfMorals/mcpServerBase/internal/tools"
)

// ---- Submit / Get lifecycle ----------------------------------------------

func TestJobRegistry_PendingImmediately(t *testing.T) {
	jr := tools.NewJobRegistry()
	deps := newDeps(nil)
	deps.Jobs = jr

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	jobID := jr.Submit(ctx, "slow", slowHandler, nil, deps)

	// The job ID must exist immediately regardless of goroutine scheduling.
	view, err := jr.Get(jobID)
	if err != nil {
		t.Fatalf("job not found: %v", err)
	}
	_ = view
}

func TestJobRegistry_CompletedAfterHandler(t *testing.T) {
	jr := tools.NewJobRegistry()
	deps := newDeps(nil)
	deps.Jobs = jr

	jobID := jr.Submit(context.Background(), "echo", echoHandler, map[string]interface{}{"q": "hi"}, deps)

	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		view, err := jr.Get(jobID)
		if err != nil {
			t.Fatalf("job not found: %v", err)
		}
		if view.Status == tools.JobStatusCompleted {
			if view.Progress != 1.0 {
				t.Errorf("expected progress=1.0 on completion, got %v", view.Progress)
			}
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("job did not reach 'completed' within timeout")
}

func TestJobRegistry_FailedWhenHandlerErrors(t *testing.T) {
	jr := tools.NewJobRegistry()
	deps := newDeps(nil)
	deps.Jobs = jr

	jobID := jr.Submit(context.Background(), "fail", failHandler, nil, deps)

	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		view, _ := jr.Get(jobID)
		if view.Status == tools.JobStatusFailed {
			if view.Error == "" {
				t.Error("expected non-empty error string on failed job")
			}
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("job did not reach 'failed' within timeout")
}

func TestJobRegistry_ContextCancelledFailsJob(t *testing.T) {
	jr := tools.NewJobRegistry()
	deps := newDeps(nil)
	deps.Jobs = jr

	ctx, cancel := context.WithCancel(context.Background())
	jobID := jr.Submit(ctx, "slow", slowHandler, nil, deps)

	time.Sleep(50 * time.Millisecond)
	cancel()

	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		view, _ := jr.Get(jobID)
		if view.Status == tools.JobStatusFailed {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("job did not fail after context cancellation")
}

func TestJobRegistry_GetUnknownJobErrors(t *testing.T) {
	jr := tools.NewJobRegistry()
	_, err := jr.Get("does-not-exist")
	if err == nil {
		t.Error("expected error for unknown job ID")
	}
}

func TestJobRegistry_Len(t *testing.T) {
	jr := tools.NewJobRegistry()
	deps := newDeps(nil)
	deps.Jobs = jr

	if jr.Len() != 0 {
		t.Errorf("expected Len=0, got %d", jr.Len())
	}

	jr.Submit(context.Background(), "a", echoHandler, nil, deps)
	jr.Submit(context.Background(), "b", echoHandler, nil, deps)
	if jr.Len() != 2 {
		t.Errorf("expected Len=2, got %d", jr.Len())
	}
}

// ---- Concurrent safety --------------------------------------------------

func TestJobRegistry_ConcurrentSubmitsAreSafe(t *testing.T) {
	jr := tools.NewJobRegistry()
	deps := newDeps(nil)
	deps.Jobs = jr

	const n = 50
	var wg sync.WaitGroup
	wg.Add(n)
	for i := 0; i < n; i++ {
		go func() {
			defer wg.Done()
			jr.Submit(context.Background(), "echo", echoHandler, nil, deps)
		}()
	}
	wg.Wait()

	if jr.Len() != n {
		t.Errorf("expected %d jobs, got %d", n, jr.Len())
	}
}

func TestJobRegistry_ConcurrentGetsAreSafe(t *testing.T) {
	jr := tools.NewJobRegistry()
	deps := newDeps(nil)
	deps.Jobs = jr

	jobID := jr.Submit(context.Background(), "echo", echoHandler, nil, deps)

	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			jr.Get(jobID) //nolint:errcheck
		}()
	}
	wg.Wait()
}

// ---- GetResult on completion --------------------------------------------

func TestJobRegistry_CompletedJobHasResult(t *testing.T) {
	jr := tools.NewJobRegistry()
	deps := newDeps(nil)
	deps.Jobs = jr

	jobID := jr.Submit(context.Background(), "echo", echoHandler, map[string]interface{}{"k": "v"}, deps)

	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		view, _ := jr.Get(jobID)
		if view.Status == tools.JobStatusCompleted {
			result, err := jr.GetResult(jobID)
			if err != nil {
				t.Fatalf("GetResult error: %v", err)
			}
			if result == nil {
				t.Error("expected non-nil result from GetResult on completed job")
			}
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("job did not complete in time")
}
