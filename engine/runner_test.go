package engine

import (
	"container/list"
	"context"
	"errors"
	"testing"
	"time"

	"github.com/gokins/core/runtime"
	"github.com/gokins/runner/runners"
)

func TestBaseRunner_ServerInfo(t *testing.T) {
	r := &baseRunner{}
	info, err := r.ServerInfo()
	if err != nil {
		t.Errorf("ServerInfo() error = %v", err)
	}
	if info == nil {
		t.Error("ServerInfo() returned nil")
	}
}

func TestBaseRunner_FindJobId_EmptyParams(t *testing.T) {
	r := &baseRunner{}
	tests := []struct {
		name    string
		buildID string
		stgNm   string
		stpNm   string
	}{
		{"empty buildID", "", "stage", "step"},
		{"empty stage", "build", "", "step"},
		{"empty step", "build", "stage", ""},
		{"all empty", "", "", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			id, ok := r.FindJobId(tt.buildID, tt.stgNm, tt.stpNm)
			if ok {
				t.Errorf("FindJobId() = (%s, true), want false", id)
			}
		})
	}
}

func TestBaseRunner_GetEnv_EmptyParams(t *testing.T) {
	r := &baseRunner{}
	tests := []struct {
		name    string
		buildID string
		jobId   string
		key     string
	}{
		{"empty jobId", "build", "", "key"},
		{"empty key", "build", "job", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			val, ok := r.GetEnv(tt.buildID, tt.jobId, tt.key)
			if ok {
				t.Errorf("GetEnv() = (%s, true), want false", val)
			}
		})
	}
}

func TestBaseRunner_ReadDir_EmptyParams(t *testing.T) {
	r := &baseRunner{}
	_, err := r.ReadDir(1, "", "path")
	if err == nil {
		t.Error("ReadDir() with empty buildID should error")
	}
	_, err = r.ReadDir(1, "build", "")
	if err == nil {
		t.Error("ReadDir() with empty path should error")
	}
}

func TestBaseRunner_ReadFile_EmptyParams(t *testing.T) {
	r := &baseRunner{}
	_, _, err := r.ReadFile(1, "", "path", 0)
	if err == nil {
		t.Error("ReadFile() with empty buildID should error")
	}
	_, _, err = r.ReadFile(1, "build", "", 0)
	if err == nil {
		t.Error("ReadFile() with empty path should error")
	}
}

func TestBaseRunner_StatFile_EmptyParams(t *testing.T) {
	r := &baseRunner{}
	_, err := r.StatFile(1, "build", "", "dir", "path")
	if err == nil {
		t.Error("StatFile() with empty jobId should error")
	}
	_, err = r.StatFile(1, "build", "job", "dir", "")
	if err == nil {
		t.Error("StatFile() with empty path should error")
	}
}

func TestBaseRunner_UploadFile_EmptyParams(t *testing.T) {
	r := &baseRunner{}
	_, err := r.UploadFile(1, "build", "", "dir", "path", 0)
	if err == nil {
		t.Error("UploadFile() with empty jobId should error")
	}
	_, err = r.UploadFile(1, "build", "job", "dir", "", 0)
	if err == nil {
		t.Error("UploadFile() with empty path should error")
	}
}

func TestBaseRunner_FindArtVersionId_EmptyParams(t *testing.T) {
	r := &baseRunner{}
	_, err := r.FindArtVersionId("", "idnt", "name")
	if err == nil {
		t.Error("FindArtVersionId() with empty buildID should error")
	}
}

func TestBaseRunner_NewArtVersionId_EmptyParams(t *testing.T) {
	r := &baseRunner{}
	_, err := r.NewArtVersionId("", "idnt", "name")
	if err == nil {
		t.Error("NewArtVersionId() with empty buildID should error")
	}
}

func TestBaseRunner_GenEnv_NilParams(t *testing.T) {
	r := &baseRunner{}
	err := r.GenEnv("build", "", nil)
	if err == nil {
		t.Error("GenEnv() with empty jobId should error")
	}
}

// Test that errors.Is works with wrapped errors
func TestErrorWrapping(t *testing.T) {
	baseErr := errors.New("base error")
	wrapped := errors.Join(errors.New("context"), baseErr)
	if !errors.Is(wrapped, baseErr) {
		t.Error("errors.Is should find base error in chain")
	}
}

func TestBaseRunner_PullJobCtx_Cancelled(t *testing.T) {
	r := &baseRunner{}
	// Initialize Mgr if needed
	if Mgr == nil {
		Mgr = &Manager{}
	}
	if Mgr.jobEgn == nil {
		Mgr.jobEgn = &JobEngine{
			execs: make(map[string]*executer),
			jobs:  make(map[string]*jobSync),
		}
	}
	
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately
	
	_, err := r.PullJobCtx(ctx, "test-runner", []string{"plugin1"})
	if err == nil {
		t.Error("PullJobCtx() with cancelled context should error")
	}
	if !errors.Is(err, ErrJobNotFound) {
		t.Errorf("PullJobCtx() error = %v, want ErrJobNotFound", err)
	}
}

func TestBaseRunner_PullJobCtx_Timeout(t *testing.T) {
	r := &baseRunner{}
	if Mgr == nil {
		Mgr = &Manager{}
	}
	if Mgr.jobEgn == nil {
		Mgr.jobEgn = &JobEngine{
			execs: make(map[string]*executer),
			jobs:  make(map[string]*jobSync),
		}
	}
	
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	
	start := time.Now()
	_, err := r.PullJobCtx(ctx, "test-runner", []string{"plugin1"})
	elapsed := time.Since(start)
	
	if err == nil {
		t.Error("PullJobCtx() should timeout and error")
	}
	if elapsed > 200*time.Millisecond {
		t.Errorf("PullJobCtx() took too long: %v", elapsed)
	}
}

func TestBaseRunner_PullJob_DelegatesToPullJobCtx(t *testing.T) {
	r := &baseRunner{}
	if Mgr == nil {
		Mgr = &Manager{}
	}
	if Mgr.jobEgn == nil {
		Mgr.jobEgn = StartJobEngine()
	}

	// Put a job into the job engine so PullJob returns immediately
	ex := &executer{
		plug:  "test-plugin",
		tms:   time.Now(),
		jobwt: list.New(),
	}
	job := &jobSync{
		task: &BuildTask{},
		step: &runtime.Step{Id: "step-1", Step: "test-plugin"},
		runjb: &runners.RunJob{
			Id:   "step-1",
			Step: "test-plugin",
		},
	}
	ex.jobwt.PushBack(job)
	Mgr.jobEgn.exelk.Lock()
	Mgr.jobEgn.execs["test-plugin"] = ex
	Mgr.jobEgn.exelk.Unlock()

	start := time.Now()
	got, err := r.PullJob("test-runner", []string{"test-plugin"})
	elapsed := time.Since(start)

	if err != nil {
		t.Fatalf("PullJob() unexpected error: %v", err)
	}
	if got == nil {
		t.Fatal("PullJob() returned nil job")
	}
	if got.Id != "step-1" {
		t.Errorf("PullJob() job id = %q, want 'step-1'", got.Id)
	}
	// Should return almost immediately since the job is already queued
	if elapsed > time.Second {
		t.Errorf("PullJob() took too long: %v", elapsed)
	}
}
