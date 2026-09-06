package engine

import (
	"context"
	"testing"

	"github.com/gokins/core/runtime"
	"github.com/gokins/gokins/comm"
	"github.com/gokins/runner/runners"
	"github.com/stretchr/testify/assert"
)

// TestUpdateBuild_NilDb verifies that updateBuild handles nil DB gracefully
// without panicking. This is a regression test for the panic/recover abuse
// where functions relied on RecoverLog to catch nil pointer dereferences.
func TestUpdateBuild_NilDb(t *testing.T) {
	// Ensure comm.Db is nil for this test
	origDb := comm.Db
	comm.Db = nil
	defer func() {
		comm.Db = origDb
	}()

	bt := &BuildTask{
		ctx: context.Background(),
	}

	// Should not panic
	assert.NotPanics(t, func() {
		bt.updateBuild(&runtime.Build{
			Id: "test-build",
		})
	})
}

// TestUpdateStage_NilDb verifies that updateStage handles nil DB gracefully
func TestUpdateStage_NilDb(t *testing.T) {
	origDb := comm.Db
	comm.Db = nil
	defer func() {
		comm.Db = origDb
	}()

	bt := &BuildTask{
		ctx: context.Background(),
	}

	assert.NotPanics(t, func() {
		bt.updateStage(&runtime.Stage{
			Id: "test-stage",
		})
	})
}

// TestUpdateStep_NilDb verifies that updateStep handles nil DB gracefully
func TestUpdateStep_NilDb(t *testing.T) {
	origDb := comm.Db
	comm.Db = nil
	defer func() {
		comm.Db = origDb
	}()

	bt := &BuildTask{
		ctx: context.Background(),
	}

	job := &jobSync{
		step: &runtime.Step{
			Id: "test-step",
		},
	}

	assert.NotPanics(t, func() {
		bt.updateStep(job)
	})
}

// TestUpdateStepCmd_NilDb verifies that updateStepCmd handles nil DB gracefully
func TestUpdateStepCmd_NilDb(t *testing.T) {
	origDb := comm.Db
	comm.Db = nil
	defer func() {
		comm.Db = origDb
	}()

	bt := &BuildTask{
		ctx: context.Background(),
	}

	cmd := &cmdSync{
		cmd: &runners.CmdContent{
			Id: "test-cmd",
		},
	}

	assert.NotPanics(t, func() {
		bt.updateStepCmd(cmd)
	})
}
