package engine

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"
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
	params := map[string]any{
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
	params := map[string]any{
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

func TestTimerEngineResetOneHourlyTimer(t *testing.T) {
	te := &TimerEngine{
		tasks: make(map[string]*timerExec),
	}
	params := map[string]any{
		"timerType": 2, // hourly
		"dates":     time.Now().Format(time.RFC3339Nano),
	}
	paramBytes, _ := json.Marshal(params)

	tmr := &model.TTrigger{
		Id:     "test-hourly",
		Types:  "timer",
		Name:   "hourly-timer",
		Params: string(paramBytes),
	}
	if err := te.resetOne(tmr); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	task, ok := te.tasks["test-hourly"]
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
	te := &TimerEngine{
		tasks: make(map[string]*timerExec),
	}
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
	if err := te.resetOne(tmr); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	task, ok := te.tasks["test-daily"]
	if !ok {
		t.Fatal("expected task to be added")
	}
	if task.typ != 3 {
		t.Fatalf("expected type 3, got %d", task.typ)
	}
	if time.Until(task.tick) > 24*time.Hour+time.Second {
		t.Fatal("tick should be within ~24 hours")
	}
}

func TestTimerEngineResetOneWeeklyTimer(t *testing.T) {
	te := &TimerEngine{
		tasks: make(map[string]*timerExec),
	}
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
	if err := te.resetOne(tmr); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	task, ok := te.tasks["test-weekly"]
	if !ok {
		t.Fatal("expected task to be added")
	}
	if task.typ != 4 {
		t.Fatalf("expected type 4, got %d", task.typ)
	}
	if time.Until(task.tick) > 7*24*time.Hour+time.Second {
		t.Fatal("tick should be within ~7 days")
	}
}

func TestTimerEngineResetOneMonthlyTimer(t *testing.T) {
	te := &TimerEngine{
		tasks: make(map[string]*timerExec),
	}
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
	if err := te.resetOne(tmr); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	task, ok := te.tasks["test-monthly"]
	if !ok {
		t.Fatal("expected task to be added")
	}
	if task.typ != 5 {
		t.Fatalf("expected type 5, got %d", task.typ)
	}
	if time.Until(task.tick) > 31*24*time.Hour+time.Second {
		t.Fatal("tick should be within ~30 days")
	}
}

func TestTimerEngineResetOneMissingTimerType(t *testing.T) {
	te := &TimerEngine{
		tasks: make(map[string]*timerExec),
	}
	// Params without timerType field
	params := map[string]any{
		"dates": time.Now().Format(time.RFC3339Nano),
	}
	paramBytes, _ := json.Marshal(params)

	tmr := &model.TTrigger{
		Id:     "test-nomtype",
		Types:  "timer",
		Name:   "bad-timer",
		Params: string(paramBytes),
	}
	err := te.resetOne(tmr)
	if err == nil {
		t.Fatal("expected error for missing timerType")
	}
}

func TestTimerEngineResetOneInvalidDates(t *testing.T) {
	te := &TimerEngine{
		tasks: make(map[string]*timerExec),
	}
	params := map[string]any{
		"timerType": 1,
		"dates":     "not-a-date",
	}
	paramBytes, _ := json.Marshal(params)

	tmr := &model.TTrigger{
		Id:     "test-baddates",
		Types:  "timer",
		Name:   "bad-timer",
		Params: string(paramBytes),
	}
	err := te.resetOne(tmr)
	if err == nil {
		t.Fatal("expected error for invalid dates")
	}
}

func TestTimerEngineResetOneOnceTimerPastTime(t *testing.T) {
	te := &TimerEngine{
		tasks: make(map[string]*timerExec),
	}
	// One-time timer set in the past — should not be scheduled
	pastTime := time.Now().Add(-time.Hour)
	params := map[string]any{
		"timerType": 0, // once
		"dates":     pastTime.Format(time.RFC3339Nano),
	}
	paramBytes, _ := json.Marshal(params)

	tmr := &model.TTrigger{
		Id:     "test-once-past",
		Types:  "timer",
		Name:   "once-past-timer",
		Params: string(paramBytes),
	}
	if err := te.resetOne(tmr); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Past one-time timer should not be added
	if _, ok := te.tasks["test-once-past"]; ok {
		t.Fatal("past one-time timer should not be scheduled")
	}
}

func TestTimerEngineResetOneOverwriteExisting(t *testing.T) {
	te := &TimerEngine{
		tasks: make(map[string]*timerExec),
	}
	// Pre-populate a task
	te.tasks["test-overwrite"] = &timerExec{
		tt:  &model.TTrigger{Id: "test-overwrite"},
		typ: 1,
	}

	// Reset with different type
	params := map[string]any{
		"timerType": 3, // daily
		"dates":     time.Now().Format(time.RFC3339Nano),
	}
	paramBytes, _ := json.Marshal(params)

	tmr := &model.TTrigger{
		Id:     "test-overwrite",
		Types:  "timer",
		Name:   "overwrite-timer",
		Params: string(paramBytes),
	}
	if err := te.resetOne(tmr); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	task := te.tasks["test-overwrite"]
	// resetOne updates typ but not tt when overwriting an existing task
	if task.typ != 3 {
		t.Fatalf("expected overwritten type 3, got %d", task.typ)
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
	if !strings.Contains(err.Error(), "timer id is empty") {
		t.Fatalf("expected error containing 'timer id is empty', got %q", err.Error())
	}
	// Verify it wraps ErrEmptyParams
	if !errors.Is(err, ErrEmptyParams) {
		t.Fatalf("expected error to wrap ErrEmptyParams, got %v", err)
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

func TestTimerEngineConcurrentResetOneAndDelete(t *testing.T) {
	te := &TimerEngine{
		tasks: make(map[string]*timerExec),
	}

	// Pre-populate some tasks
	for i := 0; i < 20; i++ {
		id := fmt.Sprintf("timer-%d", i)
		te.tasks[id] = &timerExec{
			tt:  &model.TTrigger{Id: id},
			typ: 1,
		}
	}

	var wg sync.WaitGroup

	// Concurrent resetOne calls (which now hold the lock internally)
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			futureTime := time.Now().Add(time.Hour)
			params := map[string]any{
				"timerType": 1,
				"dates":     futureTime.Format(time.RFC3339Nano),
			}
			paramBytes, _ := json.Marshal(params)
			tmr := &model.TTrigger{
				Id:     fmt.Sprintf("timer-%d", n),
				Types:  "timer",
				Name:   fmt.Sprintf("test-%d", n),
				Params: string(paramBytes),
			}
			_ = te.resetOne(tmr)
		}(i)
	}

	// Concurrent Delete calls
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			te.Delete(fmt.Sprintf("timer-%d", n))
		}(i)
	}

	wg.Wait()
	// If we get here without a race detector complaint, the test passes
}

func TestTimerEngineConcurrentReadsAndWrites(t *testing.T) {
	te := &TimerEngine{
		tasks: make(map[string]*timerExec),
	}

	var wg sync.WaitGroup

	// Writers: add tasks
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			futureTime := time.Now().Add(time.Hour)
			params := map[string]any{
				"timerType": 2,
				"dates":     futureTime.Format(time.RFC3339Nano),
			}
			paramBytes, _ := json.Marshal(params)
			tmr := &model.TTrigger{
				Id:     fmt.Sprintf("concurrent-%d", n),
				Types:  "timer",
				Name:   fmt.Sprintf("test-%d", n),
				Params: string(paramBytes),
			}
			_ = te.resetOne(tmr)
		}(i)
	}

	// Readers: read tasks via RLock (simulated by accessing tasklk.RLock)
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			te.tasklk.RLock()
			_ = len(te.tasks)
			te.tasklk.RUnlock()
		}()
	}

	// Deleters: remove tasks
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			te.Delete(fmt.Sprintf("concurrent-%d", n))
		}(i)
	}

	wg.Wait()
}
