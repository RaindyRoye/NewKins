package engine

import (
	"encoding/json"
	"sync"
	"testing"
	"time"

	"github.com/gokins/gokins/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestTimerEngineRefresh_NoDatabase tests refresh without database connection.
// The recover() inside refresh() handles the nil pointer dereference from comm.Db gracefully.
func TestTimerEngineRefresh_NoDatabase(t *testing.T) {
	te := &TimerEngine{
		tasks: make(map[string]*timerExec),
	}
	te.refresh()
	assert.Equal(t, 0, len(te.tasks))
}

// TestTimerEngineRun_EmptyTasks tests run with empty task map
func TestTimerEngineRun_EmptyTasks(t *testing.T) {
	te := &TimerEngine{
		tasks: make(map[string]*timerExec),
	}
	te.run()
	assert.Equal(t, 0, len(te.tasks))
}

// TestTimerEngineRun_ExpiredOnceTimer tests that run triggers deletion of expired
// one-time timers. The deletion happens asynchronously in a goroutine, so we wait briefly.
func TestTimerEngineRun_ExpiredOnceTimer(t *testing.T) {
	te := &TimerEngine{
		tasks: make(map[string]*timerExec),
	}
	pastTime := time.Now().Add(-time.Hour)
	te.tasks["expired-once"] = &timerExec{
		tt: &model.TTrigger{
			Id:   "expired-once",
			Name: "expired",
		},
		typ:  0, // once
		tms:  pastTime,
		tick: pastTime,
	}
	te.run()
	// The timer deletion is async (goroutine + sleep), so wait briefly
	time.Sleep(50 * time.Millisecond)
	te.tasklk.RLock()
	_, exists := te.tasks["expired-once"]
	te.tasklk.RUnlock()
	assert.False(t, exists, "expired one-time timer should be deleted")
}

// TestTimerEngineRun_FutureTimer tests that future timers are not triggered
func TestTimerEngineRun_FutureTimer(t *testing.T) {
	te := &TimerEngine{
		tasks: make(map[string]*timerExec),
	}
	futureTime := time.Now().Add(time.Hour)
	te.tasks["future-timer"] = &timerExec{
		tt: &model.TTrigger{
			Id:   "future-timer",
			Name: "future",
		},
		typ:  1,
		tms:  futureTime,
		tick: futureTime,
	}
	te.run()
	te.tasklk.RLock()
	_, exists := te.tasks["future-timer"]
	te.tasklk.RUnlock()
	assert.True(t, exists, "future timer should not be deleted")
}

// TestTimerEngineResetOne_HourlyTimer tests resetOne with hourly timer type
func TestTimerEngineResetOne_HourlyTimer(t *testing.T) {
	te := &TimerEngine{
		tasks: make(map[string]*timerExec),
	}
	params := map[string]any{
		"timerType": 2,
		"dates":     time.Now().Format(time.RFC3339Nano),
	}
	paramBytes, _ := json.Marshal(params)
	tmr := &model.TTrigger{
		Id:     "hourly-timer",
		Types:  "timer",
		Name:   "hourly",
		Params: string(paramBytes),
	}
	err := te.resetOne(tmr)
	require.NoError(t, err)
	task, exists := te.tasks["hourly-timer"]
	require.True(t, exists)
	assert.Equal(t, int64(2), task.typ)
	assert.LessOrEqual(t, time.Until(task.tick), time.Hour+time.Second)
}

// TestTimerEngineResetOne_DailyTimer tests resetOne with daily timer type
func TestTimerEngineResetOne_DailyTimer(t *testing.T) {
	te := &TimerEngine{
		tasks: make(map[string]*timerExec),
	}
	params := map[string]any{
		"timerType": 3,
		"dates":     time.Now().Format(time.RFC3339Nano),
	}
	paramBytes, _ := json.Marshal(params)
	tmr := &model.TTrigger{
		Id:     "daily-timer",
		Types:  "timer",
		Name:   "daily",
		Params: string(paramBytes),
	}
	err := te.resetOne(tmr)
	require.NoError(t, err)
	task, exists := te.tasks["daily-timer"]
	require.True(t, exists)
	assert.Equal(t, int64(3), task.typ)
	assert.LessOrEqual(t, time.Until(task.tick), 24*time.Hour+time.Second)
}

// TestTimerEngineResetOne_WeeklyTimer tests resetOne with weekly timer type
func TestTimerEngineResetOne_WeeklyTimer(t *testing.T) {
	te := &TimerEngine{
		tasks: make(map[string]*timerExec),
	}
	params := map[string]any{
		"timerType": 4,
		"dates":     time.Now().Format(time.RFC3339Nano),
	}
	paramBytes, _ := json.Marshal(params)
	tmr := &model.TTrigger{
		Id:     "weekly-timer",
		Types:  "timer",
		Name:   "weekly",
		Params: string(paramBytes),
	}
	err := te.resetOne(tmr)
	require.NoError(t, err)
	task, exists := te.tasks["weekly-timer"]
	require.True(t, exists)
	assert.Equal(t, int64(4), task.typ)
	assert.LessOrEqual(t, time.Until(task.tick), 7*24*time.Hour+time.Second)
}

// TestTimerEngineResetOne_MonthlyTimer tests resetOne with monthly timer type
func TestTimerEngineResetOne_MonthlyTimer(t *testing.T) {
	te := &TimerEngine{
		tasks: make(map[string]*timerExec),
	}
	params := map[string]any{
		"timerType": 5,
		"dates":     time.Now().Format(time.RFC3339Nano),
	}
	paramBytes, _ := json.Marshal(params)
	tmr := &model.TTrigger{
		Id:     "monthly-timer",
		Types:  "timer",
		Name:   "monthly",
		Params: string(paramBytes),
	}
	err := te.resetOne(tmr)
	require.NoError(t, err)
	task, exists := te.tasks["monthly-timer"]
	require.True(t, exists)
	assert.Equal(t, int64(5), task.typ)
	assert.LessOrEqual(t, time.Until(task.tick), 30*24*time.Hour+time.Second)
}

// TestTimerEngineResetOne_UnsupportedTimerType tests that unsupported timer type numbers
// don't add entries to the task map (they fall through the switch).
func TestTimerEngineResetOne_UnsupportedTimerType(t *testing.T) {
	te := &TimerEngine{
		tasks: make(map[string]*timerExec),
	}
	params := map[string]any{
		"timerType": 99,
		"dates":     time.Now().Format(time.RFC3339Nano),
	}
	paramBytes, _ := json.Marshal(params)
	tmr := &model.TTrigger{
		Id:     "invalid-type",
		Types:  "timer",
		Name:   "invalid",
		Params: string(paramBytes),
	}
	err := te.resetOne(tmr)
	require.NoError(t, err)
	assert.Equal(t, 0, len(te.tasks), "unsupported timer type should not be added")
}

// TestTimerEngineResetOne_MissingTimerType tests resetOne with missing timerType field
func TestTimerEngineResetOne_MissingTimerType(t *testing.T) {
	te := &TimerEngine{
		tasks: make(map[string]*timerExec),
	}
	params := map[string]any{
		"dates": time.Now().Format(time.RFC3339Nano),
	}
	paramBytes, _ := json.Marshal(params)
	tmr := &model.TTrigger{
		Id:     "missing-type",
		Types:  "timer",
		Name:   "missing",
		Params: string(paramBytes),
	}
	err := te.resetOne(tmr)
	assert.Error(t, err)
}

// TestTimerEngineResetOne_InvalidDateFormat tests resetOne with invalid date format
func TestTimerEngineResetOne_InvalidDateFormat(t *testing.T) {
	te := &TimerEngine{
		tasks: make(map[string]*timerExec),
	}
	params := map[string]any{
		"timerType": 1,
		"dates":     "not-a-valid-date",
	}
	paramBytes, _ := json.Marshal(params)
	tmr := &model.TTrigger{
		Id:     "invalid-date",
		Types:  "timer",
		Name:   "invalid",
		Params: string(paramBytes),
	}
	err := te.resetOne(tmr)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "parse dates")
}

