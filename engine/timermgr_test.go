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
	// Verify the error wraps ErrInvalidTriggerType
	if !errors.Is(err, ErrInvalidTriggerType) {
		t.Fatalf("expected error to wrap ErrInvalidTriggerType, got %v", err)
	}
	// Verify the error message contains useful context
	if !strings.Contains(err.Error(), "expected 'timer', got 'webhook'") {
		t.Fatalf("expected error to contain context, got %q", err.Error())
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

	// Readers: read tasks via TaskCount/TaskIDs
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = te.TaskCount()
			_ = te.TaskIDs()
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

func TestTimerEngineTaskCount(t *testing.T) {
	te := &TimerEngine{
		tasks: make(map[string]*timerExec),
	}
	if te.TaskCount() != 0 {
		t.Fatalf("expected 0 tasks, got %d", te.TaskCount())
	}

	te.tasks["a"] = &timerExec{tt: &model.TTrigger{Id: "a"}}
	te.tasks["b"] = &timerExec{tt: &model.TTrigger{Id: "b"}}
	if te.TaskCount() != 2 {
		t.Fatalf("expected 2 tasks, got %d", te.TaskCount())
	}

	te.Delete("a")
	if te.TaskCount() != 1 {
		t.Fatalf("expected 1 task, got %d", te.TaskCount())
	}
}

func TestTimerEngineTaskIDs(t *testing.T) {
	te := &TimerEngine{
		tasks: make(map[string]*timerExec),
	}
	ids := te.TaskIDs()
	if len(ids) != 0 {
		t.Fatalf("expected 0 IDs, got %d", len(ids))
	}

	te.tasks["x"] = &timerExec{tt: &model.TTrigger{Id: "x"}}
	te.tasks["y"] = &timerExec{tt: &model.TTrigger{Id: "y"}}
	ids = te.TaskIDs()
	if len(ids) != 2 {
		t.Fatalf("expected 2 IDs, got %d", len(ids))
	}
	found := map[string]bool{}
	for _, id := range ids {
		found[id] = true
	}
	if !found["x"] || !found["y"] {
		t.Fatalf("expected IDs x and y, got %v", ids)
	}
}

// TestTimerEngineRunSkipsFuture verifies that run() does not fire timers
// whose tick time is in the future.
func TestTimerEngineRunSkipsFuture(t *testing.T) {
	te := &TimerEngine{
		tasks: make(map[string]*timerExec),
	}
	futureTick := time.Now().Add(time.Hour)
	te.tasks["future"] = &timerExec{
		tt:   &model.TTrigger{Id: "future", Name: "future-timer"},
		typ:  1,
		tick: futureTick,
	}
	// run should not panic and should not remove the task
	te.run()
	if te.TaskCount() != 1 {
		t.Fatal("future timer should not be removed by run()")
	}
}

// TestTimerEngineExecItemNotDue verifies that execItem does nothing
// when the tick time has not yet arrived.
func TestTimerEngineExecItemNotDue(t *testing.T) {
	te := &TimerEngine{
		tasks: make(map[string]*timerExec),
	}
	futureTick := time.Now().Add(time.Hour)
	item := &timerExec{
		tt:   &model.TTrigger{Id: "notdue", Name: "not-due"},
		typ:  1,
		tick: futureTick,
	}
	te.tasks["notdue"] = item
	te.execItem(item)
	// Task should still be present and tick unchanged
	if te.TaskCount() != 1 {
		t.Fatal("not-due timer should remain")
	}
	if !item.tick.Equal(futureTick) {
		t.Fatal("tick should not have changed for a not-due item")
	}
}

// TestTimerEngineResetOneAllRecurringTypes verifies that all recurring timer
// types (1-5) are registered correctly with ticks in the near future.
func TestTimerEngineResetOneAllRecurringTypes(t *testing.T) {
	types := []int64{1, 2, 3, 4, 5}
	names := []string{"minute", "hour", "day", "week", "month"}
	maxDurations := []time.Duration{
		time.Minute + time.Second,
		time.Hour + time.Second,
		25 * time.Hour,
		8 * 24 * time.Hour,
		32 * 24 * time.Hour,
	}

	for i, typ := range types {
		te := &TimerEngine{
			tasks: make(map[string]*timerExec),
		}
		params := map[string]any{
			"timerType": typ,
			"dates":     time.Now().Format(time.RFC3339Nano),
		}
		paramBytes, _ := json.Marshal(params)
		tmr := &model.TTrigger{
			Id:     fmt.Sprintf("test-%s", names[i]),
			Types:  "timer",
			Name:   names[i],
			Params: string(paramBytes),
		}
		err := te.resetOne(tmr)
		if err != nil {
			t.Fatalf("type %d (%s): unexpected error: %v", typ, names[i], err)
		}
		task, ok := te.tasks[tmr.Id]
		if !ok {
			t.Fatalf("type %d (%s): task not found", typ, names[i])
		}
		if task.typ != typ {
			t.Fatalf("type %d (%s): expected typ=%d, got %d", typ, names[i], typ, task.typ)
		}
		until := time.Until(task.tick)
		if until > maxDurations[i] {
			t.Fatalf("type %d (%s): tick too far in future: %v (max %v)", typ, names[i], until, maxDurations[i])
		}
	}
}

// TestTimerEngineResetOnceTimerPast verifies that a one-shot timer whose
// time has already passed is NOT added to the task map.
func TestTimerEngineResetOnceTimerPast(t *testing.T) {
	te := &TimerEngine{
		tasks: make(map[string]*timerExec),
	}
	pastTime := time.Now().Add(-time.Hour)
	params := map[string]any{
		"timerType": 0,
		"dates":     pastTime.Format(time.RFC3339Nano),
	}
	paramBytes, _ := json.Marshal(params)
	tmr := &model.TTrigger{
		Id:     "past-once",
		Types:  "timer",
		Name:   "past-once",
		Params: string(paramBytes),
	}
	err := te.resetOne(tmr)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if te.TaskCount() != 0 {
		t.Fatal("past one-shot timer should not be added")
	}
}

// TestTimerEngineResetOneIdempotent verifies that calling resetOne twice
// on the same timer ID updates the existing entry rather than creating a duplicate.
func TestTimerEngineResetOneIdempotent(t *testing.T) {
	te := &TimerEngine{
		tasks: make(map[string]*timerExec),
	}
	params := map[string]any{
		"timerType": 1,
		"dates":     time.Now().Format(time.RFC3339Nano),
	}
	paramBytes, _ := json.Marshal(params)
	tmr := &model.TTrigger{
		Id:     "idem",
		Types:  "timer",
		Name:   "idem",
		Params: string(paramBytes),
	}

	err := te.resetOne(tmr)
	if err != nil {
		t.Fatalf("first resetOne failed: %v", err)
	}
	err = te.resetOne(tmr)
	if err != nil {
		t.Fatalf("second resetOne failed: %v", err)
	}
	if te.TaskCount() != 1 {
		t.Fatalf("expected 1 task after idempotent reset, got %d", te.TaskCount())
	}
}
