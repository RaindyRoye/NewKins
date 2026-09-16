package engine

import (
	"container/list"
	"sync"
	"testing"

	"github.com/gokins/core/runtime"
	"github.com/gokins/runner/runners"
)

// --- BuildTask.check() validation paths ---

func TestBuildTaskCheck_StageBuildIDMismatch(t *testing.T) {
	task := &BuildTask{
		build: &runtime.Build{
			Id: "build-1",
			Repo: &runtime.Repository{
				CloneURL: "",
			},
			Stages: []*runtime.Stage{
				{
					Id:      "stage-1",
					BuildId: "wrong-build-id", // mismatch
					Name:    "build",
					Steps: []*runtime.Step{
						{Id: "step-1", BuildId: "build-1", StageId: "stage-1", Name: "test", Step: "shell@ssh"},
					},
				},
			},
		},
		stages: make(map[string]*taskStage),
		jobs:   make(map[string]*jobSync),
	}
	result := task.check()
	if result {
		t.Fatal("expected check() to return false for stage build ID mismatch")
	}
}

func TestBuildTaskCheck_StageNameEmpty(t *testing.T) {
	task := &BuildTask{
		build: &runtime.Build{
			Id: "build-2",
			Repo: &runtime.Repository{
				CloneURL: "",
			},
			Stages: []*runtime.Stage{
				{
					Id:      "stage-2",
					BuildId: "build-2",
					Name:    "", // empty name
					Steps: []*runtime.Step{
						{Id: "step-2", BuildId: "build-2", StageId: "stage-2", Name: "test", Step: "shell@ssh"},
					},
				},
			},
		},
		stages: make(map[string]*taskStage),
		jobs:   make(map[string]*jobSync),
	}
	result := task.check()
	if result {
		t.Fatal("expected check() to return false for empty stage name")
	}
}

func TestBuildTaskCheck_StageNoSteps(t *testing.T) {
	task := &BuildTask{
		build: &runtime.Build{
			Id: "build-3",
			Repo: &runtime.Repository{
				CloneURL: "",
			},
			Stages: []*runtime.Stage{
				{
					Id:      "stage-3",
					BuildId: "build-3",
					Name:    "build",
					Steps:   []*runtime.Step{}, // empty steps
				},
			},
		},
		stages: make(map[string]*taskStage),
		jobs:   make(map[string]*jobSync),
	}
	result := task.check()
	if result {
		t.Fatal("expected check() to return false for empty steps")
	}
}

func TestBuildTaskCheck_DuplicateStageNames(t *testing.T) {
	task := &BuildTask{
		build: &runtime.Build{
			Id: "build-4",
			Repo: &runtime.Repository{
				CloneURL: "",
			},
			Stages: []*runtime.Stage{
				{
					Id:      "stage-4a",
					BuildId: "build-4",
					Name:    "build",
					Steps: []*runtime.Step{
						{Id: "step-4a", BuildId: "build-4", StageId: "stage-4a", Name: "test", Step: "shell@ssh"},
					},
				},
				{
					Id:      "stage-4b",
					BuildId: "build-4",
					Name:    "build", // duplicate name
					Steps: []*runtime.Step{
						{Id: "step-4b", BuildId: "build-4", StageId: "stage-4b", Name: "test2", Step: "shell@ssh"},
					},
				},
			},
		},
		stages: make(map[string]*taskStage),
		jobs:   make(map[string]*jobSync),
	}
	result := task.check()
	if result {
		t.Fatal("expected check() to return false for duplicate stage names")
	}
}

func TestBuildTaskCheck_StepBuildIDMismatch(t *testing.T) {
	task := &BuildTask{
		build: &runtime.Build{
			Id: "build-5",
			Repo: &runtime.Repository{
				CloneURL: "",
			},
			Stages: []*runtime.Stage{
				{
					Id:      "stage-5",
					BuildId: "build-5",
					Name:    "build",
					Steps: []*runtime.Step{
						{Id: "step-5", BuildId: "wrong-build-id", StageId: "stage-5", Name: "test", Step: "shell@ssh"},
					},
				},
			},
		},
		stages: make(map[string]*taskStage),
		jobs:   make(map[string]*jobSync),
	}
	result := task.check()
	if result {
		t.Fatal("expected check() to return false for step build ID mismatch")
	}
}