// TestTimerEngineResetOne_UpdateExistingTimer tests resetOne updates existing timer in place
func TestTimerEngineResetOne_UpdateExistingTimer(t *testing.T) {
	te := &TimerEngine{
		tasks: make(map[string]*timerExec),
	}
	futureTime := time.Now().Add(time.Hour)
	params := map[string]any{
		"timerType": 1,
		"dates":     futureTime.Format(time.RFC3339Nano),
	}
	paramBytes, _ := json.Marshal(params)
	tmr := &model.TTrigger{
		Id:     "update-timer",
		Types:  "timer",
		Name:   "update",
		Params: string(paramBytes),
	}
	err := te.resetOne(tmr)
	require.NoError(t, err)
	require.Equal(t, 1, len(te.tasks))

	// Update with a different type (hourly)
	params["timerType"] = 2
	newFutureTime := time.Now().Add(2 * time.Hour)
	params["dates"] = newFutureTime.Format(time.RFC3339Nano)
	paramBytes, _ = json.Marshal(params)
	tmr.Params = string(paramBytes)

	err = te.resetOne(tmr)
	require.NoError(t, err)
	assert.Equal(t, 1, len(te.tasks))
	assert.Equal(t, int64(2), te.tasks["update-timer"].typ)
}

// TestTimerEngineResetOne_OnceTimerPastSkipped tests that a once-timer set in the past
// is NOT added to the task map.
func TestTimerEngineResetOne_OnceTimerPastSkipped(t *testing.T) {
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
	require.NoError(t, err)
	assert.Equal(t, 0, len(te.tasks), "past once-timer should not be added")
}

