package engine

import (
	"encoding/json"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gokins/gokins/model"
)

// TestTimerEngineResetOneHourlyTimer verifies that a recurring hourly timer (type 2)
// is correctly initialized with a tick time within the next hour.
func TestTimerEngineResetOneHourlyTimer(t *testing.T) {
	te := &TimerEngine{tasks: make(map[string]*timerExec)}

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
	err := te.resetOne(tmr)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	task, ok := te.tasks["test-hourly"]
	if !ok {
		t.Fatal("expected task to be added")
	}
	if task.typ != 2 {
		t.Fatalf("expected type 2 (hourly), got %d", task.typ)
	}
	if time.Until(task.tick) > time.Hour+time.Second {
		t.Fatal("tick should be within ~1 hour")
	}
	if time.Until(task.tick) < -time.Second {
		t.Fatal("tick should not be in the past")
	}
}

// TestTimerEngineResetOneDailyTimer verifies that a recurring daily timer (type 3)
// is correctly initialized with a tick time within the next 24 hours.
func TestTimerEngineResetOneDailyTimer(t *testing.T) {
	te := &TimerEngine{tasks: make(map[string]*timerExec)}

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
		t.Fatalf("expected type 3 (daily), got %d", task.typ)
	}
	if time.Until(task.tick) > 24*time.Hour+time.Second {
		t.Fatal("tick should be within ~24 hours")
	}
	if time.Until(task.tick) < -time.Second {
		t.Fatal("tick should not be in the past")
	}
}

// TestTimerEngineResetOneWeeklyTimer verifies that a recurring weekly timer (type 4)
// is correctly initialized with a tick time within the next 7 days.
func TestTimerEngineResetOneWeeklyTimer(t *testing.T) {
	te := &TimerEngine{tasks: make(map[string]*timerExec)}

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
		t.Fatalf("expected type 4 (weekly), got %d", task.typ)
	}
	if time.Until(task.tick) > 7*24*time.Hour+time.Second {
		t.Fatal("tick should be within ~7 days")
	}
	if time.Until(task.tick) < -time.Second {
		t.Fatal("tick should not be in the past")
	}
}

// TestTimerEngineResetOneMonthlyTimer verifies that a recurring monthly timer (type 5)
// is correctly initialized with a tick time within the next 30 days.
func TestTimerEngineResetOneMonthlyTimer(t *testing.T) {
	te := &TimerEngine{tasks: make(map[string]*timerExec)}

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
		t.Fatalf("expected type 5 (monthly), got %d", task.typ)
	}
	if time.Until(task.tick) > 31*24*time.Hour+time.Second {
		t.Fatal("tick should be within ~31 days")
	}
	if time.Until(task.tick) < -time.Second {
		t.Fatal("tick should not be in the past")
	}
}

// TestTimerEngineResetOneMissingTimerType verifies that a missing timerType field
// returns a clear error.
func TestTimerEngineResetOneMissingTimerType(t *testing.T) {
	te := &TimerEngine{tasks: make(map[string]*timerExec)}

	params := map[string]any{
		// timerType intentionally omitted
		"dates": time.Now().Format(time.RFC3339Nano),
	}
	paramBytes, _ := json.Marshal(params)

	tmr := &model.TTrigger{
		Id:     "test-notype",
		Types:  "timer",
		Name:   "no-type-timer",
		Params: string(paramBytes),
	}
	err := te.resetOne(tmr)
	if err == nil {
		t.Fatal("expected error for missing timerType")
	}
	if !strings.Contains(err.Error(), "timerType") {
		t.Fatalf("expected error about timerType, got: %v", err)
	}
}

// TestTimerEngineResetOneMissingDates verifies that a missing dates field
// returns a clear error.
func TestTimerEngineResetOneMissingDates(t *testing.T) {
	te := &TimerEngine{tasks: make(map[string]*timerExec)}

	params := map[string]any{
		"timerType": 1,
		// dates intentionally omitted
	}
	paramBytes, _ := json.Marshal(params)

	tmr := &model.TTrigger{
		Id:     "test-nodates",
		Types:  "timer",
		Name:   "no-dates-timer",
		Params: string(paramBytes),
	}
	err := te.resetOne(tmr)
	if err == nil {
		t.Fatal("expected error for missing dates")
	}
}

// TestTimerEngineResetOneInvalidDatesFormat verifies that an unparseable dates
// value returns an error.
func TestTimerEngineResetOneInvalidDatesFormat(t *testing.T) {
	te := &TimerEngine{tasks: make(map[string]*timerExec)}

	params := map[string]any{
		"timerType": 1,
		"dates":     "not-a-date",
	}
	paramBytes, _ := json.Marshal(params)

	tmr := &model.TTrigger{
		Id:     "test-baddates",
		Types:  "timer",
		Name:   "bad-dates-timer",
		Params: string(paramBytes),
	}
	err := te.resetOne(tmr)
	if err == nil {
		t.Fatal("expected error for invalid dates format")
	}
}

