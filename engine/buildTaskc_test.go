package engine

import (
	"sync"
	"testing"

	"github.com/gokins/core/common"
	"github.com/gokins/core/runtime"
)

func TestBuildTask_check_EmptyStages(t *testing.T) {
	bt := &BuildTask{
		build: &runtime.Build{
			Id:     "build-1",
			Repo:   &runtime.Repository{CloneURL: "/tmp/test"},
			Stages: []*runtime.Stage{},
		},
		stages: make(map[string]*taskStage),
	}

	if bt.check() {
		t.Error("check() should return false when stages are empty")
	}
	if bt.build.Event != common.BuildEventCheckParam {
		t.Errorf("expected event %q, got %q", common.BuildEventCheckParam, bt.build.Event)
	}
	if bt.build.Error != "stages is empty" {
		t.Errorf("expected error 'stages is empty', got %q", bt.build.Error)
	}
}

func TestBuildTask_check_StageBuildIdMismatch(t *testing.T) {
	bt := &BuildTask{
		build: &runtime.Build{
			Id:   "build-1",
			Repo: &runtime.Repository{CloneURL: "/tmp/test"},
			Stages: []*runtime.Stage{
				{
					Id:      "stage-1",
					BuildId: "build-2", // Mismatch
					Name:    "stage1",
					Steps: []*runtime.Step{
						{Id: "step-1", Name: "step1", BuildId: "build-2", StageId: "stage-1"},
					},
				},
			},
		},
		stages: make(map[string]*taskStage),
	}

	if bt.check() {
		t.Error("check() should return false when stage BuildId mismatches")
	}
	expected := "stage BuildId mismatch: stage.BuildId=build-2, build.Id=build-1"
	if bt.build.Error != expected {
		t.Errorf("expected error %q, got %q", expected, bt.build.Error)
	}
}

func TestBuildTask_check_StageNameEmpty(t *testing.T) {
	bt := &BuildTask{
		build: &runtime.Build{
			Id:   "build-1",
			Repo: &runtime.Repository{CloneURL: "/tmp/test"},
			Stages: []*runtime.Stage{
				{
					Id:      "stage-1",
					BuildId: "build-1",
					Name:    "", // Empty
					Steps: []*runtime.Step{
						{Id: "step-1", Name: "step1", BuildId: "build-1", StageId: "stage-1"},
					},
				},
			},
		},
		stages: make(map[string]*taskStage),
	}

	if bt.check() {
		t.Error("check() should return false when stage name is empty")
	}
	if bt.build.Error != "stage name is empty" {
		t.Errorf("expected error 'stage name is empty', got %q", bt.build.Error)
	}
}

func TestBuildTask_check_StageStepsEmpty(t *testing.T) {
	bt := &BuildTask{
		build: &runtime.Build{
			Id:   "build-1",
			Repo: &runtime.Repository{CloneURL: "/tmp/test"},
			Stages: []*runtime.Stage{
				{
					Id:      "stage-1",
					BuildId: "build-1",
					Name:    "stage1",
					Steps:   []*runtime.Step{}, // Empty
				},
			},
		},
		stages: make(map[string]*taskStage),
	}

	if bt.check() {
		t.Error("check() should return false when stage steps are empty")
	}
	if bt.build.Error != "stage steps is empty" {
		t.Errorf("expected error 'stage steps is empty', got %q", bt.build.Error)
	}
}

func TestBuildTask_check_DuplicateStageName(t *testing.T) {
	bt := &BuildTask{
		build: &runtime.Build{
			Id:   "build-1",
			Repo: &runtime.Repository{CloneURL: "/tmp/test"},
			Stages: []*runtime.Stage{
				{
					Id:      "stage-1",
					BuildId: "build-1",
					Name:    "duplicate",
					Steps: []*runtime.Step{
						{Id: "step-1", Name: "step1", BuildId: "build-1", StageId: "stage-1", Step: "plugin1"},
					},
				},
				{
					Id:      "stage-2",
					BuildId: "build-1",
					Name:    "duplicate", // Duplicate
					Steps: []*runtime.Step{
						{Id: "step-2", Name: "step2", BuildId: "build-1", StageId: "stage-2", Step: "plugin2"},
					},
				},
			},
		},
		stages: make(map[string]*taskStage),
		joblk:  sync.RWMutex{},
		jobs:   make(map[string]*jobSync),
	}

	if bt.check() {
		t.Error("check() should return false when stage names are duplicated")
	}
	expected := `stage "duplicate" is duplicated`
	if bt.build.Error != expected {
		t.Errorf("expected error %q, got %q", expected, bt.build.Error)
	}
}

