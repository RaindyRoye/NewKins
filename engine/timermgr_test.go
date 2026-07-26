package engine

import (
	"context"
	"encoding/json"
	"sync"
	"testing"
	"time"

	"github.com/gokins/gokins/model"
)

func TestTimerEngine_NewForTest(t *testing.T) {
	te := &TimerEngine{
		tasks: make(map[string]*timerExec),
	}
	if te.tasks == nil {
		t.Fatal("tasks map should not be nil")
	}
}

func TestTimerEngine_Delete(t *testing.T) {
	te := &TimerEngine{
		tasks: make(map[string]*timerExec),
	}
	te.tasks["timer-1"] = &timerExec{
		tt: &model.TTrigger{Id: "timer-1"},
	}
	te.Delete("timer-1")
	if len(te.tasks) != 0 {
		t.Errorf("expected 0 tasks, got %d", len(te.tasks))
	}
}

func TestTimerEngine_Refresh_EmptyId(t *testing.T) {
	te := &TimerEngine{
		tasks: make(map[string]*timerExec),
	}
	err := te.Refresh(context.Background(), "")
	if err == nil {
		t.Fatal("expected error for empty id")
	}
}

func TestTimerEngine_ResetOne_NotTimer(t *testing.T) {
	te := &TimerEngine{
		tasks: make(map[string]*timerExec),
	}
	tmr := &model.TTrigger{
		Id:    "trigger-1",
		Types: "webhook",
	}
	err := te.resetOne(tmr)
	if err == nil {
		t.Fatal("expected error for non-timer type")
	}
}

func TestTimerEngine_ResetOne_InvalidParams(t *testing.T) {
	te := &TimerEngine{
		tasks: make(map[string]*timerExec),
	}
	tmr := &model.TTrigger{
		Id:     "trigger-1",
		Types:  "timer",
		Params: "invalid json",
	}
	err := te.resetOne(tmr)
	if err == nil {
		t.Fatal("expected error for invalid params")
	}
}

func TestTimerEngine_ResetOne_MissingTimerType(t *testing.T) {
	te := &TimerEngine{
		tasks: make(map[string]*timerExec),
	}
	params := map[string]any{
		"dates": time.Now().Format(time.RFC3339Nano),
	}
	data, _ := json.Marshal(params)
	tmr := &model.TTrigger{
		Id:     "trigger-1",
		Types:  "timer",
		Params: string(data),
	}
	err := te.resetOne(tmr)
	if err == nil {
		t.Fatal("expected error for missing timerType")
	}
}

func TestTimerEngine_ResetOne_InvalidDates(t *testing.T) {
	te := &TimerEngine{
		tasks: make(map[string]*timerExec),
	}
	params := map[string]any{
		"timerType": 1,
		"dates":     "invalid date",
	}
	data, _ := json.Marshal(params)
	tmr := &model.TTrigger{
		Id:     "trigger-1",
		Types:  "timer",
		Params: string(data),
	}
	err := te.resetOne(tmr)
	if err == nil {
		t.Fatal("expected error for invalid dates")
	}
}

func TestTimerEngine_ResetOne_OneTimeTimer(t *testing.T) {
	te := &TimerEngine{
		tasks: make(map[string]*timerExec),
	}
	future := time.Now().Add(time.Hour)
	params := map[string]any{
		"timerType": 0,
		"dates":     future.Format(time.RFC3339Nano),
	}
	data, _ := json.Marshal(params)
	tmr := &model.TTrigger{
		Id:     "trigger-1",
		Name:   "test-timer",
		Types:  "timer",
		Params: string(data),
	}
	err := te.resetOne(tmr)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(te.tasks) != 1 {
		t.Errorf("expected 1 task, got %d", len(te.tasks))
	}
	if te.tasks["trigger-1"] == nil {
		t.Error("expected timer-1 in tasks")
	}
}

func TestTimerEngine_ResetOne_MinuteTimer(t *testing.T) {
	te := &TimerEngine{
		tasks: make(map[string]*timerExec),
	}
	params := map[string]any{
		"timerType": 1,
		"dates":     time.Now().Format(time.RFC3339Nano),
	}
	data, _ := json.Marshal(params)
	tmr := &model.TTrigger{
		Id:     "trigger-2",
		Name:   "minute-timer",
		Types:  "timer",
		Params: string(data),
	}
	err := te.resetOne(tmr)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(te.tasks) != 1 {
		t.Errorf("expected 1 task, got %d", len(te.tasks))
	}
}

