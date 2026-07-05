package engine

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/gokins/gokins/model"
)

// --- Timer engine: all timer types ---

func TestTimerEngineResetOneHourlyTimer(t *testing.T) {
	te := &TimerEngine{tasks: make(map[string]*timerExec)}
	params := map[string]any{
		"timerType": 2, // hourly
		"dates":     time.Now().Format(time.RFC3339Nano),
	}
	paramBytes, _ := json.Marshal(params)
	tmr := &model.TTrigger{
		Id:     "hourly-1",
		Types:  "timer",
		Name:   "hourly",
		Params: string(paramBytes),
	}
	if err := te.resetOne(tmr); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	task, ok := te.tasks["hourly-1"]
	if !ok {
		t.Fatal("expected task to be in map")
	}
	if task.typ != 2 {
		t.Errorf("expected type 2, got %d", task.typ)
	}
	if time.Until(task.tick) > time.Hour+time.Second {
		t.Error("tick should be within ~1 hour")
	}
}

func TestTimerEngineResetOneDailyTimer(t *testing.T) {
	te := &TimerEngine{tasks: make(map[string]*timerExec)}
	params := map[string]any{
		"timerType": 3, // daily
		"dates":     time.Now().Format(time.RFC3339Nano),
	}
	paramBytes, _ := json.Marshal(params)
	tmr := &model.TTrigger{
		Id:     "daily-1",
		Types:  "timer",
		Name:   "daily",
		Params: string(paramBytes),
	}
	if err := te.resetOne(tmr); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	task, ok := te.tasks["daily-1"]
	if !ok {
		t.Fatal("expected task to be in map")
	}
	if task.typ != 3 {
		t.Errorf("expected type 3, got %d", task.typ)
	}
}

func TestTimerEngineResetOneWeeklyTimer(t *testing.T) {
	te := &TimerEngine{tasks: make(map[string]*timerExec)}
	params := map[string]any{
		"timerType": 4, // weekly
		"dates":     time.Now().Format(time.RFC3339Nano),
	}
	paramBytes, _ := json.Marshal(params)
	tmr := &model.TTrigger{
		Id:     "weekly-1",
		Types:  "timer",
		Name:   "weekly",
		Params: string(paramBytes),
	}
	if err := te.resetOne(tmr); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	task, ok := te.tasks["weekly-1"]
	if !ok {
		t.Fatal("expected task to be in map")
	}
	if task.typ != 4 {
		t.Errorf("expected type 4, got %d", task.typ)
	}
}

func TestTimerEngineResetOneMonthlyTimer(t *testing.T) {
	te := &TimerEngine{tasks: make(map[string]*timerExec)}
	params := map[string]any{
		"timerType": 5, // monthly
		"dates":     time.Now().Format(time.RFC3339Nano),
	}
	paramBytes, _ := json.Marshal(params)
	tmr := &model.TTrigger{
		Id:     "monthly-1",
		Types:  "timer",
		Name:   "monthly",
		Params: string(paramBytes),
	}
	if err := te.resetOne(tmr); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	task, ok := te.tasks["monthly-1"]
	if !ok {
		t.Fatal("expected task to be in map")
	}
	if task.typ != 5 {
		t.Errorf("expected type 5, got %d", task.typ)
	}
}

func TestTimerEngineResetOne_ExpiredOnce(t *testing.T) {
	te := &TimerEngine{tasks: make(map[string]*timerExec)}
	// A once-timer with a date in the past should NOT be added
	pastTime := time.Now().Add(-time.Hour)
	params := map[string]any{
		"timerType": 0,
		"dates":     pastTime.Format(time.RFC3339Nano),
	}
	paramBytes, _ := json.Marshal(params)
	tmr := &model.TTrigger{
		Id:     "expired-once",
		Types:  "timer",
		Name:   "expired",
		Params: string(paramBytes),
	}
	if err := te.resetOne(tmr); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Should not be in the map because it's already past
	if _, ok := te.tasks["expired-once"]; ok {
		t.Fatal("expected expired once-timer NOT to be in map")
	}
}

func TestTimerEngineResetOne_UpdateExisting(t *testing.T) {
	te := &TimerEngine{tasks: make(map[string]*timerExec)}

	// Add a minute timer
	params := map[string]any{
		"timerType": 1,
		"dates":     time.Now().Format(time.RFC3339Nano),
	}
	paramBytes, _ := json.Marshal(params)
	tmr := &model.TTrigger{
		Id:     "update-1",
		Types:  "timer",
		Name:   "update",
		Params: string(paramBytes),
	}
	if err := te.resetOne(tmr); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	task, ok := te.tasks["update-1"]
	if !ok {
		t.Fatal("expected task to exist")
	}
	oldTick := task.tick

	// Update to hourly
	params2 := map[string]any{
		"timerType": 2,
		"dates":     time.Now().Format(time.RFC3339Nano),
	}
	paramBytes2, _ := json.Marshal(params2)
	tmr.Params = string(paramBytes2)
	if err := te.resetOne(tmr); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	task2, ok := te.tasks["update-1"]
	if !ok {
		t.Fatal("expected task to still exist")
	}
	if task2.typ != 2 {
		t.Errorf("expected type to be updated to 2, got %d", task2.typ)
	}
	if task2.tick == oldTick {
		t.Error("expected tick to be updated")
	}
}

func TestTimerEngineResetOne_InvalidDates(t *testing.T) {
	te := &TimerEngine{tasks: make(map[string]*timerExec)}
	params := map[string]any{
		"timerType": 1,
		"dates":     "not-a-date",
	}
	paramBytes, _ := json.Marshal(params)
	tmr := &model.TTrigger{
		Id:     "bad-dates",
		Types:  "timer",
		Name:   "bad",
		Params: string(paramBytes),
	}
	err := te.resetOne(tmr)
	if err == nil {
		t.Fatal("expected error for invalid dates")
	}
}

func TestTimerEngineResetOne_MissingTimerType(t *testing.T) {
	te := &TimerEngine{tasks: make(map[string]*timerExec)}
	params := map[string]any{
		"dates": time.Now().Format(time.RFC3339Nano),
	}
	paramBytes, _ := json.Marshal(params)
	tmr := &model.TTrigger{
		Id:     "no-type",
		Types:  "timer",
		Name:   "no-type",
		Params: string(paramBytes),
	}
	err := te.resetOne(tmr)
	if err == nil {
		t.Fatal("expected error for missing timerType")
	}
}

func TestTimerEngineDeleteMultiple(t *testing.T) {
	te := &TimerEngine{tasks: make(map[string]*timerExec)}
	for i := 0; i < 5; i++ {
		id := "t" + string(rune('a'+i))
		te.tasks[id] = &timerExec{tt: &model.TTrigger{Id: id}}
	}
	if len(te.tasks) != 5 {
		t.Fatalf("expected 5 tasks, got %d", len(te.tasks))
	}
	te.Delete("ta")
	te.Delete("tc")
	if len(te.tasks) != 3 {
		t.Errorf("expected 3 tasks after deletes, got %d", len(te.tasks))
	}
	// Delete non-existent should not panic
	te.Delete("nonexistent")
}
