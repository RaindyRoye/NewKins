package engine

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/gokins/gokins/model"
)

func TestTimerEngineResetOneHourTimer(t *testing.T) {
	te := &TimerEngine{
		tasks: make(map[string]*timerExec),
	}
	// Create a recurring hour timer
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
	// tick should be within ~1 hour
	if time.Until(task.tick) > time.Hour+time.Second {
		t.Fatal("tick should be within ~1 hour")
	}
}

func TestTimerEngineResetOneDailyTimer(t *testing.T) {
	te := &TimerEngine{
		tasks: make(map[string]*timerExec),
	}
	// Create a recurring daily timer
	params := map[string]any{
		"timerType": 3, // daily
		"dates":     time.Now().Format(time.RFC3339Nano),
	}
	paramBytes, _ := json.Marshal(params)

	tmr := &model.TTrigger{
		Id:     "test-daily",
		Types:  "timer",
		Name:   "daily-timer",
		Params: string(paramBytes),
	}
	err := te.resetOne(tmr)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	task, ok := te.tasks["test-daily"]
	if !ok {
		t.Fatal("expected task to be added")
	}
	if task.typ != 3 {
		t.Fatalf("expected type 3, got %d", task.typ)
	}
	// tick should be within ~24 hours
	if time.Until(task.tick) > 24*time.Hour+time.Second {
		t.Fatal("tick should be within ~24 hours")
	}
}

func TestTimerEngineResetOneWeeklyTimer(t *testing.T) {
	te := &TimerEngine{
		tasks: make(map[string]*timerExec),
	}
	// Create a recurring weekly timer
	params := map[string]any{
		"timerType": 4, // weekly
		"dates":     time.Now().Format(time.RFC3339Nano),
	}
	paramBytes, _ := json.Marshal(params)

	tmr := &model.TTrigger{
		Id:     "test-weekly",
		Types:  "timer",
		Name:   "weekly-timer",
		Params: string(paramBytes),
	}
	err := te.resetOne(tmr)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	task, ok := te.tasks["test-weekly"]
	if !ok {
		t.Fatal("expected task to be added")
	}
	if task.typ != 4 {
		t.Fatalf("expected type 4, got %d", task.typ)
	}
	// tick should be within ~7 days
	if time.Until(task.tick) > 7*24*time.Hour+time.Second {
		t.Fatal("tick should be within ~7 days")
	}
}

func TestTimerEngineResetOneMonthlyTimer(t *testing.T) {
	te := &TimerEngine{
		tasks: make(map[string]*timerExec),
	}
	// Create a recurring monthly timer
	params := map[string]any{
		"timerType": 5, // monthly
		"dates":     time.Now().Format(time.RFC3339Nano),
	}
	paramBytes, _ := json.Marshal(params)

	tmr := &model.TTrigger{
		Id:     "test-monthly",
		Types:  "timer",
		Name:   "monthly-timer",
		Params: string(paramBytes),
	}
	err := te.resetOne(tmr)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	task, ok := te.tasks["test-monthly"]
	if !ok {
		t.Fatal("expected task to be added")
	}
	if task.typ != 5 {
		t.Fatalf("expected type 5, got %d", task.typ)
	}
	// tick should be within ~30 days
	if time.Until(task.tick) > 30*24*time.Hour+time.Second {
		t.Fatal("tick should be within ~30 days")
	}
}

func TestTimerEngineResetOnceTimerPast(t *testing.T) {
	te := &TimerEngine{
		tasks: make(map[string]*timerExec),
	}
	// Create a one-time timer set in the past (should not be added)
	pastTime := time.Now().Add(-time.Hour)
	params := map[string]any{
		"timerType": 0, // once
		"dates":     pastTime.Format(time.RFC3339Nano),
	}
	paramBytes, _ := json.Marshal(params)

	tmr := &model.TTrigger{
		Id:     "test-once-past",
		Types:  "timer",
		Name:   "once-timer-past",
		Params: string(paramBytes),
	}
	err := te.resetOne(tmr)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Task should not be added since it's in the past
	_, ok := te.tasks["test-once-past"]
	if ok {
		t.Fatal("expected past timer not to be added")
	}
}

func TestTimerEngineResetOneInvalidDates(t *testing.T) {
	te := &TimerEngine{
		tasks: make(map[string]*timerExec),
	}
	params := map[string]any{
		"timerType": 1,
		"dates":     "invalid-date-format",
	}
	paramBytes, _ := json.Marshal(params)

	tmr := &model.TTrigger{
		Id:     "test-invalid-dates",
		Types:  "timer",
		Name:   "invalid-dates",
		Params: string(paramBytes),
	}
	err := te.resetOne(tmr)
	if err == nil {
		t.Fatal("expected error for invalid date format")
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
		Id:     "test-missing-type",
		Types:  "timer",
		Name:   "missing-type",
		Params: string(paramBytes),
	}
	err := te.resetOne(tmr)
	if err == nil {
		t.Fatal("expected error for missing timerType")
	}
}

func TestTimerEngineResetOneUpdateExisting(t *testing.T) {
	te := &TimerEngine{
		tasks: make(map[string]*timerExec),
	}
	// First, add a timer
	params1 := map[string]any{
		"timerType": 1,
		"dates":     time.Now().Format(time.RFC3339Nano),
	}
	paramBytes1, _ := json.Marshal(params1)

	tmr := &model.TTrigger{
		Id:     "test-update",
		Types:  "timer",
		Name:   "update-test",
		Params: string(paramBytes1),
	}
	err := te.resetOne(tmr)
	if err != nil {
		t.Fatalf("unexpected error on first reset: %v", err)
	}

	// Update the same timer with different type
	params2 := map[string]any{
		"timerType": 2,
		"dates":     time.Now().Format(time.RFC3339Nano),
	}
	paramBytes2, _ := json.Marshal(params2)
	tmr.Params = string(paramBytes2)

	err = te.resetOne(tmr)
	if err != nil {
		t.Fatalf("unexpected error on second reset: %v", err)
	}
	task, ok := te.tasks["test-update"]
	if !ok {
		t.Fatal("expected task to exist after update")
	}
	if task.typ != 2 {
		t.Errorf("expected type 2 after update, got %d", task.typ)
	}
}
