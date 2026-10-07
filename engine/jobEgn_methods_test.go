package engine

import (
	"testing"

	"github.com/gokins/core/runtime"
)

func TestJobEngine_Put_NilJob(t *testing.T) {
	egn := &JobEngine{
		execs: make(map[string]*executer),
		jobs:  make(map[string]*jobSync),
	}
	err := egn.Put(nil)
	if err == nil {
		t.Error("Put(nil) should return error")
	}
}

func TestJobEngine_Put_EmptyStep(t *testing.T) {
	egn := &JobEngine{
		execs: make(map[string]*executer),
		jobs:  make(map[string]*jobSync),
	}
	job := &jobSync{
		step: &runtime.Step{Step: ""},
	}
	err := egn.Put(job)
	if err == nil {
		t.Error("Put() with empty step should return error")
	}
}

func TestJobEngine_Put_NewPlugin(t *testing.T) {
	egn := &JobEngine{
		execs: make(map[string]*executer),
		jobs:  make(map[string]*jobSync),
	}
	job := &jobSync{
		step: &runtime.Step{Step: "shell@ssh"},
	}
	err := egn.Put(job)
	if err == nil {
		t.Error("Put() with non-existent plugin should return error")
	}
}

func TestJobEngine_Pull_EmptyPlugs(t *testing.T) {
	egn := &JobEngine{
		execs: make(map[string]*executer),
		jobs:  make(map[string]*jobSync),
	}
	result := egn.Pull("runner1", []string{})
	if result != nil {
		t.Error("Pull() with empty plugs should return nil")
	}
}

func TestJobEngine_Pull_EmptyStrings(t *testing.T) {
	egn := &JobEngine{
		execs: make(map[string]*executer),
		jobs:  make(map[string]*jobSync),
	}
	result := egn.Pull("runner1", []string{"", ""})
	if result != nil {
		t.Error("Pull() with empty string plugs should return nil")
	}
}

func TestJobEngine_Pull_NoJobs(t *testing.T) {
	egn := &JobEngine{
		execs: make(map[string]*executer),
		jobs:  make(map[string]*jobSync),
	}
	result := egn.Pull("runner1", []string{"shell@ssh"})
	if result != nil {
		t.Error("Pull() with no queued jobs should return nil")
	}
}

func TestJobEngine_Plugins_Empty(t *testing.T) {
	egn := &JobEngine{
		execs: make(map[string]*executer),
		jobs:  make(map[string]*jobSync),
	}
	plugins := egn.Plugins()
	if len(plugins) != 0 {
		t.Errorf("Plugins() on empty engine should return empty slice, got %d", len(plugins))
	}
}

func TestJobEngine_Plugins_WithExecuters(t *testing.T) {
	egn := &JobEngine{
		execs: make(map[string]*executer),
		jobs:  make(map[string]*jobSync),
	}
	egn.execs["shell@ssh"] = &executer{plug: "shell@ssh"}
	egn.execs["gokins@git"] = &executer{plug: "gokins@git"}
	
	plugins := egn.Plugins()
	if len(plugins) != 2 {
		t.Errorf("Plugins() should return 2 plugins, got %d", len(plugins))
	}
}
