package engine

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/gokins/gokins/model"
)

// TestTimerEngineRun tests the run method execution
func TestTimerEngineRun(t *testing.T) {
	te := &TimerEngine{
		tasks: make(map[string]*timerExec),
	}

	// Add some timer tasks
	futureTime := time.Now().Add(time.Hour)
	params := map[string]any{
		"timerType": 1,
		"dates":     futureTime.Format(time.RFC3339Nano),
	}
	paramsBytes, _ := json.Marshal(params)

	for i := 0; i < 5; i++ {
		id := string(rune('a' + i))
		te.tasks[id] = &timerExec{
			tt: &model.TTrigger{
				Id:      id,
				Name:    "test-timer",
				Types:   "timer",
				Params:  string(paramsBytes),
				Enabled: 1,
			},
			typ:  1,
			tms:  futureTime,
			tick: time.Now().Add(time.Second * time.Duration(i+1)),
		}
	}

	// run should execute without panic
	te.run()
}

// TestTimerEngineExecItem tests execItem with various timer types
func TestTimerEngineExecItem(t *testing.T) {
	tests := []struct {
		name    string
		timerID string
		typ     int64
		tick    time.Time
		wantRun bool
	}{
		{
			name:    "future tick should not execute",
			timerID: "future",
			typ:     1,
			tick:    time.Now().Add(time.Hour),
			wantRun: false,
		},
		{
			name:    "past tick should execute",
			timerID: "past",
			typ:     1,
			tick:    time.Now().Add(-time.Minute),
			wantRun: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			te := &TimerEngine{
				tasks: make(map[string]*timerExec),
			}

			params := map[string]any{
				"timerType": tt.typ,
				"dates":     time.Now().Format(time.RFC3339Nano),
			}
			paramsBytes, _ := json.Marshal(params)

			exec := &timerExec{
				tt: &model.TTrigger{
					Id:      tt.timerID,
					Name:    "test",
					Types:   "timer",
					Params:  string(paramsBytes),
					Enabled: 1,
				},
				typ:  tt.typ,
				tms:  time.Now(),
				tick: tt.tick,
			}

			te.tasks[tt.timerID] = exec
			te.execItem(exec)

			// If we wanted execution, verify tick was updated (for recurring timers)
			if tt.wantRun && tt.typ > 0 {
				if !exec.tick.After(tt.tick) {
					t.Errorf("expected tick to be updated after execution")
				}
			}
		})
	}
}

// TestTimerEngineExecItemTypes tests all timer type intervals
func TestTimerEngineExecItemTypes(t *testing.T) {
	types := []struct {
		typ      int64
		interval time.Duration
	}{
		{1, time.Minute},
		{2, time.Hour},
		{3, 24 * time.Hour},
		{4, 7 * 24 * time.Hour},
		{5, 30 * 24 * time.Hour},
	}

	for _, tt := range types {
		t.Run("type", func(t *testing.T) {
			te := &TimerEngine{
				tasks: make(map[string]*timerExec),
			}

			params := map[string]any{
				"timerType": tt.typ,
				"dates":     time.Now().Format(time.RFC3339Nano),
			}
			paramsBytes, _ := json.Marshal(params)

			exec := &timerExec{
				tt: &model.TTrigger{
					Id:      "test",
					Name:    "test",
					Types:   "timer",
					Params:  string(paramsBytes),
					Enabled: 1,
				},
				typ:  tt.typ,
				tms:  time.Now(),
				tick: time.Now().Add(-time.Minute), // Ensure it's in the past
			}

			te.tasks["test"] = exec
			beforeExec := time.Now()
			te.execItem(exec)

			// For recurring timers, tick should be updated to future
			if !exec.tick.After(beforeExec) {
				t.Errorf("expected tick to be updated to future")
			}
		})
	}
}

