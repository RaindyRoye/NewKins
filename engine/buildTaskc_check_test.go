package engine

import (
	"testing"

	"github.com/gokins/core/common"
	"github.com/gokins/core/runtime"
)

func TestBuildTaskCheck_ValidBuild(t *testing.T) {
	task := &BuildTask{
		build: &runtime.Build{
			Id: "test-build",
			Repo: &runtime.Repository{
				CloneURL: "",
			},
			Stages: []*runtime.Stage{
				{
					Id:      "stage1",
					BuildId: "test-build",
					Name:    "build",
					Steps: []*runtime.Step{
						{
							Id:      "step1",
							BuildId: "test-build",
							StageId: "stage1",
							Step:    "shell@ssh",
							Name:    "compile",
						},
					},
				},
			},
		},
		stages: make(map[string]*taskStage),
		jobs:   make(map[string]*jobSync),
	}

	result := task.check()
	if !result {
		t.Fatalf("expected check() to return true for valid build, got false. Error: %s", task.build.Error)
	}

	// Verify stages were populated
	if len(task.stages) != 1 {
		t.Errorf("expected 1 stage, got %d", len(task.stages))
	}

	// Verify jobs were populated
	if len(task.jobs) != 1 {
		t.Errorf("expected 1 job, got %d", len(task.jobs))
	}
}

