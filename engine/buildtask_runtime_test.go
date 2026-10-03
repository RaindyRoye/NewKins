package engine

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/gokins/core/common"
	"github.com/gokins/core/runtime"
	"github.com/gokins/gokins/comm"
	"github.com/gokins/runner/runners"
	"github.com/stretchr/testify/assert"
)

// TestBuildTask_Write tests the Write method for progress parsing
func TestBuildTask_Write(t *testing.T) {
	tests := []struct {
		name           string
		input          string
		expectedProgress int
	}{
		{
			name:           "no progress match",
			input:          "Cloning into 'repo'...",
			expectedProgress: 0,
		},
		{
			name:           "progress 50%",
			input:          "Receiving objects:  50% (100/200)",
			expectedProgress: 40, // 50 * 0.8
		},
		{
			name:           "progress 100%",
			input:          "Receiving objects: 100% (200/200)",
			expectedProgress: 80, // 100 * 0.8
		},
		{
			name:           "empty input",
			input:          "",
			expectedProgress: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bt := &BuildTask{
				build: &runtime.Build{Id: "test-build"},
			}
			n, err := bt.Write([]byte(tt.input))
			assert.NoError(t, err)
			assert.Equal(t, len(tt.input), n)
			assert.Equal(t, tt.expectedProgress, bt.workpgss)
		})
	}
}

// TestBuildTask_GetRepo tests getRepo with various scenarios
func TestBuildTask_GetRepo(t *testing.T) {
	t.Run("not clone", func(t *testing.T) {
		bt := &BuildTask{
			isClone: false,
		}
		err := bt.getRepo()
		assert.NoError(t, err)
	})

	t.Run("clone with empty repoPath", func(t *testing.T) {
		bt := &BuildTask{
			isClone:   true,
			repoPaths: t.TempDir(),
			repoPath:  "",
			build:     &runtime.Build{Id: "test-build"},
		}
		err := bt.getRepo()
		assert.NoError(t, err)
	})
}

// TestBuildTask_UpJob tests UpJob status updates
func TestBuildTask_UpJob(t *testing.T) {
	t.Run("nil job", func(t *testing.T) {
		bt := &BuildTask{
			build: &runtime.Build{Id: "test-build"},
		}
		// Should not panic
		bt.UpJob(nil, common.BuildStatusRunning, "", 0)
	})

	t.Run("empty status", func(t *testing.T) {
		job := &jobSync{
			step:  &runtime.Step{Id: "step-1"},
			cmdmp: make(map[string]*cmdSync),
		}
		bt := &BuildTask{
			build: &runtime.Build{Id: "test-build"},
		}
		// Should not panic
		bt.UpJob(job, "", "", 0)
		assert.Equal(t, "", job.step.Status)
	})

	t.Run("valid status update", func(t *testing.T) {
		job := &jobSync{
			step:  &runtime.Step{Id: "step-1"},
			cmdmp: make(map[string]*cmdSync),
		}
		bt := &BuildTask{
			build: &runtime.Build{Id: "test-build"},
		}
		bt.UpJob(job, common.BuildStatusRunning, "test error", 1)
		assert.Equal(t, common.BuildStatusRunning, job.step.Status)
		assert.Equal(t, "test error", job.step.Error)
		assert.Equal(t, 1, job.step.ExitCode)
	})
}

// TestBuildTask_UpJobCmd tests UpJobCmd with various filesystem flags
func TestBuildTask_UpJobCmd(t *testing.T) {
	tests := []struct {
		name           string
		fs             int
		code           int
		expectedStatus string
		expectStarted  bool
		expectFinished bool
	}{
		{
			name:           "fs=1 running",
			fs:             1,
			code:           0,
			expectedStatus: common.BuildStatusRunning,
			expectStarted:  true,
			expectFinished: false,
		},
		{
			name:           "fs=2 success",
			fs:             2,
			code:           0,
			expectedStatus: common.BuildStatusOk,
			expectStarted:  false,
			expectFinished: true,
		},
		{
			name:           "fs=2 with error code",
			fs:             2,
			code:           1,
			expectedStatus: common.BuildStatusError,
			expectStarted:  false,
			expectFinished: true,
		},
		{
			name:           "fs=3 cancel",
			fs:             3,
			code:           0,
			expectedStatus: common.BuildStatusCancel,
			expectStarted:  false,
			expectFinished: true,
		},
		{
			name:           "fs=-1 error",
			fs:             -1,
			code:           2,
			expectedStatus: common.BuildStatusError,
			expectStarted:  false,
			expectFinished: true,
		},
		{
			name:           "unknown fs",
			fs:             999,
			code:           0,
			expectedStatus: common.BuildStatusPending,
			expectStarted:  false,
			expectFinished: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := &cmdSync{
				cmd:    &runners.CmdContent{Id: "cmd-1"},
				status: common.BuildStatusPending,
			}
			bt := &BuildTask{
				build: &runtime.Build{Id: "test-build"},
			}
			bt.UpJobCmd(cmd, tt.fs, tt.code)
			assert.Equal(t, tt.expectedStatus, cmd.status)
			if tt.expectStarted {
				assert.False(t, cmd.started.IsZero())
			}
			if tt.expectFinished {
				assert.False(t, cmd.finished.IsZero())
			}
			if tt.code != 0 {
				assert.Equal(t, tt.code, cmd.code)
			}
		})
	}

	t.Run("nil cmd", func(t *testing.T) {
		bt := &BuildTask{
			build: &runtime.Build{Id: "test-build"},
		}
		// Should not panic
		bt.UpJobCmd(nil, 1, 0)
	})
}

