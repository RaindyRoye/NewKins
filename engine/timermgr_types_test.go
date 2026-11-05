package engine

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/gokins/gokins/model"
)

// TestTimerEngineResetOneHourTimer verifies type=2 (hourly) timer.
func TestTimerEngineResetOneHourTimer(t *testing.T) {
	te := &TimerEngine{tasks: make(map[string]*timerExec)}
	params := map[string]any{
		"timerType": 2,
		"dates":     time.Now().Format(time.RFC3339Nano),
	}
	b, _ := json.Marshal(params)
	tmr := &model.TTrigger{
		Id: "hour-1", Types: "timer", Name: "hour", Params: string(b),
	}
	if err := te.resetOne(tmr); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	task, ok := te.tasks["hour-1"]
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

// TestTimerEngineResetOneDailyTimer verifies type=3 (daily) timer.
func TestTimerEngineResetOneDailyTimer(t *testing.T) {
	te := &TimerEngine{tasks: make(map[string]*timerExec)}
	params := map[string]any{
		"timerType": 3,
		"dates":     time.Now().Format(time.RFC3339Nano),
	}
	b, _ := json.Marshal(params)
	tmr := &model.TTrigger{
		Id: "daily-1", Types: "timer", Name: "daily", Params: string(b),
	}
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
	if time.Until(task.tick) > 25*time.Hour {
		t.Fatal("tick should be within ~24 hours")
	}
}

// TestTimerEngineResetOneWeeklyTimer verifies type=4 (weekly) timer.
func TestTimerEngineResetOneWeeklyTimer(t *testing.T) {
	te := &TimerEngine{tasks: make(map[string]*timerExec)}
	params := map[string]any{
		"timerType": 4,
		"dates":     time.Now().Format(time.RFC3339Nano),
	}
	b, _ := json.Marshal(params)
	tmr := &model.TTrigger{
		Id: "weekly-1", Types: "timer", Name: "weekly", Params: string(b),
	}
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
	if time.Until(task.tick) > 8*24*time.Hour {
		t.Fatal("tick should be within ~7 days")
	}
}

// TestTimerEngineResetOneMonthlyTimer verifies type=5 (monthly) timer.
func TestTimerEngineResetOneMonthlyTimer(t *testing.T) {
	te := &TimerEngine{tasks: make(map[string]*timerExec)}
	params := map[string]any{
		"timerType": 5,
		"dates":     time.Now().Format(time.RFC3339Nano),
	}
	b, _ := json.Marshal(params)
	tmr := &model.TTrigger{
		Id: "monthly-1", Types: "timer", Name: "monthly", Params: string(b),
	}
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
	if time.Until(task.tick) > 31*24*time.Hour {
		t.Fatal("tick should be within ~30 days")
	}
}

// TestTimerEngineResetOnePastOnceTimer verifies that a one-time timer set in
// the past is NOT registered.
func TestTimerEngineResetOnePastOnceTimer(t *testing.T) {
	te := &TimerEngine{tasks: make(map[string]*timerExec)}
	past := time.Now().Add(-time.Hour)
	params := map[string]any{
		"timerType": 0,
		"dates":     past.Format(time.RFC3339Nano),
	}
	b, _ := json.Marshal(params)
	tmr := &model.TTrigger{
		Id: "past-once", Types: "timer", Name: "past", Params: string(b),
	}
	if err := te.resetOne(tmr); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, ok := te.tasks["past-once"]; ok {
		t.Fatal("past one-time timer should not be registered")
	}
}

// TestTimerEngineResetOneMissingTimerType verifies error when timerType is absent.
func TestTimerEngineResetOneMissingTimerType(t *testing.T) {
	te := &TimerEngine{tasks: make(map[string]*timerExec)}
	params := map[string]any{
		"dates": time.Now().Format(time.RFC3339Nano),
	}
	b, _ := json.Marshal(params)
	tmr := &model.TTrigger{
		Id: "missing-type", Types: "timer", Name: "no-type", Params: string(b),
	}
	err := te.resetOne(tmr)
	if err == nil {
		t.Fatal("expected error when timerType is missing")
	}
}

// TestTimerEngineResetOneInvalidDates verifies error when dates cannot be parsed.
func TestTimerEngineResetOneInvalidDates(t *testing.T) {
	te := &TimerEngine{tasks: make(map[string]*timerExec)}
	params := map[string]any{
		"timerType": 1,
		"dates":     "not-a-date",
	}
	b, _ := json.Marshal(params)
	tmr := &model.TTrigger{
		Id: "bad-dates", Types: "timer", Name: "bad", Params: string(b),
	}
	err := te.resetOne(tmr)
	if err == nil {
		t.Fatal("expected error when dates cannot be parsed")
	}
}

// TestTimerEngineResetOneUpdateExisting verifies that resetting an existing
// timer updates its fields rather than creating a duplicate.
func TestTimerEngineResetOneUpdateExisting(t *testing.T) {
	te := &TimerEngine{tasks: make(map[string]*timerExec)}

	// First reset
	params1 := map[string]any{
		"timerType": 1,
		"dates":     time.Now().Format(time.RFC3339Nano),
	}
	b1, _ := json.Marshal(params1)
	tmr1 := &model.TTrigger{
		Id: "upd-1", Types: "timer", Name: "upd", Params: string(b1),
	}
	if err := te.resetOne(tmr1); err != nil {
		t.Fatalf("first resetOne: %v", err)
	}
	firstTick := te.tasks["upd-1"].tick

	// Wait a tiny bit so tick changes
	time.Sleep(10 * time.Millisecond)

	// Second reset with type 2
	params2 := map[string]any{
		"timerType": 2,
		"dates":     time.Now().Format(time.RFC3339Nano),
	}
	b2, _ := json.Marshal(params2)
	tmr2 := &model.TTrigger{
		Id: "upd-1", Types: "timer", Name: "upd", Params: string(b2),
	}
	if err := te.resetOne(tmr2); err != nil {
		t.Fatalf("second resetOne: %v", err)
	}

	task := te.tasks["upd-1"]
	if task.typ != 2 {
		t.Fatalf("expected type 2 after update, got %d", task.typ)
	}
	if task.tick.Equal(firstTick) {
		t.Fatal("expected tick to change after update")
	}
	if len(te.tasks) != 1 {
		t.Fatalf("expected 1 task, got %d", len(te.tasks))
	}
}

// TestTimerEngineExecItemOnceDelete verifies execItem for a one-shot (type=0)
// timer triggers deletion via a goroutine. The service.TriggerTimer call will
// fail without a DB, but util.RecoverLog should prevent a crash.
func TestTimerEngineExecItemOnceDelete(t *testing.T) {
	te := &TimerEngine{tasks: make(map[string]*timerExec)}
	tt := &model.TTrigger{Id: "exec-once", Name: "test"}
	te.tasks["exec-once"] = &timerExec{
		tt:   tt,
		typ:  0,
		tick: time.Now().Add(-time.Second),
	}

	// New signature: execItem(tt, typ, id)
	te.execItem(tt, 0, "exec-once")

	// Give the deletion goroutine time to run
	time.Sleep(100 * time.Millisecond)
	te.tasklk.RLock()
	_, exists := te.tasks["exec-once"]
	te.tasklk.RUnlock()
	if exists {
		t.Error("one-shot timer should have been deleted after execItem")
	}
}

// TestTimerEngineExecItemRepeatingTick verifies execItem for a repeating (type=1)
// timer reschedules the next tick to approximately now+1 minute.
func TestTimerEngineExecItemRepeatingTick(t *testing.T) {
	te := &TimerEngine{
		tasks:  make(map[string]*timerExec),
		stopCh: make(chan struct{}),
		wakeCh: make(chan struct{}, 1),
	}
	tt := &model.TTrigger{Id: "repeat", Name: "test"}
	te.tasks["repeat"] = &timerExec{
		tt:   tt,
		typ:  1,
		tick: time.Now().Add(-time.Second),
	}

	before := time.Now()
	te.execItem(tt, 1, "repeat")

	te.tasklk.RLock()
	task, ok := te.tasks["repeat"]
	te.tasklk.RUnlock()
	if !ok {
		t.Fatal("repeating timer should still exist after execItem")
	}
	// Next tick should be ~now + 1 minute
	nextExpected := before.Add(time.Minute)
	diff := task.tick.Sub(nextExpected)
	if diff < -2*time.Second || diff > 2*time.Second {
		t.Errorf("expected next tick near %v, got %v (diff %v)", nextExpected, task.tick, diff)
	}
}