// TestTimerEngineResetOneOnceTimerPast verifies that a one-time timer (type 0)
// with a date in the past is NOT added to the tasks map.
func TestTimerEngineResetOneOnceTimerPast(t *testing.T) {
	te := &TimerEngine{tasks: make(map[string]*timerExec)}

	pastTime := time.Now().Add(-time.Hour)
	params := map[string]any{
		"timerType": 0, // once
		"dates":     pastTime.Format(time.RFC3339Nano),
	}
	paramBytes, _ := json.Marshal(params)

	tmr := &model.TTrigger{
		Id:     "test-past-once",
		Types:  "timer",
		Name:   "past-once-timer",
		Params: string(paramBytes),
	}
	err := te.resetOne(tmr)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Once timer in the past should NOT be added
	if _, ok := te.tasks["test-past-once"]; ok {
		t.Fatal("once timer in the past should not be added to tasks")
	}
}

// TestTimerEngineResetOneUpdateExisting verifies that calling resetOne on an
// existing task ID updates the task rather than creating a duplicate.
func TestTimerEngineResetOneUpdateExisting(t *testing.T) {
	te := &TimerEngine{tasks: make(map[string]*timerExec)}

	// Add a minute timer
	params := map[string]any{
		"timerType": 1,
		"dates":     time.Now().Format(time.RFC3339Nano),
	}
	paramBytes, _ := json.Marshal(params)

	tmr := &model.TTrigger{
		Id:     "test-update",
		Types:  "timer",
		Name:   "original",
		Params: string(paramBytes),
	}
	err := te.resetOne(tmr)
	if err != nil {
		t.Fatalf("first resetOne: %v", err)
	}

	// Update to hourly timer
	params2 := map[string]any{
		"timerType": 2,
		"dates":     time.Now().Format(time.RFC3339Nano),
	}
	paramBytes2, _ := json.Marshal(params2)

	tmr2 := &model.TTrigger{
		Id:     "test-update",
		Types:  "timer",
		Name:   "updated",
		Params: string(paramBytes2),
	}
	err = te.resetOne(tmr2)
	if err != nil {
		t.Fatalf("second resetOne: %v", err)
	}

	if len(te.tasks) != 1 {
		t.Fatalf("expected 1 task, got %d", len(te.tasks))
	}
	task := te.tasks["test-update"]
	if task.typ != 2 {
		t.Fatalf("expected updated type 2, got %d", task.typ)
	}
	// Note: resetOne does not replace the tt pointer when updating an
	// existing task, so the name stays as the original.  The typ and tms
	// fields are updated.
	if task.tms.IsZero() {
		t.Fatal("expected tms to be set after update")
	}
}

// TestTimerEngineResetOneConcurrentAllTypes verifies that concurrent resetOne
// calls with all 6 timer types work correctly without races.
func TestTimerEngineResetOneConcurrentAllTypes(t *testing.T) {
	te := &TimerEngine{tasks: make(map[string]*timerExec)}

	var wg sync.WaitGroup
	for i := 0; i < 30; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			typ := int64(n % 6) // types 0-5
			futureTime := time.Now().Add(time.Hour)
			params := map[string]any{
				"timerType": typ,
				"dates":     futureTime.Format(time.RFC3339Nano),
			}
			paramBytes, _ := json.Marshal(params)
			tmr := &model.TTrigger{
				Id:     "concurrent-all-" + string(rune('A'+n)),
				Types:  "timer",
				Name:   "test",
				Params: string(paramBytes),
			}
			_ = te.resetOne(tmr)
		}(i)
	}
	wg.Wait()

	// Verify some tasks were added
	te.tasklk.RLock()
	count := len(te.tasks)
	te.tasklk.RUnlock()
	if count == 0 {
		t.Fatal("expected at least some tasks after concurrent resetOne")
	}
}

// TestTimerEngineRunEmptyTasks verifies that run() on an empty task map
// does not panic.
func TestTimerEngineRunEmptyTasks(t *testing.T) {
	te := &TimerEngine{tasks: make(map[string]*timerExec)}

	// Should not panic even with no tasks
	te.run()
}

// TestTimerEngineDeleteConcurrent verifies that concurrent Delete calls
// do not cause a data race.
func TestTimerEngineDeleteConcurrent(t *testing.T) {
	te := &TimerEngine{tasks: make(map[string]*timerExec)}

	// Pre-populate
	for i := 0; i < 50; i++ {
		id := string(rune('a' + i%26))
		te.tasks[id] = &timerExec{
			tt:  &model.TTrigger{Id: id},
			typ: 1,
		}
	}

	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			te.Delete(string(rune('a' + n%26)))
		}(i)
	}
	wg.Wait()

	// All tasks should be deleted
	te.tasklk.RLock()
	remaining := len(te.tasks)
	te.tasklk.RUnlock()
	if remaining != 0 {
		t.Fatalf("expected 0 remaining tasks, got %d", remaining)
	}
}