func TestTimerEngine_ResetOne_HourTimer(t *testing.T) {
	te := &TimerEngine{
		tasks: make(map[string]*timerExec),
	}
	params := map[string]any{
		"timerType": 2,
		"dates":     time.Now().Format(time.RFC3339Nano),
	}
	data, _ := json.Marshal(params)
	tmr := &model.TTrigger{
		Id:     "trigger-3",
		Types:  "timer",
		Params: string(data),
	}
	err := te.resetOne(tmr)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestTimerEngine_ResetOne_DayTimer(t *testing.T) {
	te := &TimerEngine{
		tasks: make(map[string]*timerExec),
	}
	params := map[string]any{
		"timerType": 3,
		"dates":     time.Now().Format(time.RFC3339Nano),
	}
	data, _ := json.Marshal(params)
	tmr := &model.TTrigger{
		Id:     "trigger-4",
		Types:  "timer",
		Params: string(data),
	}
	err := te.resetOne(tmr)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestTimerEngine_ResetOne_WeekTimer(t *testing.T) {
	te := &TimerEngine{
		tasks: make(map[string]*timerExec),
	}
	params := map[string]any{
		"timerType": 4,
		"dates":     time.Now().Format(time.RFC3339Nano),
	}
	data, _ := json.Marshal(params)
	tmr := &model.TTrigger{
		Id:     "trigger-5",
		Types:  "timer",
		Params: string(data),
	}
	err := te.resetOne(tmr)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestTimerEngine_ResetOne_MonthTimer(t *testing.T) {
	te := &TimerEngine{
		tasks: make(map[string]*timerExec),
	}
	params := map[string]any{
		"timerType": 5,
		"dates":     time.Now().Format(time.RFC3339Nano),
	}
	data, _ := json.Marshal(params)
	tmr := &model.TTrigger{
		Id:     "trigger-6",
		Types:  "timer",
		Params: string(data),
	}
	err := te.resetOne(tmr)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestTimerEngine_ResetOne_UpdateExisting(t *testing.T) {
	te := &TimerEngine{
		tasks: make(map[string]*timerExec),
	}
	// Add initial timer
	te.tasks["trigger-1"] = &timerExec{
		tt: &model.TTrigger{Id: "trigger-1"},
	}
	// Update it
	future := time.Now().Add(time.Hour)
	params := map[string]any{
		"timerType": 0,
		"dates":     future.Format(time.RFC3339Nano),
	}
	data, _ := json.Marshal(params)
	tmr := &model.TTrigger{
		Id:     "trigger-1",
		Types:  "timer",
		Params: string(data),
	}
	err := te.resetOne(tmr)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(te.tasks) != 1 {
		t.Errorf("expected 1 task, got %d", len(te.tasks))
	}
}

func TestTimerEngine_Run_Empty(t *testing.T) {
	te := &TimerEngine{
		tasks: make(map[string]*timerExec),
	}
	// Should not panic
	te.run()
}

func TestTimerEngine_ExecItem_NotReady(t *testing.T) {
	te := &TimerEngine{
		tasks: make(map[string]*timerExec),
	}
	future := time.Now().Add(time.Hour)
	te.tasks["trigger-1"] = &timerExec{
		tt:   &model.TTrigger{Id: "trigger-1"},
		tick: future,
	}
	// Should not execute (not ready yet)
	te.execItem(te.tasks["trigger-1"])
}

func TestBuildEngine_NewForTest(t *testing.T) {
	be := NewBuildEngineForTest()
	if be == nil {
		t.Fatal("expected non-nil BuildEngine")
	}
	if be.taskw == nil {
		t.Error("taskw should not be nil")
	}
	if be.tasks == nil {
		t.Error("tasks should not be nil")
	}
}

func TestBuildEngine_Get_Empty(t *testing.T) {
	be := NewBuildEngineForTest()
	_, ok := be.Get("")
	if ok {
		t.Error("expected false for empty id")
	}
}

func TestBuildEngine_Get_NotFound(t *testing.T) {
	be := NewBuildEngineForTest()
	_, ok := be.Get("nonexistent")
	if ok {
		t.Error("expected false for nonexistent id")
	}
}

func TestBuildEngine_Stop_Empty(t *testing.T) {
	be := NewBuildEngineForTest()
	// Should not panic
	be.Stop()
}

func TestBuildEngine_ConcurrentAccess(t *testing.T) {
	be := NewBuildEngineForTest()
	var wg sync.WaitGroup
	// Concurrent reads
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			be.Get("test-id")
		}(i)
	}
	wg.Wait()
}

func TestBuildEngine_Run_Empty(t *testing.T) {
	be := NewBuildEngineForTest()
	// Should not panic
	be.run()
}
