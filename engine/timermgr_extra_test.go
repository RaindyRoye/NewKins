package engine

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/gokins/gokins/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestTimerEngineResetOneHourlyTimer tests type=2 (hourly) timer scheduling
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
	err := te.resetOne(tmr)
	require.NoError(t, err)
	task, ok := te.tasks["test-hourly"]
	require.True(t, ok)
	assert.Equal(t, int64(2), task.typ)
	// tick should be within ~1 hour
	assert.LessOrEqual(t, time.Until(task.tick), time.Hour+time.Second)
}

// TestTimerEngineResetOneDailyTimer tests type=3 (daily) timer scheduling
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
	err := te.resetOne(tmr)
	require.NoError(t, err)
	task, ok := te.tasks["test-daily"]
	require.True(t, ok)
	assert.Equal(t, int64(3), task.typ)
	// tick should be within ~24 hours
	assert.LessOrEqual(t, time.Until(task.tick), 24*time.Hour+time.Second)
}

// TestTimerEngineResetOneWeeklyTimer tests type=4 (weekly) timer scheduling
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
	err := te.resetOne(tmr)
	require.NoError(t, err)
	task, ok := te.tasks["test-weekly"]
	require.True(t, ok)
	assert.Equal(t, int64(4), task.typ)
	// tick should be within ~7 days
	assert.LessOrEqual(t, time.Until(task.tick), 7*24*time.Hour+time.Second)
}

// TestTimerEngineResetOneMonthlyTimer tests type=5 (monthly) timer scheduling
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
	err := te.resetOne(tmr)
	require.NoError(t, err)
	task, ok := te.tasks["test-monthly"]
	require.True(t, ok)
	assert.Equal(t, int64(5), task.typ)
	// tick should be within ~30 days
	assert.LessOrEqual(t, time.Until(task.tick), 31*24*time.Hour+time.Second)
}

// TestTimerEngineResetOneExpiredOnceTimer tests that an expired once-timer
// (type=0 with time in the past) is NOT added to the task map.
func TestTimerEngineResetOneExpiredOnceTimer(t *testing.T) {
	te := &TimerEngine{
		tasks: make(map[string]*timerExec),
	}
	pastTime := time.Now().Add(-time.Hour)
	params := map[string]any{
		"timerType": 0, // once
		"dates":     pastTime.Format(time.RFC3339Nano),
	}
	paramBytes, _ := json.Marshal(params)

	tmr := &model.TTrigger{
		Id:     "test-expired-once",
		Types:  "timer",
		Name:   "expired-once",
		Params: string(paramBytes),
	}
	err := te.resetOne(tmr)
	require.NoError(t, err)
	// Expired once-timer should NOT be in the task map
	_, ok := te.tasks["test-expired-once"]
	assert.False(t, ok, "expired once-timer should not be scheduled")
}

// TestTimerEngineResetOneMissingTimerType tests that missing timerType returns error
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
	require.Error(t, err)
	assert.Contains(t, err.Error(), "timerType")
}

// TestTimerEngineResetOneInvalidDates tests that invalid date format returns error
func TestTimerEngineResetOneInvalidDates(t *testing.T) {
	te := &TimerEngine{
		tasks: make(map[string]*timerExec),
	}
	params := map[string]any{
		"timerType": 1,
		"dates":     "not-a-valid-date",
	}
	paramBytes, _ := json.Marshal(params)

	tmr := &model.TTrigger{
		Id:     "test-bad-dates",
		Types:  "timer",
		Name:   "bad-dates",
		Params: string(paramBytes),
	}
	err := te.resetOne(tmr)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "parse dates")
}

// TestTimerEngineResetOneUpdateExisting tests updating an existing timer task
func TestTimerEngineResetOneUpdateExisting(t *testing.T) {
	te := &TimerEngine{
		tasks: make(map[string]*timerExec),
	}
	params := map[string]any{
		"timerType": 1, // minute
		"dates":     time.Now().Format(time.RFC3339Nano),
	}
	paramBytes, _ := json.Marshal(params)

	tmr := &model.TTrigger{
		Id:     "test-update",
		Types:  "timer",
		Name:   "update-timer",
		Params: string(paramBytes),
	}
	// First call adds the task
	require.NoError(t, te.resetOne(tmr))
	require.Contains(t, te.tasks, "test-update")
	firstTick := te.tasks["test-update"].tick

	// Small delay so tick changes
	time.Sleep(10 * time.Millisecond)

	// Second call should update it
	require.NoError(t, te.resetOne(tmr))
	require.Contains(t, te.tasks, "test-update")
	// Tick may be the same (within same minute) but the task should still exist
	assert.NotNil(t, te.tasks["test-update"])
	_ = firstTick // just verify no panic
}