func TestBuildTask_check_StepBuildIdMismatch(t *testing.T) {
	bt := &BuildTask{
		build: &runtime.Build{
			Id:   "build-1",
			Repo: &runtime.Repository{CloneURL: "/tmp/test"},
			Stages: []*runtime.Stage{
				{
					Id:      "stage-1",
					BuildId: "build-1",
					Name:    "stage1",
					Steps: []*runtime.Step{
						{
							Id:      "step-1",
							Name:    "step1",
							BuildId: "build-2", // Mismatch
							StageId: "stage-1",
						},
					},
				},
			},
		},
		stages: make(map[string]*taskStage),
		joblk:  sync.RWMutex{},
		jobs:   make(map[string]*jobSync),
	}

	if bt.check() {
		t.Error("check() should return false when step BuildId mismatches")
	}
	expected := "step BuildId mismatch: step.BuildId=build-2, build.Id=build-1"
	if bt.build.Error != expected {
		t.Errorf("expected error %q, got %q", expected, bt.build.Error)
	}
}

func TestBuildTask_check_StepStageIdMismatch(t *testing.T) {
	bt := &BuildTask{
		build: &runtime.Build{
			Id:   "build-1",
			Repo: &runtime.Repository{CloneURL: "/tmp/test"},
			Stages: []*runtime.Stage{
				{
					Id:      "stage-1",
					BuildId: "build-1",
					Name:    "stage1",
					Steps: []*runtime.Step{
						{
							Id:      "step-1",
							Name:    "step1",
							BuildId: "build-1",
							StageId: "stage-2", // Mismatch
						},
					},
				},
			},
		},
		stages: make(map[string]*taskStage),
		joblk:  sync.RWMutex{},
		jobs:   make(map[string]*jobSync),
	}

	if bt.check() {
		t.Error("check() should return false when step StageId mismatches")
	}
	expected := "step StageId mismatch: step.StageId=stage-2, stage.Id=stage-1"
	if bt.build.Error != expected {
		t.Errorf("expected error %q, got %q", expected, bt.build.Error)
	}
}

func TestBuildTask_check_StepPluginEmpty(t *testing.T) {
	bt := &BuildTask{
		build: &runtime.Build{
			Id:   "build-1",
			Repo: &runtime.Repository{CloneURL: "/tmp/test"},
			Stages: []*runtime.Stage{
				{
					Id:      "stage-1",
					BuildId: "build-1",
					Name:    "stage1",
					Steps: []*runtime.Step{
						{
							Id:      "step-1",
							Name:    "step1",
							BuildId: "build-1",
							StageId: "stage-1",
							Step:    "", // Empty
						},
					},
				},
			},
		},
		stages: make(map[string]*taskStage),
	}

	if bt.check() {
		t.Error("check() should return false when step plugin is empty")
	}
	if bt.build.Error != "step plugin is empty" {
		t.Errorf("expected error 'step plugin is empty', got %q", bt.build.Error)
	}
}

func TestBuildTask_check_StepNameEmpty(t *testing.T) {
	bt := &BuildTask{
		build: &runtime.Build{
			Id:   "build-1",
			Repo: &runtime.Repository{CloneURL: "/tmp/test"},
			Stages: []*runtime.Stage{
				{
					Id:      "stage-1",
					BuildId: "build-1",
					Name:    "stage1",
					Steps: []*runtime.Step{
						{
							Id:      "step-1",
							Name:    "", // Empty
							BuildId: "build-1",
							StageId: "stage-1",
							Step:    "plugin1",
						},
					},
				},
			},
		},
		stages: make(map[string]*taskStage),
	}

	if bt.check() {
		t.Error("check() should return false when step name is empty")
	}
	if bt.build.Error != "step name is empty" {
		t.Errorf("expected error 'step name is empty', got %q", bt.build.Error)
	}
}

func TestBuildTask_check_DuplicateStepName(t *testing.T) {
	bt := &BuildTask{
		build: &runtime.Build{
			Id:   "build-1",
			Repo: &runtime.Repository{CloneURL: "/tmp/test"},
			Stages: []*runtime.Stage{
				{
					Id:      "stage-1",
					BuildId: "build-1",
					Name:    "stage1",
					Steps: []*runtime.Step{
						{
							Id:      "step-1",
							Name:    "duplicate",
							BuildId: "build-1",
							StageId: "stage-1",
							Step:    "plugin1",
						},
						{
							Id:      "step-2",
							Name:    "duplicate", // Duplicate
							BuildId: "build-1",
							StageId: "stage-1",
							Step:    "plugin2",
						},
					},
				},
			},
		},
		stages: make(map[string]*taskStage),
		joblk:  sync.RWMutex{},
		jobs:   make(map[string]*jobSync),
	}

	if bt.check() {
		t.Error("check() should return false when step names are duplicated")
	}
	expected := `step "duplicate" is duplicated`
	if bt.build.Error != expected {
		t.Errorf("expected error %q, got %q", expected, bt.build.Error)
	}
}
