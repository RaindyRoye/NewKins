package engine

import (
	"testing"
)

func TestTimerEngine_Delete_NonExistent(t *testing.T) {
	egn := &TimerEngine{
		tasks: make(map[string]*timerExec),
	}
	egn.Delete("nonexistent") // Should not panic
}

func TestTimerEngine_Delete_Existing(t *testing.T) {
	egn := &TimerEngine{
		tasks: make(map[string]*timerExec),
	}
	egn.tasks["timer1"] = &timerExec{}
	egn.Delete("timer1")
	
	if _, exists := egn.tasks["timer1"]; exists {
		t.Error("Delete() should remove the timer")
	}
}

func TestTimerEngine_Refresh_EmptyID(t *testing.T) {
	egn := &TimerEngine{
		tasks: make(map[string]*timerExec),
	}
	err := egn.Refresh("")
	if err == nil {
		t.Error("Refresh() with empty ID should return error")
	}
}

// TestTimerEngine_Refresh_RequiresDB is skipped because Refresh() requires
// a database connection. In a real environment with DB initialized, this would
// test the full refresh flow.
func TestTimerEngine_Refresh_RequiresDB(t *testing.T) {
	t.Skip("Refresh() requires database connection - tested in integration tests")
}
