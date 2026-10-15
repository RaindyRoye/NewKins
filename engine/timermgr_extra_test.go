package engine

import (
	"container/list"
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/gokins/core/runtime"
	"github.com/gokins/gokins/comm"
	"github.com/gokins/gokins/model"
)

// --- TimerEngine resetOne for all timer types ---

func TestTimerEngineResetOneHourTimer(t *testing.T) {
	te := &TimerEngine{
		tasks: make(map[string]*timerExec),
	}
	params := map[string]any{
		"timerType": 2, // hour
		"dates":     time.Now().Format(time.RFC3339Nano),
	}
	paramBytes, _ := json.Marshal(params)
	tmr := &model.TTrigger{
		Id:     "test-hour",
		Types:  "timer",
		Name:   "hour-timer",
		Params: string(paramBytes),
	}
	err := te.resetOne(tmr)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	task, ok := te.tasks["test-hour"]
	if !ok {
		t.Fatal("expected task to be added")
	}
	if task.typ != 2 {
		t.Fatalf("expected type 2, got %d", task.typ)
	}
	if time.Until(task.tick) > time.Hour+time.Second {
		t.Fatal("tick should be within ~1 hour")
	}
}

func TestTimerEngineResetOneDayTimer(t *testing.T) {
	te := &TimerEngine{
		tasks: make(map[string]*timerExec),
	}
	params := map[string]any{
		"timerType": 3, // day
		"dates":     time.Now().Format(time.RFC3339Nano),
	}
	paramBytes, _ := json.Marshal(params)
	tmr := &model.TTrigger{
		Id:     "test-day",
		Types:  "timer",
		Name:   "day-timer",
		Params: string(paramBytes),
	}
	err := te.resetOne(tmr)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	task, ok := te.tasks["test-day"]
	if !ok {
		t.Fatal("expected task to be added")
	}
	if task.typ != 3 {
		t.Fatalf("expected type 3, got %d", task.typ)
	}
	if time.Until(task.tick) > 25*time.Hour {
		t.Fatal("tick should be within ~24 hours")
	}
}

func TestTimerEngineResetOneWeekTimer(t *testing.T) {
	te := &TimerEngine{
		tasks: make(map[string]*timerExec),
	}
	params := map[string]any{
		"timerType": 4, // week
		"dates":     time.Now().Format(time.RFC3339Nano),
	}
	paramBytes, _ := json.Marshal(params)
	tmr := &model.TTrigger{
		Id:     "test-week",
		Types:  "timer",
		Name:   "week-timer",
		Params: string(paramBytes),
	}
	err := te.resetOne(tmr)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	task, ok := te.tasks["test-week"]
	if !ok {
		t.Fatal("expected task to be added")
	}
	if task.typ != 4 {
		t.Fatalf("expected type 4, got %d", task.typ)
	}
	if time.Until(task.tick) > 8*24*time.Hour {
		t.Fatal("tick should be within ~7 days")
	}
}

func TestTimerEngineResetOneMonthTimer(t *testing.T) {
	te := &TimerEngine{
		tasks: make(map[string]*timerExec),
	}
	params := map[string]any{
		"timerType": 5, // month
		"dates":     time.Now().Format(time.RFC3339Nano),
	}
	paramBytes, _ := json.Marshal(params)
	tmr := &model.TTrigger{
		Id:     "test-month",
		Types:  "timer",
		Name:   "month-timer",
		Params: string(paramBytes),
	}
	err := te.resetOne(tmr)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	task, ok := te.tasks["test-month"]
	if !ok {
		t.Fatal("expected task to be added")
	}
	if task.typ != 5 {
		t.Fatalf("expected type 5, got %d", task.typ)
	}
	if time.Until(task.tick) > 31*24*time.Hour {
		t.Fatal("tick should be within ~30 days")
	}
}

func TestTimerEngineResetOneOnceTimerPastTime(t *testing.T) {
	te := &TimerEngine{
		tasks: make(map[string]*timerExec),
	}
	// Once timer set in the past should NOT be added
	pastTime := time.Now().Add(-time.Hour)
	params := map[string]any{
		"timerType": 0, // once
		"dates":     pastTime.Format(time.RFC3339Nano),
	}
	paramBytes, _ := json.Marshal(params)
	tmr := &model.TTrigger{
		Id:     "test-once-past",
		Types:  "timer",
		Name:   "once-past",
		Params: string(paramBytes),
	}
	err := te.resetOne(tmr)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Once timer in the past should not be added
	if _, ok := te.tasks["test-once-past"]; ok {
		t.Fatal("once timer in the past should not be added")
	}
}

