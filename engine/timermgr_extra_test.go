package engine

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/gokins/gokins/comm"
	"github.com/gokins/gokins/model"
	_ "github.com/mattn/go-sqlite3"
	"xorm.io/xorm"
)

func setupTimerTestDB(t *testing.T) {
	t.Helper()
	origDb := comm.Db
	t.Cleanup(func() { comm.Db = origDb })

	db, err := xorm.NewEngine("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("failed to init test DB: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	// Sync the TTrigger model to create the table
	err = db.Sync2(&model.TTrigger{}, &model.TBuild{})
	if err != nil {
		t.Fatalf("failed to sync schema: %v", err)
	}

	comm.Db = db
}

func TestTimerEngine_Refresh_WithDB(t *testing.T) {
	setupTimerTestDB(t)

	te := &TimerEngine{
		tasks: make(map[string]*timerExec),
	}

	// Insert a timer trigger
	futureTime := time.Now().Add(time.Hour)
	params := map[string]any{
		"timerType": 1,
		"dates":     futureTime.Format(time.RFC3339Nano),
	}
	paramsBytes, _ := json.Marshal(params)

	trigger := &model.TTrigger{
		Id:      "timer-1",
		Aid:     1, // Set aid explicitly since it's a composite PK
		Types:   "timer",
		Name:    "test-timer",
		Params:  string(paramsBytes),
		Enabled: 1,
		Created: time.Now(),
		Updated: time.Now(),
	}
	_, err := comm.Db.InsertOne(trigger)
	if err != nil {
		t.Fatalf("failed to insert trigger: %v", err)
	}

	// Call refresh - should load from DB
	te.refresh()

	// Verify task was created
	te.tasklk.RLock()
	_, exists := te.tasks["timer-1"]
	te.tasklk.RUnlock()

	if !exists {
		t.Error("expected timer-1 to be loaded into tasks")
	}
}

func TestTimerEngine_Refresh_DBError(t *testing.T) {
	// Don't setup DB - should handle error gracefully
	origDb := comm.Db
	comm.Db = nil
	t.Cleanup(func() { comm.Db = origDb })

	te := &TimerEngine{
		tasks: make(map[string]*timerExec),
	}

	// Should not panic even without DB
	te.refresh()

	// Tasks should still be empty
	if len(te.tasks) != 0 {
		t.Errorf("expected 0 tasks, got %d", len(te.tasks))
	}
}

func TestTimerEngine_ExecItem_OnceTimer(t *testing.T) {
	setupTimerTestDB(t)

	te := &TimerEngine{
		tasks: make(map[string]*timerExec),
	}

	// Create a one-time timer that's due
	pastTime := time.Now().Add(-time.Minute)
	params := map[string]any{
		"timerType": 0, // once
		"dates":     pastTime.Format(time.RFC3339Nano),
	}
	paramsBytes, _ := json.Marshal(params)

	trigger := &model.TTrigger{
		Id:      "once-timer",
		Types:   "timer",
		Name:    "once-test",
		Params:  string(paramsBytes),
		Enabled: 1,
		Created: time.Now(),
	}

	task := &timerExec{
		tt:   trigger,
		typ:  0,
		tick: pastTime,
		tms:  pastTime,
	}
	te.tasks["once-timer"] = task

	// Execute - should trigger and schedule deletion
	te.execItem(task)

	// Wait a bit for the deletion goroutine
	time.Sleep(20 * time.Millisecond)

	// Task should be deleted
	te.tasklk.RLock()
	_, exists := te.tasks["once-timer"]
	te.tasklk.RUnlock()

	if exists {
		t.Error("expected once timer to be deleted after execution")
	}
}

func TestTimerEngine_ExecItem_MinuteTimer(t *testing.T) {
	setupTimerTestDB(t)

	te := &TimerEngine{
		tasks: make(map[string]*timerExec),
	}

	// Create a minute timer that's due
	pastTime := time.Now().Add(-time.Minute)
	params := map[string]any{
		"timerType": 1, // minute
		"dates":     pastTime.Format(time.RFC3339Nano),
	}
	paramsBytes, _ := json.Marshal(params)

	trigger := &model.TTrigger{
		Id:      "minute-timer",
		Types:   "timer",
		Name:    "minute-test",
		Params:  string(paramsBytes),
		Enabled: 1,
		Created: time.Now(),
	}

	task := &timerExec{
		tt:   trigger,
		typ:  1,
		tick: pastTime,
		tms:  pastTime,
	}
	te.tasks["minute-timer"] = task

	beforeTick := task.tick
	te.execItem(task)

	// Task should still exist (recurring)
	te.tasklk.RLock()
	exists := te.tasks["minute-timer"]
	te.tasklk.RUnlock()

	if exists == nil {
		t.Error("expected minute timer to still exist")
	}

	// Tick should be updated to next minute
	if !task.tick.After(beforeTick) {
		t.Error("expected tick to be updated to next minute")
	}
}

func TestTimerEngine_ExecItem_NotDue(t *testing.T) {
	te := &TimerEngine{
		tasks: make(map[string]*timerExec),
	}

	// Create a timer that's not yet due
	futureTime := time.Now().Add(time.Hour)
	trigger := &model.TTrigger{
		Id:      "future-timer",
		Types:   "timer",
		Name:    "future-test",
		Enabled: 1,
	}

	task := &timerExec{
		tt:   trigger,
		typ:  1,
		tick: futureTime,
		tms:  futureTime,
	}
	te.tasks["future-timer"] = task

	beforeTick := task.tick
	te.execItem(task)

	// Tick should not be updated (not due yet)
	if !task.tick.Equal(beforeTick) {
		t.Error("expected tick to remain unchanged when not due")
	}
}

func TestTimerEngine_ExecItem_AllTypes(t *testing.T) {
	setupTimerTestDB(t)

	te := &TimerEngine{
		tasks: make(map[string]*timerExec),
	}

	types := []int64{2, 3, 4, 5} // hour, day, week, month
	names := []string{"hour", "day", "week", "month"}

	for i, typ := range types {
		pastTime := time.Now().Add(-time.Hour * 24)
		params := map[string]any{
			"timerType": typ,
			"dates":     pastTime.Format(time.RFC3339Nano),
		}
		paramsBytes, _ := json.Marshal(params)

		trigger := &model.TTrigger{
			Id:      names[i] + "-timer",
			Types:   "timer",
			Name:    names[i] + "-test",
			Params:  string(paramsBytes),
			Enabled: 1,
			Created: time.Now(),
		}

		task := &timerExec{
			tt:   trigger,
			typ:  typ,
			tick: pastTime,
			tms:  pastTime,
		}
		te.tasks[trigger.Id] = task

		te.execItem(task)

		// Task should still exist (all are recurring)
		te.tasklk.RLock()
		exists := te.tasks[trigger.Id]
		te.tasklk.RUnlock()

		if exists == nil {
			t.Errorf("expected %s timer to still exist", names[i])
		}
	}
}

func TestTimerEngine_Run(t *testing.T) {
	setupTimerTestDB(t)

	te := &TimerEngine{
		tasks: make(map[string]*timerExec),
	}

	// Add multiple timers
	for i := 0; i < 3; i++ {
		futureTime := time.Now().Add(time.Hour)
		params := map[string]any{
			"timerType": 1,
			"dates":     futureTime.Format(time.RFC3339Nano),
		}
		paramsBytes, _ := json.Marshal(params)

		trigger := &model.TTrigger{
			Id:      "timer-" + string(rune('0'+i)),
			Types:   "timer",
			Name:    "test-" + string(rune('0'+i)),
			Params:  string(paramsBytes),
			Enabled: 1,
		}

		task := &timerExec{
			tt:   trigger,
			typ:  1,
			tick: futureTime,
			tms:  futureTime,
		}
		te.tasks[trigger.Id] = task
	}

	// Run should execute all tasks without panic
	te.run()

	// All tasks should still exist (not due)
	te.tasklk.RLock()
	count := len(te.tasks)
	te.tasklk.RUnlock()

	if count != 3 {
		t.Errorf("expected 3 tasks, got %d", count)
	}
}
