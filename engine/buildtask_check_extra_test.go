package engine

import (
	"testing"

	"github.com/gokins/core/common"
	"github.com/gokins/core/runtime"
)

// --- check() edge case tests ---

func TestBuildTaskCheck_DuplicateStageNames(t *testing.T) {
	buildId := "build-1"
	stageId := "stage-1"
	step := &runtime.Step{
		Id:      "step-1",
		BuildId: buildId,
		StageId: stageId,
		Step:    "shell@ssh",
		Name:    "s1",
	}
	stages := []*runtime.Stage{
		{
			Id:      stageId,
			BuildId: buildId,
			Name:    "build",
			Steps:   []*runtime.Step{step},
		},
		{
			Id:      "stage-2",
			BuildId: buildId,
			Name:    "build", // duplicate name
			Steps:   []*runtime.Step{{Id: "step-2", BuildId: buildId, StageId: "stage-2", Step: "shell@ssh", Name: "s2"}},
		},
	}
	task := &BuildTask{
		build: &runtime.Build{
			Id:     buildId,
			Repo:   &runtime.Repository{CloneURL: ""},
			Stages: stages,
		},
		stages: make(map[string]*taskStage),
		jobs:   make(map[string]*jobSync),
	}
	if task.check() {
		t.Fatal("expected check() to return false for duplicate stage names")
	}
	if task.build.Error != "build Stages.build is repeat" {
		t.Errorf("expected 'build Stages.build is repeat', got %q", task.build.Error)
	}
}

func TestBuildTaskCheck_StageBuildIdMismatch(t *testing.T) {
	task := &BuildTask{
		build: &runtime.Build{
			Id:   "build-1",
			Repo: &runtime.Repository{CloneURL: ""},
			Stages: []*runtime.Stage{
				{
					Id:      "stage-1",
					BuildId: "wrong-build", // mismatch
					Name:    "build",
					Steps: []*runtime.Step{
						{Id: "step-1", BuildId: "build-1", StageId: "stage-1", Step: "shell@ssh", Name: "s1"},
					},
				},
			},
		},
		stages: make(map[string]*taskStage),
		jobs:   make(map[string]*jobSync),
	}
	if task.check() {
		t.Fatal("expected check() to return false for stage build ID mismatch")
	}
}

func TestBuildTaskCheck_EmptyStageName(t *testing.T) {
	buildId := "build-1"
	task := &BuildTask{
		build: &runtime.Build{
			Id:   buildId,
			Repo: &runtime.Repository{CloneURL: ""},
			Stages: []*runtime.Stage{
				{
					Id:      "stage-1",
					BuildId: buildId,
					Name:    "", // empty name
					Steps: []*runtime.Step{
						{Id: "step-1", BuildId: buildId, StageId: "stage-1", Step: "shell@ssh", Name: "s1"},
					},
				},
			},
		},
		stages: make(map[string]*taskStage),
		jobs:   make(map[string]*jobSync),
	}
	if task.check() {
		t.Fatal("expected check() to return false for empty stage name")
	}
	if task.build.Error != "build Stage name is empty" {
		t.Errorf("expected 'build Stage name is empty', got %q", task.build.Error)
	}
}

func TestBuildTaskCheck_StageWithEmptySteps(t *testing.T) {
	buildId := "build-1"
	task := &BuildTask{
		build: &runtime.Build{
			Id:   buildId,
			Repo: &runtime.Repository{CloneURL: ""},
			Stages: []*runtime.Stage{
				{
					Id:      "stage-1",
					BuildId: buildId,
					Name:    "build",
					Steps:   []*runtime.Step{}, // empty steps
				},
			},
		},
		stages: make(map[string]*taskStage),
		jobs:   make(map[string]*jobSync),
	}
	if task.check() {
		t.Fatal("expected check() to return false for stage with empty steps")
	}
	if task.build.Error != "build Stages is empty" {
		t.Errorf("expected 'build Stages is empty', got %q", task.build.Error)
	}
}

func TestBuildTaskCheck_StepBuildIdMismatch(t *testing.T) {
	buildId := "build-1"
	task := &BuildTask{
		build: &runtime.Build{
			Id:   buildId,
			Repo: &runtime.Repository{CloneURL: ""},
			Stages: []*runtime.Stage{
				{
					Id:      "stage-1",
					BuildId: buildId,
					Name:    "build",
					Steps: []*runtime.Step{
						{
							Id:      "step-1",
							BuildId: "wrong-build", // mismatch
							StageId: "stage-1",
							Step:    "shell@ssh",
							Name:    "s1",
						},
					},
				},
			},
		},
		stages: make(map[string]*taskStage),
		jobs:   make(map[string]*jobSync),
	}
	if task.check() {
		t.Fatal("expected check() to return false for step build ID mismatch")
	}
}

