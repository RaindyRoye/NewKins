package engine

import (
	"container/list"
	"sync"
	"testing"
	"time"

	"github.com/gokins/core/common"
	"github.com/gokins/core/runtime"
)

// TestJobSyncDoneChannel verifies that doneCh is closed when job reaches terminal status
func TestJobSyncDoneChannel(t *testing.T) {
	job := &jobSync{
		step: &runtime.Step{
			Name: "test-step",
		},
		cmdmp:  make(map[string]*cmdSync),
		doneCh: make(chan struct{}),
	}

	// Initially, doneCh should not be closed
	select {
	case <-job.doneCh:
		t.Error("doneCh should not be closed initially")
	case <-time.After(10 * time.Millisecond):
		// Expected
	}

	// Set status to running (non-terminal)
	job.status(common.BuildStatusRunning, "")
	select {
	case <-job.doneCh:
		t.Error("doneCh should not be closed for running status")
	case <-time.After(10 * time.Millisecond):
		// Expected
	}

	// Set status to success (terminal)
	job.status(common.BuildStatusOk, "")
	select {
	case <-job.doneCh:
		// Expected - channel should be closed
	case <-time.After(10 * time.Millisecond):
		t.Error("doneCh should be closed when status reaches terminal state")
	}
}

// TestJobSyncSignalDoneIdempotent verifies that signalDone can be called multiple times safely
func TestJobSyncSignalDoneIdempotent(t *testing.T) {
	job := &jobSync{
		step: &runtime.Step{
			Name: "test-step",
		},
		cmdmp:  make(map[string]*cmdSync),
		doneCh: make(chan struct{}),
	}

	// Call signalDone multiple times - should not panic
	job.signalDone()
	job.signalDone()
	job.signalDone()

	// Verify channel is closed
	select {
	case <-job.doneCh:
		// Expected
	case <-time.After(10 * time.Millisecond):
		t.Error("doneCh should be closed after signalDone")
	}
}

// TestJobSyncInitDoneCh verifies that initDoneCh creates the channel if nil
func TestJobSyncInitDoneCh(t *testing.T) {
	job := &jobSync{
		step: &runtime.Step{
			Name: "test-step",
		},
		cmdmp: make(map[string]*cmdSync),
	}

	if job.doneCh != nil {
		t.Error("doneCh should be nil initially")
	}

	job.initDoneCh()
	if job.doneCh == nil {
		t.Error("doneCh should be created by initDoneCh")
	}

	// Call again - should not create new channel
	ch1 := job.doneCh
	job.initDoneCh()
	if job.doneCh != ch1 {
		t.Error("initDoneCh should not recreate existing channel")
	}
}

// TestJobEnginePullWakeChannel verifies that Put signals pullWakeCh
func TestJobEnginePullWakeChannel(t *testing.T) {
	je := &JobEngine{
		wakeCh:     make(chan struct{}, 1),
		pullWakeCh: make(chan struct{}, 1),
		jobs:       make(map[string]*jobSync),
		execs:      make(map[string]*executer),
	}

	// Create an executor
	ex := &executer{
		plug:  "test-plugin",
		tms:   time.Now(),
		jobwt: list.New(),
	}
	je.execs["test-plugin"] = ex

	// Create a job
	job := &jobSync{
		step: &runtime.Step{
			Name: "test-step",
			Step: "test-plugin",
		},
		cmdmp: make(map[string]*cmdSync),
	}

	// Put the job
	err := je.Put(job)
	if err != nil {
		t.Fatalf("Put failed: %v", err)
	}

	// Verify pullWakeCh was signaled
	select {
	case <-je.pullWakeCh:
		// Expected
	case <-time.After(10 * time.Millisecond):
		t.Error("pullWakeCh should be signaled when job is put")
	}
}

// TestConcurrentSignalDone verifies that concurrent signalDone calls are safe
func TestConcurrentSignalDone(t *testing.T) {
	job := &jobSync{
		step: &runtime.Step{
			Name: "test-step",
		},
		cmdmp:  make(map[string]*cmdSync),
		doneCh: make(chan struct{}),
	}

	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			job.signalDone()
		}()
	}

	wg.Wait()

	// Verify channel is closed
	select {
	case <-job.doneCh:
		// Expected
	case <-time.After(10 * time.Millisecond):
		t.Error("doneCh should be closed after concurrent signalDone calls")
	}
}
