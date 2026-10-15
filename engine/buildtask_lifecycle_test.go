package engine

import (
	"context"
	"testing"
	"time"

	"github.com/gokins/core/common"
	"github.com/gokins/core/runtime"
	"github.com/gokins/runner/runners"
)

// ---------- BuildTask.status / stop / Cancel / WorkProgress ----------

func TestBuildTask_StatusSetsEventWhenProvided(t *testing.T) {
	bt := &BuildTask{
		build: &runtime.Build{Id: "b1"},
	}
	bt.status(common.BuildStatusError, "boom", common.BuildEventGetRepo)
	if bt.build.Status != common.BuildStatusError {
		t.Errorf("expected status %q, got %q", common.BuildStatusError, bt.build.Status)
	}
	if bt.build.Error != "boom" {
		t.Errorf("expected error %q, got %q", "boom", bt.build.Error)
	}
	if bt.build.Event != common.BuildEventGetRepo {
		t.Errorf("expected event %q, got %q", common.BuildEventGetRepo, bt.build.Event)
	}
}

func TestBuildTask_StatusWithoutEvent(t *testing.T) {
	bt := &BuildTask{
		build: &runtime.Build{Id: "b1", Event: "keep-me"},
	}
	bt.status(common.BuildStatusRunning, "go")
	if bt.build.Event != "keep-me" {
		t.Errorf("expected event to be preserved when not provided, got %q", bt.build.Event)
	}
	if bt.build.Status != common.BuildStatusRunning {
		t.Errorf("expected status %q, got %q", common.BuildStatusRunning, bt.build.Status)
	}
}

func TestBuildTask_StopdWithNilContext(t *testing.T) {
	bt := &BuildTask{build: &runtime.Build{Id: "b1"}}
	if !bt.stopd() {
		t.Error("expected stopd() to return true when ctx is nil")
	}
}

func TestBuildTask_StopdWithActiveContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	bt := &BuildTask{build: &runtime.Build{Id: "b1"}, ctx: ctx}
	if bt.stopd() {
		t.Error("expected stopd() to return false for active context")
	}
	cancel()
	if !bt.stopd() {
		t.Error("expected stopd() to return true after cancel")
	}
}

func TestBuildTask_StopAndCancel(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	bt := &BuildTask{
		build:     &runtime.Build{Id: "b1"},
		ctx:       ctx,
		cncl:      cancel,
		ctrlendtm: time.Now(),
	}
	bt.stop()
	if !bt.ctrlendtm.IsZero() {
		t.Error("expected ctrlendtm to be zeroed by stop()")
	}
	select {
	case <-ctx.Done():
		// OK
	default:
		t.Error("expected context to be canceled by stop()")
	}
}

func TestBuildTask_CancelSetsCtrlEndTime(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	bt := &BuildTask{
		build: &runtime.Build{Id: "b1"},
		ctx:   ctx,
		cncl:  cancel,
	}
	before := time.Now()
	bt.Cancel()
	if bt.ctrlendtm.Before(before) {
		t.Error("expected ctrlendtm to be set to roughly now")
	}
	select {
	case <-ctx.Done():
	default:
		t.Error("expected context to be canceled by Cancel()")
	}
}

func TestBuildTask_WorkProgress(t *testing.T) {
	bt := &BuildTask{build: &runtime.Build{Id: "b1"}, workpgss: 42}
	if bt.WorkProgress() != 42 {
		t.Errorf("expected 42, got %d", bt.WorkProgress())
	}
}

// ---------- BuildTask.Write parses git progress ----------

func TestBuildTask_WriteParsesProgress(t *testing.T) {
	bt := &BuildTask{build: &runtime.Build{Id: "b1"}}
	line := "Receiving objects:  50% (5/10)\n"
	n, err := bt.Write([]byte(line))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if n != len(line) {
		t.Errorf("expected n=%d, got %d", len(line), n)
	}
	// workpgss = int(float64(50) * 0.8) = 40
	if bt.workpgss != 40 {
		t.Errorf("expected workpgss=40, got %d", bt.workpgss)
	}
}

