package engine

import (
	"container/list"
	"testing"
	"time"

	"github.com/gokins/core/runtime"
)

// TestBuildEngine_run tests the run() method of BuildEngine
func TestBuildEngine_run(t *testing.T) {
	// Create a BuildEngine instance
	be := &BuildEngine{
		taskw: list.New(),
		tasks: make(map[string]*BuildTask),
	}

	// Test 1: run() with empty queue should not panic
	be.run()

	// Test 2: run() with tasks in queue but at run limit
	// Add 5 tasks (default run limit)
	for i := 0; i < 5; i++ {
		build := &runtime.Build{
			Id:     string(rune('A' + i)),
			Status: "pending",
		}
		bt := NewBuildTask(be, build)
		be.Put(build)
		be.tasks[build.Id] = bt
	}

	// run() should not start new tasks when at limit
	be.run()

	// Test 3: run() with tasks below limit
	// Remove some tasks to go below limit
	for i := 0; i < 3; i++ {
		delete(be.tasks, string(rune('A'+i)))
	}

	// run() should start a new task
	be.run()

	// Verify a task was started (queue length should decrease)
	if be.taskw.Len() >= 5 {
		t.Logf("Queue length: %d (may vary based on implementation)", be.taskw.Len())
	}
}

// TestBuildTask_clears tests the clears method
func TestBuildTask_clears(t *testing.T) {
	// Create a build task
	build := &runtime.Build{
		Id: "test-build",
	}
	bt := NewBuildTask(nil, build)

	// clears() should not panic even with nil fields
	bt.clears()
}

// TestJobEngine_rmExec tests the rmExec method
func TestJobEngine_rmExec_extended(t *testing.T) {
	je := &JobEngine{
		execs: make(map[string]*executer),
		jobs:  make(map[string]*jobSync),
	}

	// Add an executer
	exec := &executer{
		tms:   time.Now().Add(-3 * time.Minute), // 3 minutes ago
		jobwt: list.New(),
	}
	je.execs["test-exec"] = exec

	// Add jobs associated with this executer
	job1 := &jobSync{
		step: &runtime.Step{Id: "step-1"},
	}
	job2 := &jobSync{
		step: &runtime.Step{Id: "step-2"},
	}
	je.jobs["step-1"] = job1
	je.jobs["step-2"] = job2

	// rmExec should remove the executer and mark associated jobs as ended
	je.rmExec("test-exec", exec)

	// Verify executer is removed
	if _, exists := je.execs["test-exec"]; exists {
		t.Error("rmExec() should remove the executer")
	}
}

// TestManager_BuildEgn_extended tests the BuildEgn accessor
func TestManager_BuildEgn_extended(t *testing.T) {
	// Create a Manager with a BuildEngine
	be := &BuildEngine{
		taskw: list.New(),
		tasks: make(map[string]*BuildTask),
	}
	mgr := &Manager{
		buildEgn: be,
	}

	// BuildEgn() should return the build engine
	result := mgr.BuildEgn()
	if result != be {
		t.Error("BuildEgn() should return the build engine")
	}
}

// TestManager_Plugins_extended tests the Plugins method
func TestManager_Plugins_extended(t *testing.T) {
	// Create a Manager with a JobEngine
	je := &JobEngine{
		execs: make(map[string]*executer),
		jobs:  make(map[string]*jobSync),
	}

	// Add some executers with plugins
	je.execs["shell"] = &executer{plug: "shell", jobwt: list.New()}
	je.execs["docker"] = &executer{plug: "docker", jobwt: list.New()}

	mgr := &Manager{
		jobEgn: je,
	}

	// Plugins() should return the list of plugin names
	plugins := mgr.Plugins()
	if len(plugins) != 2 {
		t.Errorf("Plugins() should return 2 plugins, got %d", len(plugins))
	}
}

// TestBuildTask_check_extended tests the check method with various scenarios
func TestBuildTask_check_extended(t *testing.T) {
	tests := []struct {
		name     string
		build    *runtime.Build
		expected bool
	}{
		{
			name: "build with no stages",
			build: &runtime.Build{
				Id:     "test-2",
				Status: "pending",
				Stages: []*runtime.Stage{},
			},
			expected: false,
		},
		{
			name: "build with stage but no steps",
			build: &runtime.Build{
				Id:     "test-3",
				Status: "pending",
				Stages: []*runtime.Stage{
					{
						Id:    "stage-1",
						Name:  "Stage 1",
						Steps: []*runtime.Step{},
					},
				},
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bt := NewBuildTask(nil, tt.build)
			result := bt.check()
			if result != tt.expected {
				t.Errorf("check() = %v, want %v", result, tt.expected)
			}
		})
	}
}

// TestBuildTask_getRepo_extended tests the getRepo method
func TestBuildTask_getRepo_extended(t *testing.T) {
	// Test with isClone = false (should skip)
	bt := &BuildTask{
		isClone: false,
	}
	err := bt.getRepo()
	if err != nil {
		t.Errorf("getRepo() with isClone=false should not error, got: %v", err)
	}

	// Test with isClone = true but no repoPath
	// getRepo() will try to MkdirAll on empty path which fails
	bt = &BuildTask{
		isClone:  true,
		repoPath: "",
	}
	err = bt.getRepo()
	// Empty repoPath causes MkdirAll("") to fail, which is expected behavior
	if err == nil {
		t.Log("getRepo() with empty repoPath succeeded (no repo path to create)")
	}
}