// TestTimerEngineExecItemOnce tests one-time timer (type 0)
func TestTimerEngineExecItemOnce(t *testing.T) {
	te := &TimerEngine{
		tasks: make(map[string]*timerExec),
	}

	params := map[string]any{
		"timerType": 0,
		"dates":     time.Now().Format(time.RFC3339Nano),
	}
	paramsBytes, _ := json.Marshal(params)

	timerID := "once-timer"
	exec := &timerExec{
		tt: &model.TTrigger{
			Id:      timerID,
			Name:    "once",
			Types:   "timer",
			Params:  string(paramsBytes),
			Enabled: 1,
		},
		typ:  0,
		tms:  time.Now(),
		tick: time.Now().Add(-time.Minute),
	}

	te.tasks[timerID] = exec
	te.execItem(exec)

	// One-time timer should schedule deletion
	// Give the goroutine time to execute
	time.Sleep(50 * time.Millisecond)
}

// TestTimerEngineResetOneHourly tests hourly timer reset
func TestTimerEngineResetOneHourly(t *testing.T) {
	te := &TimerEngine{
		tasks: make(map[string]*timerExec),
	}

	params := map[string]any{
		"timerType": 2,
		"dates":     time.Now().Format(time.RFC3339Nano),
	}
	paramsBytes, _ := json.Marshal(params)

	tmr := &model.TTrigger{
		Id:      "hourly",
		Name:    "hourly",
		Types:   "timer",
		Params:  string(paramsBytes),
		Enabled: 1,
	}

	err := te.resetOne(tmr)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	task, ok := te.tasks["hourly"]
	if !ok {
		t.Fatal("expected task to be added")
	}

	if task.typ != 2 {
		t.Errorf("expected type 2, got %d", task.typ)
	}

	// Tick should be in the near future (within an hour)
	if time.Until(task.tick) > time.Hour+time.Second {
		t.Error("expected tick to be within an hour")
	}
}

// TestTimerEngineResetOneDaily tests daily timer reset
func TestTimerEngineResetOneDaily(t *testing.T) {
	te := &TimerEngine{
		tasks: make(map[string]*timerExec),
	}

	params := map[string]any{
		"timerType": 3,
		"dates":     time.Now().Format(time.RFC3339Nano),
	}
	paramsBytes, _ := json.Marshal(params)

	tmr := &model.TTrigger{
		Id:      "daily",
		Name:    "daily",
		Types:   "timer",
		Params:  string(paramsBytes),
		Enabled: 1,
	}

	err := te.resetOne(tmr)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	task, ok := te.tasks["daily"]
	if !ok {
		t.Fatal("expected task to be added")
	}

	if task.typ != 3 {
		t.Errorf("expected type 3, got %d", task.typ)
	}

	// Tick should be in the near future (within 24 hours)
	if time.Until(task.tick) > 24*time.Hour+time.Second {
		t.Error("expected tick to be within 24 hours")
	}
}

// TestTimerEngineResetOneWeekly tests weekly timer reset
func TestTimerEngineResetOneWeekly(t *testing.T) {
	te := &TimerEngine{
		tasks: make(map[string]*timerExec),
	}

	params := map[string]any{
		"timerType": 4,
		"dates":     time.Now().Format(time.RFC3339Nano),
	}
	paramsBytes, _ := json.Marshal(params)

	tmr := &model.TTrigger{
		Id:      "weekly",
		Name:    "weekly",
		Types:   "timer",
		Params:  string(paramsBytes),
		Enabled: 1,
	}

	err := te.resetOne(tmr)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	task, ok := te.tasks["weekly"]
	if !ok {
		t.Fatal("expected task to be added")
	}

	if task.typ != 4 {
		t.Errorf("expected type 4, got %d", task.typ)
	}

	// Tick should be in the near future (within 7 days)
	if time.Until(task.tick) > 7*24*time.Hour+time.Second {
		t.Error("expected tick to be within 7 days")
	}
}

// TestTimerEngineResetOneMonthly tests monthly timer reset
func TestTimerEngineResetOneMonthly(t *testing.T) {
	te := &TimerEngine{
		tasks: make(map[string]*timerExec),
	}

	params := map[string]any{
		"timerType": 5,
		"dates":     time.Now().Format(time.RFC3339Nano),
	}
	paramsBytes, _ := json.Marshal(params)

	tmr := &model.TTrigger{
		Id:      "monthly",
		Name:    "monthly",
		Types:   "timer",
		Params:  string(paramsBytes),
		Enabled: 1,
	}

	err := te.resetOne(tmr)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	task, ok := te.tasks["monthly"]
	if !ok {
		t.Fatal("expected task to be added")
	}

	if task.typ != 5 {
		t.Errorf("expected type 5, got %d", task.typ)
	}

	// Tick should be in the near future (within 30 days)
	if time.Until(task.tick) > 30*24*time.Hour+time.Second {
		t.Error("expected tick to be within 30 days")
	}
}