// TestTimerEngineResetOne_WhitespaceDates tests that whitespace-only dates are rejected
func TestTimerEngineResetOne_WhitespaceDates(t *testing.T) {
	te := &TimerEngine{
		tasks: make(map[string]*timerExec),
	}
	params := map[string]any{
		"timerType": 1,
		"dates":     "   ",
	}
	paramBytes, _ := json.Marshal(params)
	tmr := &model.TTrigger{
		Id:     "ws-dates",
		Types:  "timer",
		Name:   "ws",
		Params: string(paramBytes),
	}
	err := te.resetOne(tmr)
	assert.Error(t, err)
}

// TestTimerEngineResetOne_EmptyParams tests resetOne with empty JSON object
func TestTimerEngineResetOne_EmptyParams(t *testing.T) {
	te := &TimerEngine{
		tasks: make(map[string]*timerExec),
	}
	tmr := &model.TTrigger{
		Id:     "empty-params",
		Types:  "timer",
		Name:   "empty",
		Params: `{}`,
	}
	err := te.resetOne(tmr)
	assert.Error(t, err)
}

// TestTimerEngineConcurrentResetOne_OnceTimer tests concurrent resetOne for once timers
func TestTimerEngineConcurrentResetOne_OnceTimer(t *testing.T) {
	te := &TimerEngine{
		tasks: make(map[string]*timerExec),
	}
	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			futureTime := time.Now().Add(time.Hour)
			params := map[string]any{
				"timerType": 0,
				"dates":     futureTime.Format(time.RFC3339Nano),
			}
			paramBytes, _ := json.Marshal(params)
			tmr := &model.TTrigger{
				Id:     "once-concurrent",
				Types:  "timer",
				Name:   "once",
				Params: string(paramBytes),
			}
			_ = te.resetOne(tmr)
		}()
	}
	wg.Wait()
	assert.LessOrEqual(t, len(te.tasks), 1)
}

// TestTimerEngineRun_RecurringTimerAdvancesTick tests that a recurring minute timer
// advances its tick after execution.
func TestTimerEngineRun_RecurringTimerAdvancesTick(t *testing.T) {
	te := &TimerEngine{
		tasks: make(map[string]*timerExec),
	}
	// Set a minute timer with tick in the past so it triggers
	pastTick := time.Now().Add(-time.Second)
	te.tasks["recurring-minute"] = &timerExec{
		tt: &model.TTrigger{
			Id:   "recurring-minute",
			Name: "recurring",
		},
		typ:  1, // minute
		tms:  pastTick,
		tick: pastTick,
	}
	te.run()
	// Allow async TriggerTimer call to fail without database
	time.Sleep(20 * time.Millisecond)
	te.tasklk.RLock()
	task, exists := te.tasks["recurring-minute"]
	te.tasklk.RUnlock()
	require.True(t, exists, "recurring timer should not be deleted")
	// tick should have been advanced to the future
	assert.True(t, task.tick.After(pastTick), "tick should have been advanced")
}