func TestBuildTaskCheck_StepStageIdMismatch(t *testing.T) {
	buildId := "build-1"
	task := &BuildTask{
		build: &runtime.Build{
			Id:   buildId,
			Repo: &runtime.Repository{CloneURL: ""},
			Stages: []*runtime.Stage{
				{
					Id:      "stage-1",
					BuildId: buildId,
					Name:    "build",
					Steps: []*runtime.Step{
						{
							Id:      "step-1",
							BuildId: buildId,
							StageId: "wrong-stage", // mismatch
							Step:    "shell@ssh",
							Name:    "s1",
						},
					},
				},
			},
		},
		stages: make(map[string]*taskStage),
		jobs:   make(map[string]*jobSync),
	}
	if task.check() {
		t.Fatal("expected check() to return false for step stage ID mismatch")
	}
}

func TestBuildTaskCheck_EmptyStepPlugin(t *testing.T) {
	buildId := "build-1"
	task := &BuildTask{
		build: &runtime.Build{
			Id:   buildId,
			Repo: &runtime.Repository{CloneURL: ""},
			Stages: []*runtime.Stage{
				{
					Id:      "stage-1",
					BuildId: buildId,
					Name:    "build",
					Steps: []*runtime.Step{
						{
							Id:      "step-1",
							BuildId: buildId,
							StageId: "stage-1",
							Step:    "", // empty plugin
							Name:    "s1",
						},
					},
				},
			},
		},
		stages: make(map[string]*taskStage),
		jobs:   make(map[string]*jobSync),
	}
	if task.check() {
		t.Fatal("expected check() to return false for empty step plugin")
	}
	if task.build.Error != "build Step Plugin is empty" {
		t.Errorf("expected 'build Step Plugin is empty', got %q", task.build.Error)
	}
}

func TestBuildTaskCheck_EmptyStepName(t *testing.T) {
	buildId := "build-1"
	task := &BuildTask{
		build: &runtime.Build{
			Id:   buildId,
			Repo: &runtime.Repository{CloneURL: ""},
			Stages: []*runtime.Stage{
				{
					Id:      "stage-1",
					BuildId: buildId,
					Name:    "build",
					Steps: []*runtime.Step{
						{
							Id:      "step-1",
							BuildId: buildId,
							StageId: "stage-1",
							Step:    "shell@ssh",
							Name:    "", // empty name
						},
					},
				},
			},
		},
		stages: make(map[string]*taskStage),
		jobs:   make(map[string]*jobSync),
	}
	if task.check() {
		t.Fatal("expected check() to return false for empty step name")
	}
	if task.build.Error != "build Step name is empty" {
		t.Errorf("expected 'build Step name is empty', got %q", task.build.Error)
	}
}

func TestBuildTaskCheck_DuplicateStepNames(t *testing.T) {
	buildId := "build-1"
	stageId := "stage-1"
	task := &BuildTask{
		build: &runtime.Build{
			Id:   buildId,
			Repo: &runtime.Repository{CloneURL: ""},
			Stages: []*runtime.Stage{
				{
					Id:      stageId,
					BuildId: buildId,
					Name:    "build",
					Steps: []*runtime.Step{
						{Id: "step-1", BuildId: buildId, StageId: stageId, Step: "shell@ssh", Name: "s1"},
						{Id: "step-2", BuildId: buildId, StageId: stageId, Step: "shell@ssh", Name: "s1"}, // duplicate
					},
				},
			},
		},
		stages: make(map[string]*taskStage),
		jobs:   make(map[string]*jobSync),
	}
	if task.check() {
		t.Fatal("expected check() to return false for duplicate step names")
	}
	if task.build.Error != "build Job.s1 is repeat" {
		t.Errorf("expected 'build Job.s1 is repeat', got %q", task.build.Error)
	}
}

// --- Write() progress parsing tests ---

func TestBuildTaskWrite_HundredPercent(t *testing.T) {
	task := &BuildTask{workpgss: 0}
	input := ": 100% (500/500)\n"
	_, _ = task.Write([]byte(input))
	expected := int(float64(100) * 0.8)
	if task.workpgss != expected {
		t.Errorf("expected workpgss=%d, got %d", expected, task.workpgss)
	}
}

// --- UpJob tests ---

func TestBuildTaskUpJob_NilJob(t *testing.T) {
	task := &BuildTask{build: &runtime.Build{Id: "b1"}}
	// Should not panic
	task.UpJob(nil, common.BuildStatusOk, "", 0)
}