func TestBuildTaskCheck_StepStageIDMismatch(t *testing.T) {
	task := &BuildTask{
		build: &runtime.Build{
			Id: "build-6",
			Repo: &runtime.Repository{
				CloneURL: "",
			},
			Stages: []*runtime.Stage{
				{
					Id:      "stage-6",
					BuildId: "build-6",
					Name:    "build",
					Steps: []*runtime.Step{
						{Id: "step-6", BuildId: "build-6", StageId: "wrong-stage-id", Name: "test", Step: "shell@ssh"},
					},
				},
			},
		},
		stages: make(map[string]*taskStage),
		jobs:   make(map[string]*jobSync),
	}
	result := task.check()
	if result {
		t.Fatal("expected check() to return false for step stage ID mismatch")
	}
}

func TestBuildTaskCheck_StepPluginEmpty(t *testing.T) {
	task := &BuildTask{
		build: &runtime.Build{
			Id: "build-7",
			Repo: &runtime.Repository{
				CloneURL: "",
			},
			Stages: []*runtime.Stage{
				{
					Id:      "stage-7",
					BuildId: "build-7",
					Name:    "build",
					Steps: []*runtime.Step{
						{Id: "step-7", BuildId: "build-7", StageId: "stage-7", Name: "test", Step: ""}, // empty plugin
					},
				},
			},
		},
		stages: make(map[string]*taskStage),
		jobs:   make(map[string]*jobSync),
	}
	result := task.check()
	if result {
		t.Fatal("expected check() to return false for empty step plugin")
	}
}

func TestBuildTaskCheck_StepNameEmpty(t *testing.T) {
	task := &BuildTask{
		build: &runtime.Build{
			Id: "build-8",
			Repo: &runtime.Repository{
				CloneURL: "",
			},
			Stages: []*runtime.Stage{
				{
					Id:      "stage-8",
					BuildId: "build-8",
					Name:    "build",
					Steps: []*runtime.Step{
						{Id: "step-8", BuildId: "build-8", StageId: "stage-8", Name: "", Step: "shell@ssh"}, // empty name
					},
				},
			},
		},
		stages: make(map[string]*taskStage),
		jobs:   make(map[string]*jobSync),
	}
	result := task.check()
	if result {
		t.Fatal("expected check() to return false for empty step name")
	}
}

func TestBuildTaskCheck_DuplicateStepNames(t *testing.T) {
	task := &BuildTask{
		build: &runtime.Build{
			Id: "build-9",
			Repo: &runtime.Repository{
				CloneURL: "",
			},
			Stages: []*runtime.Stage{
				{
					Id:      "stage-9",
					BuildId: "build-9",
					Name:    "build",
					Steps: []*runtime.Step{
						{Id: "step-9a", BuildId: "build-9", StageId: "stage-9", Name: "test", Step: "shell@ssh"},
						{Id: "step-9b", BuildId: "build-9", StageId: "stage-9", Name: "test", Step: "shell@ssh"}, // duplicate
					},
				},
			},
		},
		stages: make(map[string]*taskStage),
		jobs:   make(map[string]*jobSync),
	}
	result := task.check()
	if result {
		t.Fatal("expected check() to return false for duplicate step names")
	}
}

// --- Manager accessor tests ---

func TestManagerBuildEgn(t *testing.T) {
	e := &BuildEngine{
		taskw: list.New(),
		tasks: make(map[string]*BuildTask),
	}
	m := &Manager{buildEgn: e}
	if m.BuildEgn() != e {
		t.Error("BuildEgn() should return the build engine")
	}
}

