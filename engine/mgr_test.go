package engine

import (
	"testing"
)

// TestManagerAccessors verifies the Manager accessor methods return the
// correct internal fields and handle nil fields gracefully.
func TestManagerAccessors(t *testing.T) {
	be := &BuildEngine{}
	je := &JobEngine{}
	te := &TimerEngine{}
	hr := &HbtpRunner{}

	m := &Manager{
		buildEgn: be,
		jobEgn:   je,
		timerEgn: te,
		hrun:     hr,
	}

	if m.BuildEgn() != be {
		t.Error("BuildEgn() did not return the expected engine")
	}
	if m.HRun() != hr {
		t.Error("HRun() did not return the expected runner")
	}
	if m.TimerEng() != te {
		t.Error("TimerEng() did not return the expected engine")
	}

	// Plugins() with nil jobEgn
	mNil := &Manager{}
	if plugs := mNil.Plugins(); plugs != nil {
		t.Errorf("Plugins() with nil jobEgn should return nil, got %v", plugs)
	}

	// Plugins() with valid jobEgn - just verify it doesn't panic
	_ = m.Plugins()
}

// TestManagerPluginsDelegation verifies Plugins delegates to the JobEngine.
func TestManagerPluginsDelegation(t *testing.T) {
	je := &JobEngine{
		execs: make(map[string]*executer),
	}
	je.execs["p1"] = &executer{plug: "p1"}
	je.execs["p2"] = &executer{plug: "p2"}

	m := &Manager{jobEgn: je}
	plugs := m.Plugins()
	if len(plugs) != 2 {
		t.Fatalf("expected 2 plugins, got %d", len(plugs))
	}
	found := map[string]bool{}
	for _, p := range plugs {
		found[p] = true
	}
	if !found["p1"] || !found["p2"] {
		t.Errorf("expected p1 and p2, got %v", plugs)
	}
}