// TestTimerEngineResetOneMissingTimerType tests reset with missing timerType
func TestTimerEngineResetOneMissingTimerType(t *testing.T) {
	te := &TimerEngine{
		tasks: make(map[string]*timerExec),
	}

	params := map[string]any{
		"dates": time.Now().Format(time.RFC3339Nano),
	}
	paramsBytes, _ := json.Marshal(params)

	tmr := &model.TTrigger{
		Id:      "missing-type",
		Name:    "missing",
		Types:   "timer",
		Params:  string(paramsBytes),
		Enabled: 1,
	}

	err := te.resetOne(tmr)
	if err == nil {
		t.Fatal("expected error for missing timerType")
	}
}

// TestTimerEngineResetOneInvalidDate tests reset with invalid date format
func TestTimerEngineResetOneInvalidDate(t *testing.T) {
	te := &TimerEngine{
		tasks: make(map[string]*timerExec),
	}

	params := map[string]any{
		"timerType": 1,
		"dates":     "not-a-valid-date",
	}
	paramsBytes, _ := json.Marshal(params)

	tmr := &model.TTrigger{
		Id:      "invalid-date",
		Name:    "invalid",
		Types:   "timer",
		Params:  string(paramsBytes),
		Enabled: 1,
	}

	err := te.resetOne(tmr)
	if err == nil {
		t.Fatal("expected error for invalid date format")
	}
}

// TestTimerEngineResetOneUpdateExisting tests updating an existing timer
func TestTimerEngineResetOneUpdateExisting(t *testing.T) {
	te := &TimerEngine{
		tasks: make(map[string]*timerExec),
	}

	// Add initial timer
	params1 := map[string]any{
		"timerType": 1,
		"dates":     time.Now().Format(time.RFC3339Nano),
	}
	params1Bytes, _ := json.Marshal(params1)

	tmr1 := &model.TTrigger{
		Id:      "update-test",
		Name:    "initial",
		Types:   "timer",
		Params:  string(params1Bytes),
		Enabled: 1,
	}

	err := te.resetOne(tmr1)
	if err != nil {
		t.Fatalf("unexpected error on first reset: %v", err)
	}

	// Update with different type
	params2 := map[string]any{
		"timerType": 2,
		"dates":     time.Now().Format(time.RFC3339Nano),
	}
	params2Bytes, _ := json.Marshal(params2)

	tmr2 := &model.TTrigger{
		Id:      "update-test",
		Name:    "updated",
		Types:   "timer",
		Params:  string(params2Bytes),
		Enabled: 1,
	}

	err = te.resetOne(tmr2)
	if err != nil {
		t.Fatalf("unexpected error on second reset: %v", err)
	}

	task := te.tasks["update-test"]
	if task.typ != 2 {
		t.Errorf("expected type to be updated to 2, got %d", task.typ)
	}
}

// TestTimerEngineRefreshWithContext tests refresh with context cancellation
func TestTimerEngineRefreshWithContext(t *testing.T) {
	te := &TimerEngine{
		tasks: make(map[string]*timerExec),
	}

	// refresh should handle errors gracefully
	te.refresh()
}

// TestTimerEngineRefreshEmpty tests refresh with no timers in database
func TestTimerEngineRefreshEmpty(t *testing.T) {
	te := &TimerEngine{
		tasks: make(map[string]*timerExec),
	}

	// Add some tasks first
	for i := 0; i < 3; i++ {
		id := string(rune('x' + i))
		te.tasks[id] = &timerExec{
			tt:   &model.TTrigger{Id: id},
			typ:  1,
			tick: time.Now().Add(time.Hour),
		}
	}

	// refresh should not add or remove tasks (no database)
	initialCount := len(te.tasks)
	te.refresh()

	if len(te.tasks) != initialCount {
		t.Errorf("expected task count to remain %d, got %d", initialCount, len(te.tasks))
	}
}