// TestTaskStage_Status tests taskStage status updates
func TestTaskStage_Status(t *testing.T) {
	t.Run("status without event", func(t *testing.T) {
		stg := &taskStage{
			stage: &runtime.Stage{
				Id:     "stage-1",
				Status: common.BuildStatusPending,
			},
			jobs: make(map[string]*jobSync),
		}
		stg.status(common.BuildStatusRunning, "test error")
		assert.Equal(t, common.BuildStatusRunning, stg.stage.Status)
		assert.Equal(t, "test error", stg.stage.Error)
	})

	t.Run("status with event", func(t *testing.T) {
		stg := &taskStage{
			stage: &runtime.Stage{
				Id:     "stage-1",
				Status: common.BuildStatusPending,
			},
			jobs: make(map[string]*jobSync),
		}
		stg.status(common.BuildStatusError, "test error", "test-event")
		assert.Equal(t, common.BuildStatusError, stg.stage.Status)
		assert.Equal(t, "test error", stg.stage.Error)
		assert.Equal(t, "test-event", stg.stage.Event)
	})

	t.Run("status preserves event when empty", func(t *testing.T) {
		stg := &taskStage{
			stage: &runtime.Stage{
				Id:     "stage-1",
				Status: common.BuildStatusPending,
				Event:  "initial-event",
			},
			jobs: make(map[string]*jobSync),
		}
		stg.status(common.BuildStatusRunning, "new error")
		assert.Equal(t, "initial-event", stg.stage.Event)
	})
}

// TestJobSync_Status tests jobSync status updates
func TestJobSync_Status(t *testing.T) {
	t.Run("status without event", func(t *testing.T) {
		job := &jobSync{
			step: &runtime.Step{
				Id:     "step-1",
				Status: common.BuildStatusPending,
			},
			cmdmp: make(map[string]*cmdSync),
		}
		job.status(common.BuildStatusRunning, "test error")
		assert.Equal(t, common.BuildStatusRunning, job.step.Status)
		assert.Equal(t, "test error", job.step.Error)
	})

	t.Run("status with event", func(t *testing.T) {
		job := &jobSync{
			step: &runtime.Step{
				Id:     "step-1",
				Status: common.BuildStatusPending,
			},
			cmdmp: make(map[string]*cmdSync),
		}
		job.status(common.BuildStatusError, "test error", "test-event")
		assert.Equal(t, common.BuildStatusError, job.step.Status)
		assert.Equal(t, "test error", job.step.Error)
		assert.Equal(t, "test-event", job.step.Event)
	})
}

// TestBuildTask_Cancel tests cancel behavior
func TestBuildTask_Cancel(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	bt := &BuildTask{
		build: &runtime.Build{Id: "test-build"},
		ctx:   ctx,
		cncl:  cancel,
	}
	
	before := time.Now()
	bt.Cancel()
	after := time.Now()
	
	assert.False(t, bt.ctrlendtm.IsZero())
	assert.True(t, bt.ctrlendtm.After(before) || bt.ctrlendtm.Equal(before))
	assert.True(t, bt.ctrlendtm.Before(after) || bt.ctrlendtm.Equal(after))
	assert.Error(t, ctx.Err())
}

// TestBuildTask_Stop tests stop behavior
func TestBuildTask_Stop(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	bt := &BuildTask{
		build: &runtime.Build{Id: "test-build"},
		ctx:   ctx,
		cncl:  cancel,
	}
	
	bt.stop()
	assert.True(t, bt.ctrlendtm.IsZero())
	assert.Error(t, ctx.Err())
}

// TestBuildTask_Run_WithStages tests run with multiple stages
func TestBuildTask_Run_WithStages(t *testing.T) {
	bt := &BuildTask{
		build: &runtime.Build{
			Id:     "test-build",
			Status: common.BuildStatusPending,
			Repo: &runtime.Repository{
				CloneURL: t.TempDir(),
			},
			Stages: []*runtime.Stage{
				{Id: "stage-1", Name: "build", BuildId: "test-build"},
				{Id: "stage-2", Name: "test", BuildId: "test-build"},
			},
		},
		stages: make(map[string]*taskStage),
		jobs:   make(map[string]*jobSync),
	}
	
	// Mock the database to avoid actual writes
	origDb := comm.Db
	defer func() { comm.Db = origDb }()
	comm.Db = nil // This will cause updateBuild to fail gracefully
	
	bt.run()
	
	// Should complete without panic
	assert.False(t, bt.build.Finished.IsZero())
}

// TestBuildTask_ConcurrentStatusUpdates tests thread safety of status updates
func TestBuildTask_ConcurrentStatusUpdates(t *testing.T) {
	bt := &BuildTask{
		build: &runtime.Build{Id: "test-build"},
	}
	
	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			bt.status(common.BuildStatusRunning, "", "")
		}(i)
	}
	wg.Wait()
	
	assert.Equal(t, common.BuildStatusRunning, bt.build.Status)
}