func TestBuildTask_WriteIgnoresNonMatching(t *testing.T) {
	bt := &BuildTask{build: &runtime.Build{Id: "b1"}, workpgss: 7}
	n, err := bt.Write([]byte("Cloning into 'foo'...\n"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if n == 0 {
		t.Error("expected non-zero byte count returned")
	}
	if bt.workpgss != 7 {
		t.Errorf("expected workpgss unchanged (7), got %d", bt.workpgss)
	}
}

func TestBuildTask_WriteEmpty(t *testing.T) {
	bt := &BuildTask{build: &runtime.Build{Id: "b1"}, workpgss: 10}
	n, err := bt.Write([]byte{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if n != 0 {
		t.Errorf("expected 0 bytes written, got %d", n)
	}
	if bt.workpgss != 10 {
		t.Errorf("expected workpgss unchanged (10), got %d", bt.workpgss)
	}
}

func TestBuildTask_WriteMultipleProgressLines(t *testing.T) {
	bt := &BuildTask{build: &runtime.Build{Id: "b1"}}
	// Write() calls FindAllStringSubmatch(line,-1)[0] -- it takes the *first*
	// regex match in the full input, NOT the last. So 10% wins.
	input := "Receiving objects:  10% (1/10)\nReceiving objects:  90% (9/10)\n"
	n, err := bt.Write([]byte(input))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if n != len(input) {
		t.Errorf("expected n=%d, got %d", len(input), n)
	}
	// First match: int(float64(10) * 0.8) = 8
	if bt.workpgss != 8 {
		t.Errorf("expected workpgss=8 (first match wins), got %d", bt.workpgss)
	}
}

// ---------- regBfb regex sanity ----------

func TestRegBfbMatches(t *testing.T) {
	cases := []struct {
		line string
		want string
	}{
		{"Receiving objects:  50% (5/10)", "50"},
		{"Resolving deltas: 100% (3/3)", "100"},
		{"no match here", ""},
		{": 5% (1/20)", "5"},
	}
	for _, tc := range cases {
		if !regBfb.MatchString(tc.line) && tc.want != "" {
			t.Errorf("expected regex to match %q", tc.line)
			continue
		}
		if regBfb.MatchString(tc.line) {
			subs := regBfb.FindAllStringSubmatch(tc.line, -1)[0]
			if len(subs) < 2 || subs[1] != tc.want {
				t.Errorf("expected %q, got %v", tc.want, subs)
			}
		}
	}
}

// ---------- taskStage.status ----------

func TestTaskStage_StatusSetsEvent(t *testing.T) {
	stg := &taskStage{
		stage: &runtime.Stage{Id: "s1"},
		jobs:  make(map[string]*jobSync),
	}
	stg.status(common.BuildStatusError, "oops", "evt1")
	if stg.stage.Status != common.BuildStatusError {
		t.Errorf("expected %q, got %q", common.BuildStatusError, stg.stage.Status)
	}
	if stg.stage.Error != "oops" {
		t.Errorf("expected %q, got %q", "oops", stg.stage.Error)
	}
	if stg.stage.Event != "evt1" {
		t.Errorf("expected %q, got %q", "evt1", stg.stage.Event)
	}
}

func TestTaskStage_StatusWithoutEvent(t *testing.T) {
	stg := &taskStage{
		stage: &runtime.Stage{Id: "s1", Event: "keep"},
		jobs:  make(map[string]*jobSync),
	}
	stg.status(common.BuildStatusOk, "fine")
	if stg.stage.Event != "keep" {
		t.Errorf("expected event preserved, got %q", stg.stage.Event)
	}
}

// ---------- jobSync.status ----------

func TestJobSync_StatusSetsEvent(t *testing.T) {
	js := &jobSync{step: &runtime.Step{Id: "st1"}}
	js.status(common.BuildStatusRunning, "busy", "evt")
	if js.step.Status != common.BuildStatusRunning {
		t.Errorf("expected %q, got %q", common.BuildStatusRunning, js.step.Status)
	}
	if js.step.Error != "busy" {
		t.Errorf("expected %q, got %q", "busy", js.step.Error)
	}
	if js.step.Event != "evt" {
		t.Errorf("expected %q, got %q", "evt", js.step.Event)
	}
}

func TestJobSync_StatusWithoutEvent(t *testing.T) {
	js := &jobSync{step: &runtime.Step{Id: "st1", Event: "keep"}}
	js.status(common.BuildStatusOk, "")
	if js.step.Event != "keep" {
		t.Errorf("expected event preserved, got %q", js.step.Event)
	}
}

// ---------- BuildTask.GetJob ----------

func TestBuildTask_GetJobEmptyID(t *testing.T) {
	bt := &BuildTask{build: &runtime.Build{Id: "b1"}, jobs: make(map[string]*jobSync)}
	j, ok := bt.GetJob("")
	if ok || j != nil {
		t.Error("expected nil,false for empty job id")
	}
}

func TestBuildTask_GetJobMissing(t *testing.T) {
	bt := &BuildTask{build: &runtime.Build{Id: "b1"}, jobs: make(map[string]*jobSync)}
	j, ok := bt.GetJob("nope")
	if ok || j != nil {
		t.Error("expected nil,false for missing job")
	}
}

func TestBuildTask_GetJobFound(t *testing.T) {
	js := &jobSync{step: &runtime.Step{Id: "j1"}}
	bt := &BuildTask{
		build: &runtime.Build{Id: "b1"},
		jobs:  map[string]*jobSync{"j1": js},
	}
	got, ok := bt.GetJob("j1")
	if !ok || got != js {
		t.Error("expected found job to be returned")
	}
}

// ---------- BuildTask.UpJob / UpJobCmd edge cases ----------

func TestBuildTask_UpJobNilJobIgnored(t *testing.T) {
	bt := &BuildTask{build: &runtime.Build{Id: "b1"}}
	// Must not panic
	bt.UpJob(nil, common.BuildStatusOk, "", 0)
}

func TestBuildTask_UpJobEmptyStatusIgnored(t *testing.T) {
	js := &jobSync{step: &runtime.Step{Id: "j1"}}
	bt := &BuildTask{build: &runtime.Build{Id: "b1"}}
	bt.UpJob(js, "", "err", 1)
	if js.step.Status != "" {
		t.Errorf("expected step status untouched, got %q", js.step.Status)
	}
}

func TestBuildTask_UpJobCmdNilCmdIgnored(t *testing.T) {
	bt := &BuildTask{build: &runtime.Build{Id: "b1"}}
	// Must not panic
	bt.UpJobCmd(nil, 1, 0)
}

func TestBuildTask_UpJobCmdRunning(t *testing.T) {
	cmd := &cmdSync{cmd: &runners.CmdContent{Id: "c1"}}
	bt := &BuildTask{build: &runtime.Build{Id: "b1"}}
	before := time.Now()
	bt.UpJobCmd(cmd, 1, 0)
	if cmd.status != common.BuildStatusRunning {
		t.Errorf("expected running, got %q", cmd.status)
	}
	if cmd.started.Before(before) {
		t.Error("expected started to be set")
	}
}

func TestBuildTask_UpJobCmdSuccess(t *testing.T) {
	cmd := &cmdSync{cmd: &runners.CmdContent{Id: "c1"}}
	bt := &BuildTask{build: &runtime.Build{Id: "b1"}}
	bt.UpJobCmd(cmd, 2, 0)
	if cmd.status != common.BuildStatusOk {
		t.Errorf("expected ok, got %q", cmd.status)
	}
}

func TestBuildTask_UpJobCmdNonZeroExit(t *testing.T) {
	cmd := &cmdSync{cmd: &runners.CmdContent{Id: "c1"}}
	bt := &BuildTask{build: &runtime.Build{Id: "b1"}}
	bt.UpJobCmd(cmd, 2, 137)
	if cmd.status != common.BuildStatusError {
		t.Errorf("expected error status, got %q", cmd.status)
	}
	if cmd.code != 137 {
		t.Errorf("expected code 137, got %d", cmd.code)
	}
}

func TestBuildTask_UpJobCmdCancel(t *testing.T) {
	cmd := &cmdSync{cmd: &runners.CmdContent{Id: "c1"}}
	bt := &BuildTask{build: &runtime.Build{Id: "b1"}}
	bt.UpJobCmd(cmd, 3, 42)
	if cmd.status != common.BuildStatusCancel {
		t.Errorf("expected cancel, got %q", cmd.status)
	}
	if cmd.code != 42 {
		t.Errorf("expected code 42, got %d", cmd.code)
	}
}

func TestBuildTask_UpJobCmdError(t *testing.T) {
	cmd := &cmdSync{cmd: &runners.CmdContent{Id: "c1"}}
	bt := &BuildTask{build: &runtime.Build{Id: "b1"}}
	bt.UpJobCmd(cmd, -1, 99)
	if cmd.status != common.BuildStatusError {
		t.Errorf("expected error, got %q", cmd.status)
	}
	if cmd.code != 99 {
		t.Errorf("expected code 99, got %d", cmd.code)
	}
}

func TestBuildTask_UpJobCmdUnknownFSIgnored(t *testing.T) {
	cmd := &cmdSync{cmd: &runners.CmdContent{Id: "c1"}, status: common.BuildStatusPending}
	bt := &BuildTask{build: &runtime.Build{Id: "b1"}}
	bt.UpJobCmd(cmd, 999, 0)
	if cmd.status != common.BuildStatusPending {
		t.Errorf("expected status untouched, got %q", cmd.status)
	}
}

// ---------- baseRunner.FindJobId edge cases ----------

func TestBaseRunner_FindJobId_EmptyInputs(t *testing.T) {
	br := &baseRunner{}
	if _, ok := br.FindJobId("", "s", "st"); ok {
		t.Error("expected false for empty buildID")
	}
	if _, ok := br.FindJobId("b", "", "st"); ok {
		t.Error("expected false for empty stageName")
	}
	if _, ok := br.FindJobId("b", "s", ""); ok {
		t.Error("expected false for empty stepName")
	}
}

// ---------- BuildTask.Show on stopped task ----------

func TestBuildTask_ShowStoppedReturnsFalse(t *testing.T) {
	bt := &BuildTask{build: &runtime.Build{Id: "b1"}}
	// ctx nil => stopd() returns true
	_, ok := bt.Show()
	if ok {
		t.Error("expected Show() to return false for stopped task")
	}
}