func TestBuildTaskUpJob_EmptyStatus(t *testing.T) {
	job := &jobSync{
		step:  &runtime.Step{Id: "step-1"},
		cmdmp: make(map[string]*cmdSync),
	}
	task := &BuildTask{build: &runtime.Build{Id: "b1"}}
	// Should return early without modifying job
	task.UpJob(job, "", "", 0)
	if job.step.Status != "" {
		t.Errorf("expected status unchanged, got %q", job.step.Status)
	}
}

func TestBuildTaskUpJob_SetsStatus(t *testing.T) {
	job := &jobSync{
		step:  &runtime.Step{Id: "step-1"},
		cmdmp: make(map[string]*cmdSync),
	}
	task := &BuildTask{build: &runtime.Build{Id: "b1"}}
	task.UpJob(job, common.BuildStatusError, "something failed", 1)
	job.RLock()
	defer job.RUnlock()
	if job.step.Status != common.BuildStatusError {
		t.Errorf("expected status %q, got %q", common.BuildStatusError, job.step.Status)
	}
	if job.step.Error != "something failed" {
		t.Errorf("expected error 'something failed', got %q", job.step.Error)
	}
	if job.step.ExitCode != 1 {
		t.Errorf("expected exit code 1, got %d", job.step.ExitCode)
	}
}

// --- UpJobCmd tests ---

func TestBuildTaskUpJobCmd_NilCmd(t *testing.T) {
	task := &BuildTask{build: &runtime.Build{Id: "b1"}}
	// Should not panic
	task.UpJobCmd(nil, 1, 0)
}

func TestBuildTaskUpJobCmd_StartRunning(t *testing.T) {
	cmd := &cmdSync{status: common.BuildStatusPending}
	task := &BuildTask{build: &runtime.Build{Id: "b1"}}
	task.UpJobCmd(cmd, 1, 0) // fs=1 -> running
	cmd.RLock()
	defer cmd.RUnlock()
	if cmd.status != common.BuildStatusRunning {
		t.Errorf("expected status %q, got %q", common.BuildStatusRunning, cmd.status)
	}
	if cmd.started.IsZero() {
		t.Error("expected started time to be set")
	}
}

func TestBuildTaskUpJobCmd_FinishOk(t *testing.T) {
	cmd := &cmdSync{status: common.BuildStatusRunning}
	task := &BuildTask{build: &runtime.Build{Id: "b1"}}
	task.UpJobCmd(cmd, 2, 0) // fs=2, code=0 -> ok
	cmd.RLock()
	defer cmd.RUnlock()
	if cmd.status != common.BuildStatusOk {
		t.Errorf("expected status %q, got %q", common.BuildStatusOk, cmd.status)
	}
	if cmd.finished.IsZero() {
		t.Error("expected finished time to be set")
	}
}

func TestBuildTaskUpJobCmd_FinishWithError(t *testing.T) {
	cmd := &cmdSync{status: common.BuildStatusRunning}
	task := &BuildTask{build: &runtime.Build{Id: "b1"}}
	task.UpJobCmd(cmd, 2, 127) // fs=2, code=127 -> error
	cmd.RLock()
	defer cmd.RUnlock()
	if cmd.status != common.BuildStatusError {
		t.Errorf("expected status %q, got %q", common.BuildStatusError, cmd.status)
	}
	if cmd.code != 127 {
		t.Errorf("expected code 127, got %d", cmd.code)
	}
}

func TestBuildTaskUpJobCmd_Cancel(t *testing.T) {
	cmd := &cmdSync{status: common.BuildStatusRunning}
	task := &BuildTask{build: &runtime.Build{Id: "b1"}}
	task.UpJobCmd(cmd, 3, 0) // fs=3 -> cancel
	cmd.RLock()
	defer cmd.RUnlock()
	if cmd.status != common.BuildStatusCancel {
		t.Errorf("expected status %q, got %q", common.BuildStatusCancel, cmd.status)
	}
}

func TestBuildTaskUpJobCmd_Error(t *testing.T) {
	cmd := &cmdSync{status: common.BuildStatusRunning}
	task := &BuildTask{build: &runtime.Build{Id: "b1"}}
	task.UpJobCmd(cmd, -1, 42) // fs=-1 -> error
	cmd.RLock()
	defer cmd.RUnlock()
	if cmd.status != common.BuildStatusError {
		t.Errorf("expected status %q, got %q", common.BuildStatusError, cmd.status)
	}
	if cmd.code != 42 {
		t.Errorf("expected code 42, got %d", cmd.code)
	}
}