func TestManagerHRun(t *testing.T) {
	hr := &HbtpRunner{}
	m := &Manager{hrun: hr}
	if m.HRun() != hr {
		t.Error("HRun() should return the hbtp runner")
	}
}

func TestManagerTimerEng(t *testing.T) {
	te := &TimerEngine{
		tasks: make(map[string]*timerExec),
	}
	m := &Manager{timerEgn: te}
	if m.TimerEng() != te {
		t.Error("TimerEng() should return the timer engine")
	}
}

func TestManagerPlugins_NilJobEngine(t *testing.T) {
	m := &Manager{jobEgn: nil}
	plugs := m.Plugins()
	if plugs != nil {
		t.Errorf("Plugins() should return nil when jobEgn is nil, got %v", plugs)
	}
}

func TestManagerPlugins_WithJobEngine(t *testing.T) {
	je := &JobEngine{
		execs: make(map[string]*executer),
		jobs:  make(map[string]*jobSync),
	}
	je.execs["shell@ssh"] = &executer{plug: "shell@ssh", jobwt: list.New()}
	je.execs["gokins@git"] = &executer{plug: "gokins@git", jobwt: list.New()}

	m := &Manager{jobEgn: je}
	plugs := m.Plugins()
	if len(plugs) != 2 {
		t.Fatalf("expected 2 plugins, got %d", len(plugs))
	}
	found := map[string]bool{}
	for _, p := range plugs {
		found[p] = true
	}
	if !found["shell@ssh"] || !found["gokins@git"] {
		t.Errorf("expected shell@ssh and gokins@git, got %v", plugs)
	}
}

// --- BuildTask concurrent access tests ---

func TestBuildTaskConcurrentStatusUpdates(t *testing.T) {
	bt := &BuildTask{
		build: &runtime.Build{Id: "concurrent-build"},
	}

	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			bt.status("running", "test error")
			bt.status("ok", "")
		}()
	}
	wg.Wait()
}

func TestBuildTaskConcurrentGetJob(t *testing.T) {
	job := &jobSync{
		step: &runtime.Step{Id: "step-1", Name: "test"},
	}
	bt := &BuildTask{
		build: &runtime.Build{},
		jobs: map[string]*jobSync{
			"step-1": job,
		},
	}

	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			bt.GetJob("step-1")
			bt.GetJob("nonexistent")
		}()
	}
	wg.Wait()
}

// --- taskStage concurrent access tests ---

func TestTaskStageConcurrentStatusUpdates(t *testing.T) {
	ts := &taskStage{
		stage: &runtime.Stage{Name: "build"},
	}

	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			ts.status("running", "test")
			ts.status("ok", "")
		}()
	}
	wg.Wait()
}

// --- jobSync concurrent access tests ---

func TestJobSyncConcurrentStatusUpdates(t *testing.T) {
	js := &jobSync{
		step: &runtime.Step{Name: "compile"},
	}

	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			js.status("running", "test")
			js.status("ok", "")
		}()
	}
	wg.Wait()
}

// --- cmdSync tests ---

func TestCmdSyncInitialState(t *testing.T) {
	cmd := &cmdSync{
		cmd:    &runners.CmdContent{Id: "cmd-1", Conts: "echo test"},
		status: "pending",
	}
	if cmd.status != "pending" {
		t.Errorf("expected status 'pending', got %q", cmd.status)
	}
	if cmd.cmd.Id != "cmd-1" {
		t.Errorf("expected cmd ID 'cmd-1', got %q", cmd.cmd.Id)
	}
}

// --- BuildEngine concurrent Put and Stop ---

func TestBuildEngineConcurrentPutAndStop(t *testing.T) {
	e := &BuildEngine{
		taskw: list.New(),
		tasks: make(map[string]*BuildTask),
	}

	var wg sync.WaitGroup
	// Concurrent puts
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			e.Put(&runtime.Build{Id: "build-" + string(rune(n))})
		}(i)
	}
	// Concurrent stops (should not panic on empty tasks)
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			e.Stop()
		}()
	}
	wg.Wait()
}
