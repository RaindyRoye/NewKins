package engine

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/gokins/gokins/model"
)

func TestTimerEngineNew(t *testing.T) {
	te := &TimerEngine{
		tasks: make(map[string]*timerExec),
	}
	if te.tasks == nil {
		t.Fatal("expected tasks map to be initialized")
	}
	if len(te.tasks) != 0 {
		t.Fatalf("expected empty tasks map, got %d", len(te.tasks))
	}
}

func TestTimerEngineDelete(t *testing.T) {
	te := &TimerEngine{
		tasks: make(map[string]*timerExec),
	}
	// Add a task
	te.tasks["test-id"] = &timerExec{
		tt:  &model.TTrigger{Id: "test-id"},
		typ: 1,
	}
	if len(te.tasks) != 1 {
		t.Fatalf("expected 1 task, got %d", len(te.tasks))
	}

	// Delete it
	te.Delete("test-id")
	if len(te.tasks) != 0 {
		t.Fatalf("expected 0 tasks after delete, got %d", len(te.tasks))
	}

	// Delete non-existent should not panic
	te.Delete("non-existent")
}

func TestTimerEngineResetOneInvalidType(t *testing.T) {
	te := &TimerEngine{
		tasks: make(map[string]*timerExec),
	}
	tmr := &model.TTrigger{
		Id:     "test-1",
		Types:  "webhook", // wrong type
		Name:   "test",
		Params: `{}`,
	}
	err := te.resetOne(tmr)
	if err == nil {
		t.Fatal("expected error for invalid trigger type")
	}
	expected := "expected trigger type 'timer', got 'webhook'"
	if err.Error() != expected {
		t.Fatalf("expected error %q, got %q", expected, err.Error())
	}
}

func TestTimerEngineResetOneInvalidJSON(t *testing.T) {
	te := &TimerEngine{
		tasks: make(map[string]*timerExec),
	}
	tmr := &model.TTrigger{
		Id:     "test-2",
		Types:  "timer",
		Name:   "test",
		Params: `{invalid json`,
	}
	err := te.resetOne(tmr)
	if err == nil {
		t.Fatal("expected error for invalid JSON params")
	}
}

func TestTimerEngineResetOneOnceTimer(t *testing.T) {
	te := &TimerEngine{
		tasks: make(map[string]*timerExec),
	}
	// Create a one-time timer set in the future
	futureTime := time.Now().Add(time.Hour)
	params := map[string]interface{}{
		"timerType": 0, // once
		"dates":     futureTime.Format(time.RFC3339Nano),
	}
	paramBytes, _ := json.Marshal(params)

	tmr := &model.TTrigger{
		Id:     "test-once",
		Types:  "timer",
		Name:   "once-timer",
		Params: string(paramBytes),
	}
	err := te.resetOne(tmr)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Verify task was added
	task, ok := te.tasks["test-once"]
	if !ok {
		t.Fatal("expected task to be added")
	}
	if task.typ != 0 {
		t.Fatalf("expected type 0, got %d", task.typ)
	}
}

func TestTimerEngineResetOneMinuteTimer(t *testing.T) {
	te := &TimerEngine{
		tasks: make(map[string]*timerExec),
	}
	// Create a recurring minute timer
	params := map[string]interface{}{
		"timerType": 1, // minute
		"dates":     time.Now().Format(time.RFC3339Nano),
	}
	paramBytes, _ := json.Marshal(params)

	tmr := &model.TTrigger{
		Id:     "test-minute",
		Types:  "timer",
		Name:   "minute-timer",
		Params: string(paramBytes),
	}
	err := te.resetOne(tmr)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	task, ok := te.tasks["test-minute"]
	if !ok {
		t.Fatal("expected task to be added")
	}
	if task.typ != 1 {
		t.Fatalf("expected type 1, got %d", task.typ)
	}
	// tick should be in the near future
	if time.Until(task.tick) > time.Minute+time.Second {
		t.Fatal("tick should be within ~1 minute")
	}
}

func TestTimerEngineRefreshEmptyID(t *testing.T) {
	te := &TimerEngine{
		tasks: make(map[string]*timerExec),
	}
	err := te.Refresh("")
	if err == nil {
		t.Fatal("expected error for empty timer ID")
	}
	if err.Error() != "timer id is empty" {
		t.Fatalf("expected 'timer id is empty', got %q", err.Error())
	}
}

func TestTimerEngineResetOnePanicRecovery(t *testing.T) {
	te := &TimerEngine{
		tasks: make(map[string]*timerExec),
	}
	// Passing nil should trigger a panic in the function body
	// The deferred recover should catch it and return an error
	defer func() {
		if r := recover(); r != nil {
			// If panic escapes, the test should fail
			t.Fatalf("panic escaped resetOne: %v", r)
		}
	}()
	// nil trigger should cause a nil pointer dereference panic
	err := te.resetOne(nil)
	if err == nil {
		t.Fatal("expected error from panic recovery")
	}
}