func TestBuildTaskUpJobCmd_UnknownFs(t *testing.T) {
	cmd := &cmdSync{status: common.BuildStatusPending}
	task := &BuildTask{build: &runtime.Build{Id: "b1"}}
	task.UpJobCmd(cmd, 99, 0) // unknown fs -> no change
	cmd.RLock()
	defer cmd.RUnlock()
	if cmd.status != common.BuildStatusPending {
		t.Errorf("expected status unchanged, got %q", cmd.status)
	}
}

// --- getRepo tests ---

func TestBuildTaskGetRepo_NotClone(t *testing.T) {
	task := &BuildTask{
		build:   &runtime.Build{Id: "b1"},
		isClone: false,
	}
	err := task.getRepo()
	if err != nil {
		t.Fatalf("expected nil error when isClone=false, got: %v", err)
	}
}

// --- status methods ---

func TestBuildTaskStatus_SetsFields(t *testing.T) {
	bd := &runtime.Build{Id: "b1"}
	task := &BuildTask{build: bd}
	task.status(common.BuildStatusRunning, "", "running-event")
	if bd.Status != common.BuildStatusRunning {
		t.Errorf("expected status %q, got %q", common.BuildStatusRunning, bd.Status)
	}
	if bd.Event != "running-event" {
		t.Errorf("expected event 'running-event', got %q", bd.Event)
	}
}

func TestBuildTaskStatus_ErrorOnly(t *testing.T) {
	bd := &runtime.Build{Id: "b1"}
	task := &BuildTask{build: bd}
	task.status(common.BuildStatusError, "something broke")
	if bd.Error != "something broke" {
		t.Errorf("expected error 'something broke', got %q", bd.Error)
	}
	if bd.Status != common.BuildStatusError {
		t.Errorf("expected status %q, got %q", common.BuildStatusError, bd.Status)
	}
}

func TestTaskStageStatus_SetsFields(t *testing.T) {
	stg := &runtime.Stage{Id: "s1"}
	ts := &taskStage{stage: stg, jobs: make(map[string]*jobSync)}
	ts.status(common.BuildStatusOk, "", "stage-done")
	if stg.Status != common.BuildStatusOk {
		t.Errorf("expected status %q, got %q", common.BuildStatusOk, stg.Status)
	}
	if stg.Event != "stage-done" {
		t.Errorf("expected event 'stage-done', got %q", stg.Event)
	}
}

func TestJobSyncStatus_SetsFields(t *testing.T) {
	step := &runtime.Step{Id: "step-1"}
	job := &jobSync{step: step}
	job.status(common.BuildStatusCancel, "cancelled", "cancel-event")
	if step.Status != common.BuildStatusCancel {
		t.Errorf("expected status %q, got %q", common.BuildStatusCancel, step.Status)
	}
	if step.Error != "cancelled" {
		t.Errorf("expected error 'cancelled', got %q", step.Error)
	}
	if step.Event != "cancel-event" {
		t.Errorf("expected event 'cancel-event', got %q", step.Event)
	}
}

// --- stopd / Cancel / WorkProgress ---

func TestBuildTaskStopd_NilCtx(t *testing.T) {
	task := &BuildTask{build: &runtime.Build{Id: "b1"}}
	// ctx is nil -> stopd() should return true
	if !task.stopd() {
		t.Error("expected stopd() to return true when ctx is nil")
	}
}

func TestBuildTaskCancel_SetsCtrlEnd(t *testing.T) {
	task := &BuildTask{build: &runtime.Build{Id: "b1"}}
	if !task.ctrlendtm.IsZero() {
		t.Error("expected ctrlendtm to be zero initially")
	}
	task.Cancel()
	if task.ctrlendtm.IsZero() {
		t.Error("expected ctrlendtm to be set after Cancel()")
	}
}

func TestBuildTaskWorkProgress_Default(t *testing.T) {
	task := &BuildTask{build: &runtime.Build{Id: "b1"}}
	if task.WorkProgress() != 0 {
		t.Errorf("expected WorkProgress()=0, got %d", task.WorkProgress())
	}
}

// --- NewBuildTask ---

func TestNewBuildTask_Fields(t *testing.T) {
	egn := &BuildEngine{}
	bd := &runtime.Build{Id: "b1", PipelineId: "p1"}
	task := NewBuildTask(egn, bd)
	if task.egn != egn {
		t.Error("expected egn to be set")
	}
	if task.build != bd {
		t.Error("expected build to be set")
	}
	if task.build.Id != "b1" {
		t.Errorf("expected build.Id='b1', got %q", task.build.Id)
	}
}