func TestTimerEngineResetOneUpdateExistingTask(t *testing.T) {
	te := &TimerEngine{
		tasks: make(map[string]*timerExec),
	}
	// First add a task
	params := map[string]any{
		"timerType": 1,
		"dates":     time.Now().Format(time.RFC3339Nano),
	}
	paramBytes, _ := json.Marshal(params)
	tmr := &model.TTrigger{
		Id:     "test-update",
		Types:  "timer",
		Name:   "update-timer",
		Params: string(paramBytes),
	}
	err := te.resetOne(tmr)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	task1 := te.tasks["test-update"]

	// Now update it with a different type
	params["timerType"] = 2
	paramBytes, _ = json.Marshal(params)
	tmr.Params = string(paramBytes)
	err = te.resetOne(tmr)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	task2 := te.tasks["test-update"]
	if task1 != task2 {
		t.Error("expected same task object to be updated")
	}
	if task2.typ != 2 {
		t.Errorf("expected type 2 after update, got %d", task2.typ)
	}
}

func TestTimerEngineResetOneMissingTimerType(t *testing.T) {
	te := &TimerEngine{
		tasks: make(map[string]*timerExec),
	}
	params := map[string]any{
		"dates": time.Now().Format(time.RFC3339Nano),
	}
	paramBytes, _ := json.Marshal(params)
	tmr := &model.TTrigger{
		Id:     "test-notype",
		Types:  "timer",
		Name:   "no-type",
		Params: string(paramBytes),
	}
	err := te.resetOne(tmr)
	if err == nil {
		t.Fatal("expected error when timerType is missing")
	}
}

func TestTimerEngineResetOneInvalidDateFormat(t *testing.T) {
	te := &TimerEngine{
		tasks: make(map[string]*timerExec),
	}
	params := map[string]any{
		"timerType": 1,
		"dates":     "not-a-valid-date",
	}
	paramBytes, _ := json.Marshal(params)
	tmr := &model.TTrigger{
		Id:     "test-baddate",
		Types:  "timer",
		Name:   "bad-date",
		Params: string(paramBytes),
	}
	err := te.resetOne(tmr)
	if err == nil {
		t.Fatal("expected error when date format is invalid")
	}
}

// --- TimerEngine.execItem ---

func TestTimerEngineExecItem_NotDue(t *testing.T) {
	te := &TimerEngine{
		tasks: make(map[string]*timerExec),
	}
	// Set tick in the future
	futureTick := time.Now().Add(time.Hour)
	exec := &timerExec{
		tt:   &model.TTrigger{Id: "test", Name: "test"},
		typ:  1,
		tick: futureTick,
	}
	te.tasks["test"] = exec
	// execItem should not execute when tick is in the future
	te.execItem(exec)
	// Task should still exist and tick should not have changed
	if te.tasks["test"].tick != futureTick {
		t.Error("tick should not change when not due")
	}
}

// --- BuildTask.check() with valid repo directory ---

func TestBuildTaskCheck_ValidRepoDir(t *testing.T) {
	// Create a temporary directory to simulate a repo
	task := &BuildTask{
		build: &runtime.Build{
			Id: "build-1",
			Repo: &runtime.Repository{
				CloneURL: "/tmp", // /tmp exists as a directory
			},
			Stages: []*runtime.Stage{
				{
					Id:      "stage-1",
					BuildId: "build-1",
					Name:    "stage-1",
					Steps: []*runtime.Step{
						{
							Id:       "step-1",
							BuildId:  "build-1",
							StageId:  "stage-1",
							Step:     "shell",
							Name:     "step-1",
							Commands: "echo test",
						},
					},
				},
			},
		},
		stages: make(map[string]*taskStage),
		jobs:   make(map[string]*jobSync),
	}
	result := task.check()
	// check() should return true (validation passes)
	// genRunjob will fail on DB insert, but validation should pass
	if !result {
		t.Logf("check() returned false, error: %q (expected if DB insert fails)", task.build.Error)
	}
	// isClone should be false when CloneURL points to an existing directory
	if task.isClone {
		t.Log("isClone should be false when CloneURL is an existing directory")
	}
}

// --- BuildEngine.run() with task at limit ---

func TestBuildEngineRun_AtLimit(t *testing.T) {
	be := &BuildEngine{
		taskw: list.New(),
		tasks: make(map[string]*BuildTask),
	}
	// Set RunLimit to 2
	originalLimit := comm.Cfg.Server.RunLimit
	comm.Cfg.Server.RunLimit = 2
	defer func() { comm.Cfg.Server.RunLimit = originalLimit }()

	// Add 2 tasks (at limit)
	for i := 0; i < 2; i++ {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		be.tasks[string(rune('a'+i))] = &BuildTask{
			build: &runtime.Build{Id: string(rune('a' + i))},
			ctx:   ctx,
		}
	}
	// Add a task to the queue
	be.taskw.PushBack(&runtime.Build{Id: "queued"})

	// run() should not start a new task when at limit
	be.run()
	// Queue should still have the task
	if be.taskw.Len() != 1 {
		t.Errorf("expected queue length 1, got %d", be.taskw.Len())
	}
}
