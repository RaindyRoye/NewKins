package engine

import (
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/gokins/gokins/model"
)

// Test additional timer types (hourly, daily, weekly, monthly) for coverage.

func TestTimerEngineResetOneHourlyTimer(t *testing.T) {
	te := &TimerEngine{tasks: make(map[string]*timerExec)}
	params := map[string]any{
		"timerType": 2, // hourly
		"dates":     time.Now().Format(time.RFC3339Nano),
	}
	b, _ := json.Marshal(params)
	tmr := &model.TTrigger{Id: "hourly-1", Types: "timer", Name: "hourly", Params: string(b)}
	if err := te.resetOne(tmr); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	task, ok := te.tasks["hourly-1"]
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

func TestTimerEngineResetOneDailyTimer(t *testing.T) {
	te := &TimerEngine{tasks: make(map[string]*timerExec)}
	params := map[string]any{
		"timerType": 3, // daily
		"dates":     time.Now().Format(time.RFC3339Nano),
	}
	b, _ := json.Marshal(params)
	tmr := &model.TTrigger{Id: "daily-1", Types: "timer", Name: "daily", Params: string(b)}
	if err := te.resetOne(tmr); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	task, ok := te.tasks["daily-1"]
	if !ok {
		t.Fatal("expected task to be added")
	}
	if task.typ != 3 {
		t.Fatalf("expected type 3, got %d", task.typ)
	}
}

func TestTimerEngineResetOneWeeklyTimer(t *testing.T) {
	te := &TimerEngine{tasks: make(map[string]*timerExec)}
	params := map[string]any{
		"timerType": 4, // weekly
		"dates":     time.Now().Format(time.RFC3339Nano),
	}
	b, _ := json.Marshal(params)
	tmr := &model.TTrigger{Id: "weekly-1", Types: "timer", Name: "weekly", Params: string(b)}
	if err := te.resetOne(tmr); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	task, ok := te.tasks["weekly-1"]
	if !ok {
		t.Fatal("expected task to be added")
	}
	if task.typ != 4 {
		t.Fatalf("expected type 4, got %d", task.typ)
	}
}

func TestTimerEngineResetOneMonthlyTimer(t *testing.T) {
	te := &TimerEngine{tasks: make(map[string]*timerExec)}
	params := map[string]any{
		"timerType": 5, // monthly
		"dates":     time.Now().Format(time.RFC3339Nano),
	}
	b, _ := json.Marshal(params)
	tmr := &model.TTrigger{Id: "monthly-1", Types: "timer", Name: "monthly", Params: string(b)}
	if err := te.resetOne(tmr); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	task, ok := te.tasks["monthly-1"]
	if !ok {
		t.Fatal("expected task to be added")
	}
	if task.typ != 5 {
		t.Fatalf("expected type 5, got %d", task.typ)
	}
}

func TestTimerEngineResetOneInvalidTimerType(t *testing.T) {
	te := &TimerEngine{tasks: make(map[string]*timerExec)}
	params := map[string]any{
		"timerType": 99, // unknown
		"dates":     time.Now().Format(time.RFC3339Nano),
	}
	b, _ := json.Marshal(params)
	tmr := &model.TTrigger{Id: "bad-1", Types: "timer", Name: "bad", Params: string(b)}
	err := te.resetOne(tmr)
	if err != nil {
		t.Fatalf("unexpected error for unknown type (should be ignored): %v", err)
	}
	// Unknown type should not add a task
	if _, ok := te.tasks["bad-1"]; ok {
		t.Error("unknown timer type should not add a task")
	}
}

func TestTimerEngineResetOneOnceTimerPast(t *testing.T) {
	te := &TimerEngine{tasks: make(map[string]*timerExec)}
	// A one-time timer in the past should NOT be added
	pastTime := time.Now().Add(-time.Hour)
	params := map[string]any{
		"timerType": 0, // once
		"dates":     pastTime.Format(time.RFC3339Nano),
	}
	b, _ := json.Marshal(params)
	tmr := &model.TTrigger{Id: "past-1", Types: "timer", Name: "past", Params: string(b)}
	err := te.resetOne(tmr)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Should NOT add a past one-time timer
	if _, ok := te.tasks["past-1"]; ok {
		t.Error("past one-time timer should not be added")
	}
}

func TestTimerEngineResetOneMissingTimerType(t *testing.T) {
	te := &TimerEngine{tasks: make(map[string]*timerExec)}
	params := map[string]any{
		"dates": time.Now().Format(time.RFC3339Nano),
	}
	b, _ := json.Marshal(params)
	tmr := &model.TTrigger{Id: "miss-1", Types: "timer", Name: "missing", Params: string(b)}
	err := te.resetOne(tmr)
	if err == nil {
		t.Fatal("expected error for missing timerType")
	}
}

func TestTimerEngineResetOneInvalidDates(t *testing.T) {
	te := &TimerEngine{tasks: make(map[string]*timerExec)}
	params := map[string]any{
		"timerType": 1,
		"dates":     "not-a-date",
	}
	b, _ := json.Marshal(params)
	tmr := &model.TTrigger{Id: "bad-date", Types: "timer", Name: "bad", Params: string(b)}
	err := te.resetOne(tmr)
	if err == nil {
		t.Fatal("expected error for invalid dates format")
	}
}

func TestTimerEngineResetOneUpdateExisting(t *testing.T) {
	te := &TimerEngine{tasks: make(map[string]*timerExec)}
	// Pre-populate a task
	te.tasks["reuse-1"] = &timerExec{
		tt:  &model.TTrigger{Id: "reuse-1"},
		typ: 1,
	}
	// Reset with a different type
	params := map[string]any{
		"timerType": 3, // daily
		"dates":     time.Now().Format(time.RFC3339Nano),
	}
	b, _ := json.Marshal(params)
	tmr := &model.TTrigger{Id: "reuse-1", Types: "timer", Name: "reuse", Params: string(b)}
	err := te.resetOne(tmr)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	task := te.tasks["reuse-1"]
	if task.typ != 3 {
		t.Fatalf("expected type updated to 3, got %d", task.typ)
	}
}

func TestTimerEngineResetOneNilTrigger(t *testing.T) {
	te := &TimerEngine{tasks: make(map[string]*timerExec)}
	err := te.resetOne(nil)
	if err == nil {
		t.Fatal("expected error from nil trigger (panic recovery)")
	}
}

func TestTimerEngineDeleteNonExistent(t *testing.T) {
	te := &TimerEngine{tasks: make(map[string]*timerExec)}
	// Should not panic
	te.Delete("nonexistent")
}

// Note: TimerEngine.Refresh requires database access and is tested separately
// in integration tests. The empty ID case is tested in timermgr_test.go.

// Test that sentinel errors are properly defined and usable
func TestSentinelErrors(t *testing.T) {
	sentinels := []struct {
		name string
		err  error
	}{
		{"ErrBuildNotFound", ErrBuildNotFound},
		{"ErrJobNotFound", ErrJobNotFound},
		{"ErrCmdNotFound", ErrCmdNotFound},
		{"ErrInvalidFSType", ErrInvalidFSType},
		{"ErrEmptyParams", ErrEmptyParams},
		{"ErrArtifactoryNotFound", ErrArtifactoryNotFound},
		{"ErrArtifactNotFound", ErrArtifactNotFound},
		{"ErrPermissionDenied", ErrPermissionDenied},
		{"ErrPluginNotFound", ErrPluginNotFound},
		{"ErrInvalidTriggerType", ErrInvalidTriggerType},
		{"ErrArtifactoryDisabled", ErrArtifactoryDisabled},
		{"ErrUnknownWebhookType", ErrUnknownWebhookType},
		{"ErrAssetNotFound", ErrAssetNotFound},
		{"ErrInvalidConfig", ErrInvalidConfig},
		{"ErrDuplicateEntry", ErrDuplicateEntry},
		{"ErrRepositoryNil", ErrRepositoryNil},
	}
	for _, s := range sentinels {
		t.Run(s.name, func(t *testing.T) {
			if s.err == nil {
				t.Fatalf("%s should not be nil", s.name)
			}
			if !errors.Is(s.err, s.err) {
				t.Fatalf("%s should match itself via errors.Is", s.name)
			}
		})
	}
}