func TestBuildTaskCheck_StageBuildIdMismatch(t *testing.T) {
	task := &BuildTask{
		build: &runtime.Build{
			Id: "test-build",
			Repo: &runtime.Repository{
				CloneURL: "",
			},
			Stages: []*runtime.Stage{
				{
					Id:      "stage1",
					BuildId: "wrong-build",
					Name:    "build",
					Steps: []*runtime.Step{
						{
							Id:      "step1",
							BuildId: "test-build",
							StageId: "stage1",
							Step:    "shell@ssh",
							Name:    "compile",
						},
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

	if task.build.Event != common.BuildEventCheckParam {
		t.Errorf("expected event %q, got %q", common.BuildEventCheckParam, task.build.Event)
	}

	expected := "Stage Build id err:wrong-build/test-build"
	if task.build.Error != expected {
		t.Errorf("expected error %q, got %q", expected, task.build.Error)
	}
}

func TestBuildTaskCheck_EmptyStageName(t *testing.T) {
	task := &BuildTask{
		build: &runtime.Build{
			Id: "test-build",
			Repo: &runtime.Repository{
				CloneURL: "",
			},
			Stages: []*runtime.Stage{
				{
					Id:      "stage1",
					BuildId: "test-build",
					Name:    "",
					Steps: []*runtime.Step{
						{
							Id:      "step1",
							BuildId: "test-build",
							StageId: "stage1",
							Step:    "shell@ssh",
							Name:    "compile",
						},
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

	if task.build.Event != common.BuildEventCheckParam {
		t.Errorf("expected event %q, got %q", common.BuildEventCheckParam, task.build.Event)
	}

	expected := "build Stage name is empty"
	if task.build.Error != expected {
		t.Errorf("expected error %q, got %q", expected, task.build.Error)
	}
}

func TestBuildTaskCheck_EmptySteps(t *testing.T) {
	task := &BuildTask{
		build: &runtime.Build{
			Id: "test-build",
			Repo: &runtime.Repository{
				CloneURL: "",
			},
			Stages: []*runtime.Stage{
				{
					Id:      "stage1",
					BuildId: "test-build",
					Name:    "build",
					Steps:   []*runtime.Step{},
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

	if task.build.Event != common.BuildEventCheckParam {
		t.Errorf("expected event %q, got %q", common.BuildEventCheckParam, task.build.Event)
	}

	expected := "build Stages is empty"
	if task.build.Error != expected {
		t.Errorf("expected error %q, got %q", expected, task.build.Error)
	}
}

func TestBuildTaskCheck_DuplicateStageName(t *testing.T) {
	task := &BuildTask{
		build: &runtime.Build{
			Id: "test-build",
			Repo: &runtime.Repository{
				CloneURL: "",
			},
			Stages: []*runtime.Stage{
				{
					Id:      "stage1",
					BuildId: "test-build",
					Name:    "build",
					Steps: []*runtime.Step{
						{
							Id:      "step1",
							BuildId: "test-build",
							StageId: "stage1",
							Step:    "shell@ssh",
							Name:    "compile",
						},
					},
				},
				{
					Id:      "stage2",
					BuildId: "test-build",
					Name:    "build",
					Steps: []*runtime.Step{
						{
							Id:      "step2",
							BuildId: "test-build",
							StageId: "stage2",
							Step:    "shell@ssh",
							Name:    "test",
						},
					},
				},
			},
		},
		stages: make(map[string]*taskStage),
		jobs:   make(map[string]*jobSync),
	}

	result := task.check()
	if result {
		t.Fatal("expected check() to return false for duplicate stage name")
	}

	if task.build.Event != common.BuildEventCheckParam {
		t.Errorf("expected event %q, got %q", common.BuildEventCheckParam, task.build.Event)
	}

	expected := "build Stages.build is repeat"
	if task.build.Error != expected {
		t.Errorf("expected error %q, got %q", expected, task.build.Error)
	}
}

func TestBuildTaskCheck_StepBuildIdMismatch(t *testing.T) {
	task := &BuildTask{
		build: &runtime.Build{
			Id: "test-build",
			Repo: &runtime.Repository{
				CloneURL: "",
			},
			Stages: []*runtime.Stage{
				{
					Id:      "stage1",
					BuildId: "test-build",
					Name:    "build",
					Steps: []*runtime.Step{
						{
							Id:      "step1",
							BuildId: "wrong-build",
							StageId: "stage1",
							Step:    "shell@ssh",
							Name:    "compile",
						},
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

	if task.build.Event != common.BuildEventCheckParam {
		t.Errorf("expected event %q, got %q", common.BuildEventCheckParam, task.build.Event)
	}

	expected := "Job Build id err:test-build/test-build"
	if task.build.Error != expected {
		t.Errorf("expected error %q, got %q", expected, task.build.Error)
	}
}

func TestBuildTaskCheck_StepStageIdMismatch(t *testing.T) {
	task := &BuildTask{
		build: &runtime.Build{
			Id: "test-build",
			Repo: &runtime.Repository{
				CloneURL: "",
			},
			Stages: []*runtime.Stage{
				{
					Id:      "stage1",
					BuildId: "test-build",
					Name:    "build",
					Steps: []*runtime.Step{
						{
							Id:      "step1",
							BuildId: "test-build",
							StageId: "wrong-stage",
							Step:    "shell@ssh",
							Name:    "compile",
						},
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

	if task.build.Event != common.BuildEventCheckParam {
		t.Errorf("expected event %q, got %q", common.BuildEventCheckParam, task.build.Event)
	}

	expected := "Job Stage id err:test-build/test-build"
	if task.build.Error != expected {
		t.Errorf("expected error %q, got %q", expected, task.build.Error)
	}
}

func TestBuildTaskCheck_EmptyStepPlugin(t *testing.T) {
	task := &BuildTask{
		build: &runtime.Build{
			Id: "test-build",
			Repo: &runtime.Repository{
				CloneURL: "",
			},
			Stages: []*runtime.Stage{
				{
					Id:      "stage1",
					BuildId: "test-build",
					Name:    "build",
					Steps: []*runtime.Step{
						{
							Id:      "step1",
							BuildId: "test-build",
							StageId: "stage1",
							Step:    "",
							Name:    "compile",
						},
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

	if task.build.Event != common.BuildEventCheckParam {
		t.Errorf("expected event %q, got %q", common.BuildEventCheckParam, task.build.Event)
	}

	expected := "build Step Plugin is empty"
	if task.build.Error != expected {
		t.Errorf("expected error %q, got %q", expected, task.build.Error)
	}
}

func TestBuildTaskCheck_EmptyStepName(t *testing.T) {
	task := &BuildTask{
		build: &runtime.Build{
			Id: "test-build",
			Repo: &runtime.Repository{
				CloneURL: "",
			},
			Stages: []*runtime.Stage{
				{
					Id:      "stage1",
					BuildId: "test-build",
					Name:    "build",
					Steps: []*runtime.Step{
						{
							Id:      "step1",
							BuildId: "test-build",
							StageId: "stage1",
							Step:    "shell@ssh",
							Name:    "",
						},
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

	if task.build.Event != common.BuildEventCheckParam {
		t.Errorf("expected event %q, got %q", common.BuildEventCheckParam, task.build.Event)
	}

	expected := "build Step name is empty"
	if task.build.Error != expected {
		t.Errorf("expected error %q, got %q", expected, task.build.Error)
	}
}

func TestBuildTaskCheck_DuplicateStepName(t *testing.T) {
	task := &BuildTask{
		build: &runtime.Build{
			Id: "test-build",
			Repo: &runtime.Repository{
				CloneURL: "",
			},
			Stages: []*runtime.Stage{
				{
					Id:      "stage1",
					BuildId: "test-build",
					Name:    "build",
					Steps: []*runtime.Step{
						{
							Id:      "step1",
							BuildId: "test-build",
							StageId: "stage1",
							Step:    "shell@ssh",
							Name:    "compile",
						},
						{
							Id:      "step2",
							BuildId: "test-build",
							StageId: "stage1",
							Step:    "shell@ssh",
							Name:    "compile",
						},
					},
				},
			},
		},
		stages: make(map[string]*taskStage),
		jobs:   make(map[string]*jobSync),
	}

	result := task.check()
	if result {
		t.Fatal("expected check() to return false for duplicate step name")
	}

	if task.build.Event != common.BuildEventCheckParam {
		t.Errorf("expected event %q, got %q", common.BuildEventCheckParam, task.build.Event)
	}

	expected := "build Job.compile is repeat"
	if task.build.Error != expected {
		t.Errorf("expected error %q, got %q", expected, task.build.Error)
	}
}

func TestBuildTaskCheck_MultipleStages(t *testing.T) {
	task := &BuildTask{
		build: &runtime.Build{
			Id: "test-build",
			Repo: &runtime.Repository{
				CloneURL: "",
			},
			Stages: []*runtime.Stage{
				{
					Id:      "stage1",
					BuildId: "test-build",
					Name:    "build",
					Steps: []*runtime.Step{
						{
							Id:      "step1",
							BuildId: "test-build",
							StageId: "stage1",
							Step:    "shell@ssh",
							Name:    "compile",
						},
					},
				},
				{
					Id:      "stage2",
					BuildId: "test-build",
					Name:    "test",
					Steps: []*runtime.Step{
						{
							Id:      "step2",
							BuildId: "test-build",
							StageId: "stage2",
							Step:    "shell@ssh",
							Name:    "run-tests",
						},
					},
				},
			},
		},
		stages: make(map[string]*taskStage),
		jobs:   make(map[string]*jobSync),
	}

	result := task.check()
	if !result {
		t.Fatalf("expected check() to return true for valid multi-stage build, got false. Error: %s", task.build.Error)
	}

	if len(task.stages) != 2 {
		t.Errorf("expected 2 stages, got %d", len(task.stages))
	}

	if len(task.jobs) != 2 {
		t.Errorf("expected 2 jobs, got %d", len(task.jobs))
	}
}
