package engine

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestManager_BuildEgn(t *testing.T) {
	egn := &BuildEngine{}
	mgr := &Manager{
		buildEgn: egn,
	}

	result := mgr.BuildEgn()
	assert.Equal(t, egn, result, "BuildEgn() should return the build engine")
}

func TestManager_BuildEgn_Nil(t *testing.T) {
	mgr := &Manager{}
	result := mgr.BuildEgn()
	assert.Nil(t, result, "BuildEgn() should return nil when not initialized")
}

func TestManager_HRun(t *testing.T) {
	runner := &HbtpRunner{}
	mgr := &Manager{
		hrun: runner,
	}

	result := mgr.HRun()
	assert.Equal(t, runner, result, "HRun() should return the hbtp runner")
}

func TestManager_HRun_Nil(t *testing.T) {
	mgr := &Manager{}
	result := mgr.HRun()
	assert.Nil(t, result, "HRun() should return nil when not initialized")
}

func TestManager_TimerEng(t *testing.T) {
	timer := &TimerEngine{}
	mgr := &Manager{
		timerEgn: timer,
	}

	result := mgr.TimerEng()
	assert.Equal(t, timer, result, "TimerEng() should return the timer engine")
}

func TestManager_TimerEng_Nil(t *testing.T) {
	mgr := &Manager{}
	result := mgr.TimerEng()
	assert.Nil(t, result, "TimerEng() should return nil when not initialized")
}

func TestManager_Plugins_WithJobEngine(t *testing.T) {
	je := &JobEngine{
		execs: make(map[string]*executer),
		jobs:  make(map[string]*jobSync),
	}
	mgr := &Manager{
		jobEgn: je,
	}

	result := mgr.Plugins()
	assert.NotNil(t, result, "Plugins() should return a slice")
	assert.Equal(t, 0, len(result), "Plugins() should return empty slice when no plugins registered")
}

func TestManager_Plugins_WithPlugins(t *testing.T) {
	je := &JobEngine{
		execs: map[string]*executer{
			"plugin1": {plug: "plugin1"},
			"plugin2": {plug: "plugin2"},
		},
		jobs: make(map[string]*jobSync),
	}
	mgr := &Manager{
		jobEgn: je,
	}

	result := mgr.Plugins()
	assert.Equal(t, 2, len(result), "Plugins() should return 2 plugins")

	// Check that both plugins are present (order not guaranteed)
	found := make(map[string]bool)
	for _, p := range result {
		found[p] = true
	}
	assert.True(t, found["plugin1"], "Plugins() should contain plugin1")
	assert.True(t, found["plugin2"], "Plugins() should contain plugin2")
}

func TestManager_Plugins_NilJobEngine(t *testing.T) {
	mgr := &Manager{}
	result := mgr.Plugins()
	assert.Nil(t, result, "Plugins() should return nil when jobEgn is nil")
}
