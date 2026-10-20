package engine

import (
	"container/list"
	"testing"
	"time"

	"github.com/gokins/gokins/model"
)

// --- TimerEngine.run tests ---

func TestTimerEngineRun_Empty(t *testing.T) {
	te := &TimerEngine{
		tasks: make(map[string]*timerExec),
	}
	// run with no tasks should not panic
	te.run()
}

func TestTimerEngineRun_WithFutureTask(t *testing.T) {
	te := &TimerEngine{
		tasks: make(map[string]*timerExec),
	}
	// Add a task that should execute in the future
	future := time.Now().Add(time.Hour)
	te.tasks["timer-1"] = &timerExec{
		tt:   &model.TTrigger{Id: "timer-1", Name: "test"},
		typ:  0,
		tms:  future,
		tick: future,
	}
	// run should not panic
	te.run()
	// Task should still exist (not yet time to execute)
	if _, ok := te.tasks["timer-1"]; !ok {
		t.Error("future task should still exist")
	}
}

func TestTimerEngineRun_WithPastTask(t *testing.T) {
	te := &TimerEngine{
		tasks: make(map[string]*timerExec),
	}
	// Add a task that should have executed
	past := time.Now().Add(-time.Minute)
	te.tasks["timer-2"] = &timerExec{
		tt:   &model.TTrigger{Id: "timer-2", Name: "test", Enabled: 1},
		typ:  0, // once
		tms:  past,
		tick: past,
	}
	// run should execute this task
	te.run()
	// Task should be deleted after execution (type 0 is one-time)
	time.Sleep(50 * time.Millisecond)
	if _, ok := te.tasks["timer-2"]; ok {
		t.Error("one-time task should be deleted after execution")
	}
}

// --- TimerEngine.execItem tests ---

func TestTimerEngineExecItem_OnceTimer(t *testing.T) {
	te := &TimerEngine{
		tasks: make(map[string]*timerExec),
	}
	past := time.Now().Add(-time.Minute)
	tmr := &timerExec{
		tt:   &model.TTrigger{Id: "timer-3", Name: "test", Enabled: 1},
		typ:  0,
		tms:  past,
		tick: past,
	}
	te.tasks["timer-3"] = tmr

	// execItem should schedule deletion
	te.execItem(tmr)
	time.Sleep(50 * time.Millisecond)
	// Task should be removed
	if _, ok := te.tasks["timer-3"]; ok {
		t.Error("one-time timer should be removed")
	}
}

func TestTimerEngineExecItem_MinuteTimer(t *testing.T) {
	te := &TimerEngine{
		tasks: make(map[string]*timerExec),
	}
	past := time.Now().Add(-time.Minute)
	tmr := &timerExec{
		tt:   &model.TTrigger{Id: "timer-4", Name: "test", Enabled: 1},
		typ:  1, // minute
		tms:  past,
		tick: past,
	}
	te.tasks["timer-4"] = tmr

	// execItem should reschedule to next minute
	te.execItem(tmr)
	// Task should still exist (recurring)
	if _, ok := te.tasks["timer-4"]; !ok {
		t.Error("recurring timer should still exist")
	}
	// Tick should be in the future
	if time.Until(tmr.tick) <= 0 {
		t.Error("tick should be rescheduled to future")
	}
}

func TestTimerEngineExecItem_HourlyTimer(t *testing.T) {
	te := &TimerEngine{
		tasks: make(map[string]*timerExec),
	}
	past := time.Now().Add(-time.Hour)
	tmr := &timerExec{
		tt:   &model.TTrigger{Id: "timer-5", Name: "test", Enabled: 1},
		typ:  2, // hourly
		tms:  past,
		tick: past,
	}
	te.tasks["timer-5"] = tmr

	te.execItem(tmr)
	if _, ok := te.tasks["timer-5"]; !ok {
		t.Error("hourly timer should still exist")
	}
	if time.Until(tmr.tick) <= 0 {
		t.Error("tick should be rescheduled to future")
	}
}

// --- TimerEngine.refresh tests ---

func TestTimerEngineRefresh_NoDatabase(t *testing.T) {
	te := &TimerEngine{
		tasks: make(map[string]*timerExec),
	}
	// refresh without database should not panic
	// (comm.Db is nil in test environment)
	te.refresh()
}

// --- TimerEngine integration with list ---

func TestTimerEngineWithList(t *testing.T) {
	te := &TimerEngine{
		tasks: make(map[string]*timerExec),
	}
	// Simulate the timer list structure
	_ = list.New()
	// Add multiple tasks
	for i := 0; i < 5; i++ {
		id := string(rune('a' + i))
		te.tasks[id] = &timerExec{
			tt:   &model.TTrigger{Id: id, Name: "test", Enabled: 1},
			typ:  int64(i % 6),
			tms:  time.Now(),
			tick: time.Now().Add(time.Hour),
		}
	}
	// run should not panic
	te.run()
	// All tasks should still exist (future ticks)
	if len(te.tasks) != 5 {
		t.Errorf("expected 5 tasks, got %d", len(te.tasks))
	}
}
