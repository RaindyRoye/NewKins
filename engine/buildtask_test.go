package engine

import (
	"fmt"
	"testing"
	"time"

	"github.com/gokins/core/common"
	"github.com/gokins/core/runtime"
)

func TestGitClone(t *testing.T) {
	task := BuildTask{
		build: &runtime.Build{
			Id: "1231",
			Repo: &runtime.Repository{
				Name:     "",
				Token:    "",
				Sha:      "c202ee042db1fc8b8c16c6c968195cec6185d7db",
				CloneURL: "https://gitee.com/SuperHeroJim/gokins-test",
			},
		},
	}
	err := task.getRepo()
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println(task.repoPath)
	fmt.Println(task.isClone)
}

func TestNewBuildTaskInitializesJobDoneCh(t *testing.T) {
	egn := &BuildEngine{}
	bd := &runtime.Build{Id: "test-build"}
	bt := NewBuildTask(egn, bd)
	
	if bt.jobDoneCh == nil {
		t.Error("NewBuildTask should initialize jobDoneCh")
	}
	
	// Verify it's a buffered channel with capacity 1
	if cap(bt.jobDoneCh) != 1 {
		t.Errorf("jobDoneCh should have capacity 1, got %d", cap(bt.jobDoneCh))
	}
}

func TestUpJobSignalsJobDoneCh(t *testing.T) {
	egn := &BuildEngine{}
	bd := &runtime.Build{Id: "test-build"}
	bt := NewBuildTask(egn, bd)
	
	job := &jobSync{
		step: &runtime.Step{},
	}
	
	// Test that terminal status signals the channel
	terminalStatuses := []string{
		common.BuildStatusOk,
		common.BuildStatusError,
		common.BuildStatusCancel,
	}
	
	for _, status := range terminalStatuses {
		t.Run("status_"+status, func(t *testing.T) {
			// Reset the channel
			bt.jobDoneCh = make(chan struct{}, 1)
			
			bt.UpJob(job, status, "test error", 0)
			
			// Give the goroutine time to signal
			time.Sleep(10 * time.Millisecond)
			
			// Channel should have been signaled
			select {
			case <-bt.jobDoneCh:
				// Success
			default:
				t.Errorf("UpJob with status %s should signal jobDoneCh", status)
			}
		})
	}
}

func TestUpJobDoesNotSignalNonTerminalStatus(t *testing.T) {
	egn := &BuildEngine{}
	bd := &runtime.Build{Id: "test-build"}
	bt := NewBuildTask(egn, bd)
	
	job := &jobSync{
		step: &runtime.Step{},
	}
	
	// Test that non-terminal status does not signal the channel
	nonTerminalStatuses := []string{
		common.BuildStatusRunning,
		common.BuildStatusPending,
		common.BuildStatusPreparation,
	}
	
	for _, status := range nonTerminalStatuses {
		t.Run("status_"+status, func(t *testing.T) {
			// Reset the channel
			bt.jobDoneCh = make(chan struct{}, 1)
			
			bt.UpJob(job, status, "", 0)
			
			// Give the goroutine time to execute
			time.Sleep(10 * time.Millisecond)
			
			// Channel should NOT have been signaled
			select {
			case <-bt.jobDoneCh:
				t.Errorf("UpJob with non-terminal status %s should not signal jobDoneCh", status)
			default:
				// Success - channel was not signaled
			}
		})
	}
}

func TestTaskStageStatusSignalsDoneCh(t *testing.T) {
	stage := &taskStage{
		stage:  &runtime.Stage{},
		jobs:   make(map[string]*jobSync),
		doneCh: make(chan struct{}, 1),
	}
	
	// Test that terminal status signals the channel
	terminalStatuses := []string{
		common.BuildStatusOk,
		common.BuildStatusError,
		common.BuildStatusCancel,
	}
	
	for _, status := range terminalStatuses {
		t.Run("status_"+status, func(t *testing.T) {
			// Reset the channel
			stage.doneCh = make(chan struct{}, 1)
			
			stage.status(status, "test error", "test-event")
			
			// Channel should have been signaled
			select {
			case <-stage.doneCh:
				// Success
			default:
				t.Errorf("taskStage.status with terminal status %s should signal doneCh", status)
			}
		})
	}
}

func TestTaskStageStatusDoesNotSignalNonTerminalStatus(t *testing.T) {
	stage := &taskStage{
		stage:  &runtime.Stage{},
		jobs:   make(map[string]*jobSync),
		doneCh: make(chan struct{}, 1),
	}
	
	// Test that non-terminal status does not signal the channel
	nonTerminalStatuses := []string{
		common.BuildStatusRunning,
		common.BuildStatusPending,
	}
	
	for _, status := range nonTerminalStatuses {
		t.Run("status_"+status, func(t *testing.T) {
			// Reset the channel
			stage.doneCh = make(chan struct{}, 1)
			
			stage.status(status, "", "")
			
			// Channel should NOT have been signaled
			select {
			case <-stage.doneCh:
				t.Errorf("taskStage.status with non-terminal status %s should not signal doneCh", status)
			default:
				// Success - channel was not signaled
			}
		})
	}
}

func TestUpJobHandlesNilJobDoneCh(t *testing.T) {
	egn := &BuildEngine{}
	bd := &runtime.Build{Id: "test-build"}
	bt := &BuildTask{
		egn:       egn,
		build:     bd,
		jobDoneCh: nil, // Explicitly nil
	}
	
	job := &jobSync{
		step: &runtime.Step{},
	}
	
	// Should not panic even with nil channel
	bt.UpJob(job, common.BuildStatusOk, "", 0)
}

func TestTaskStageStatusHandlesNilDoneCh(t *testing.T) {
	stage := &taskStage{
		stage:  &runtime.Stage{},
		jobs:   make(map[string]*jobSync),
		doneCh: nil, // Explicitly nil
	}
	
	// Should not panic even with nil channel
	stage.status(common.BuildStatusOk, "", "")
}
