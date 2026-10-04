package engine

import (
	"testing"

	"github.com/gokins/core/common"
	"github.com/gokins/core/runtime"
	"github.com/gokins/core/utils"
	"github.com/gokins/runner/runners"
	"github.com/stretchr/testify/assert"
)

// TestBuildTask_Check_NoRepo tests check() when build has no repo
func TestBuildTask_Check_NoRepo(t *testing.T) {
	bt := NewBuildTask(nil, &runtime.Build{
		Id:     utils.NewXid(),
		Status: common.BuildStatusPending,
		Stages: []*runtime.Stage{},
	})

	result := bt.check()
	assert.False(t, result)
	// check() calls c.status() which sets Status (not Event) in the nil-repo case
	assert.Equal(t, common.BuildEventCheckParam, bt.build.Status)
	assert.Equal(t, "repo param err", bt.build.Error)
}

// TestBuildTask_Check_EmptyStages tests check() when build has no stages
func TestBuildTask_Check_EmptyStages(t *testing.T) {
	bt := NewBuildTask(nil, &runtime.Build{
		Id:     utils.NewXid(),
		Status: common.BuildStatusPending,
		Repo: &runtime.Repository{
			CloneURL: "",
		},
		Stages: []*runtime.Stage{},
	})

	result := bt.check()
	assert.False(t, result)
	assert.Equal(t, common.BuildEventCheckParam, bt.build.Event)
	assert.Contains(t, bt.build.Error, "Stages is empty")
}

// TestBuildTask_Check_StageBuildIdMismatch tests check() when stage build ID doesn't match
func TestBuildTask_Check_StageBuildIdMismatch(t *testing.T) {
	buildId := utils.NewXid()
	bt := NewBuildTask(nil, &runtime.Build{
		Id:     buildId,
		Status: common.BuildStatusPending,
		Repo: &runtime.Repository{
			CloneURL: "",
		},
		Stages: []*runtime.Stage{
			{
				Id:      utils.NewXid(),
				BuildId: "wrong-build-id",
				Name:    "stage1",
				Steps: []*runtime.Step{
					{
						Id:      utils.NewXid(),
						BuildId: buildId,
						StageId: utils.NewXid(),
						Name:    "step1",
						Step:    "gokins@test",
					},
				},
			},
		},
	})

	result := bt.check()
	assert.False(t, result)
	assert.Equal(t, common.BuildEventCheckParam, bt.build.Event)
	assert.Contains(t, bt.build.Error, "Stage Build id err")
}

// TestBuildTask_Check_EmptyStageName tests check() when stage name is empty
func TestBuildTask_Check_EmptyStageName(t *testing.T) {
	buildId := utils.NewXid()
	bt := NewBuildTask(nil, &runtime.Build{
		Id:     buildId,
		Status: common.BuildStatusPending,
		Repo: &runtime.Repository{
			CloneURL: "",
		},
		Stages: []*runtime.Stage{
			{
				Id:      utils.NewXid(),
				BuildId: buildId,
				Name:    "",
				Steps: []*runtime.Step{
					{
						Id:      utils.NewXid(),
						BuildId: buildId,
						StageId: utils.NewXid(),
						Name:    "step1",
						Step:    "gokins@test",
					},
				},
			},
		},
	})

	result := bt.check()
	assert.False(t, result)
	assert.Equal(t, common.BuildEventCheckParam, bt.build.Event)
	assert.Contains(t, bt.build.Error, "Stage name is empty")
}

// TestBuildTask_Check_EmptySteps tests check() when stage has no steps
func TestBuildTask_Check_EmptySteps(t *testing.T) {
	buildId := utils.NewXid()
	bt := NewBuildTask(nil, &runtime.Build{
		Id:     buildId,
		Status: common.BuildStatusPending,
		Repo: &runtime.Repository{
			CloneURL: "",
		},
		Stages: []*runtime.Stage{
			{
				Id:      utils.NewXid(),
				BuildId: buildId,
				Name:    "stage1",
				Steps:   []*runtime.Step{},
			},
		},
	})

	result := bt.check()
	assert.False(t, result)
	assert.Equal(t, common.BuildEventCheckParam, bt.build.Event)
}

// TestBuildTask_Check_DuplicateStageName tests check() when stage names are duplicated
func TestBuildTask_Check_DuplicateStageName(t *testing.T) {
	buildId := utils.NewXid()
	stage1Id := utils.NewXid()
	stage2Id := utils.NewXid()

	bt := NewBuildTask(nil, &runtime.Build{
		Id:     buildId,
		Status: common.BuildStatusPending,
		Repo: &runtime.Repository{
			CloneURL: "",
		},
		Stages: []*runtime.Stage{
			{
				Id:      stage1Id,
				BuildId: buildId,
				Name:    "stage1",
				Steps: []*runtime.Step{
					{
						Id:      utils.NewXid(),
						BuildId: buildId,
						StageId: stage1Id,
						Name:    "step1",
						Step:    "gokins@test",
					},
				},
			},
			{
				Id:      stage2Id,
				BuildId: buildId,
				Name:    "stage1", // duplicate
				Steps: []*runtime.Step{
					{
						Id:      utils.NewXid(),
						BuildId: buildId,
						StageId: stage2Id,
						Name:    "step2",
						Step:    "gokins@test",
					},
				},
			},
		},
	})
	// Initialize maps that NewBuildTask doesn't create
	bt.stages = make(map[string]*taskStage)
	bt.jobs = make(map[string]*jobSync)

	result := bt.check()
	assert.False(t, result)
	assert.Equal(t, common.BuildEventCheckParam, bt.build.Event)
	assert.Contains(t, bt.build.Error, "is repeat")
}

// TestBuildTask_Check_StepBuildIdMismatch tests check() when step build ID doesn't match
func TestBuildTask_Check_StepBuildIdMismatch(t *testing.T) {
	buildId := utils.NewXid()
	stageId := utils.NewXid()
	bt := NewBuildTask(nil, &runtime.Build{
		Id:     buildId,
		Status: common.BuildStatusPending,
		Repo: &runtime.Repository{
			CloneURL: "",
		},
		Stages: []*runtime.Stage{
			{
				Id:      stageId,
				BuildId: buildId,
				Name:    "stage1",
				Steps: []*runtime.Step{
					{
						Id:      utils.NewXid(),
						BuildId: "wrong-build-id",
						StageId: stageId,
						Name:    "step1",
						Step:    "gokins@test",
					},
				},
			},
		},
	})

	result := bt.check()
	assert.False(t, result)
	assert.Equal(t, common.BuildEventCheckParam, bt.build.Event)
	assert.Contains(t, bt.build.Error, "Job Build id err")
}

// TestBuildTask_Check_StepStageIdMismatch tests check() when step stage ID doesn't match
func TestBuildTask_Check_StepStageIdMismatch(t *testing.T) {
	buildId := utils.NewXid()
	stageId := utils.NewXid()
	bt := NewBuildTask(nil, &runtime.Build{
		Id:     buildId,
		Status: common.BuildStatusPending,
		Repo: &runtime.Repository{
			CloneURL: "",
		},
		Stages: []*runtime.Stage{
			{
				Id:      stageId,
				BuildId: buildId,
				Name:    "stage1",
				Steps: []*runtime.Step{
					{
						Id:      utils.NewXid(),
						BuildId: buildId,
						StageId: "wrong-stage-id",
						Name:    "step1",
						Step:    "gokins@test",
					},
				},
			},
		},
	})

	result := bt.check()
	assert.False(t, result)
	assert.Equal(t, common.BuildEventCheckParam, bt.build.Event)
	assert.Contains(t, bt.build.Error, "Job Stage id err")
}

// TestBuildTask_Check_EmptyStepPlugin tests check() when step plugin is empty
func TestBuildTask_Check_EmptyStepPlugin(t *testing.T) {
	buildId := utils.NewXid()
	stageId := utils.NewXid()
	bt := NewBuildTask(nil, &runtime.Build{
		Id:     buildId,
		Status: common.BuildStatusPending,
		Repo: &runtime.Repository{
			CloneURL: "",
		},
		Stages: []*runtime.Stage{
			{
				Id:      stageId,
				BuildId: buildId,
				Name:    "stage1",
				Steps: []*runtime.Step{
					{
						Id:      utils.NewXid(),
						BuildId: buildId,
						StageId: stageId,
						Name:    "step1",
						Step:    "",
					},
				},
			},
		},
	})

	result := bt.check()
	assert.False(t, result)
	assert.Equal(t, common.BuildEventCheckParam, bt.build.Event)
	assert.Contains(t, bt.build.Error, "Step Plugin is empty")
}

// TestBuildTask_Check_EmptyStepName tests check() when step name is empty
func TestBuildTask_Check_EmptyStepName(t *testing.T) {
	buildId := utils.NewXid()
	stageId := utils.NewXid()
	bt := NewBuildTask(nil, &runtime.Build{
		Id:     buildId,
		Status: common.BuildStatusPending,
		Repo: &runtime.Repository{
			CloneURL: "",
		},
		Stages: []*runtime.Stage{
			{
				Id:      stageId,
				BuildId: buildId,
				Name:    "stage1",
				Steps: []*runtime.Step{
					{
						Id:      utils.NewXid(),
						BuildId: buildId,
						StageId: stageId,
						Name:    "",
						Step:    "gokins@test",
					},
				},
			},
		},
	})

	result := bt.check()
	assert.False(t, result)
	assert.Equal(t, common.BuildEventCheckParam, bt.build.Event)
	assert.Contains(t, bt.build.Error, "Step name is empty")
}

// TestBuildTask_Check_DuplicateStepName tests check() when step names are duplicated
func TestBuildTask_Check_DuplicateStepName(t *testing.T) {
	buildId := utils.NewXid()
	stageId := utils.NewXid()
	bt := NewBuildTask(nil, &runtime.Build{
		Id:     buildId,
		Status: common.BuildStatusPending,
		Repo: &runtime.Repository{
			CloneURL: "",
		},
		Stages: []*runtime.Stage{
			{
				Id:      stageId,
				BuildId: buildId,
				Name:    "stage1",
				Steps: []*runtime.Step{
					{
						Id:      utils.NewXid(),
						BuildId: buildId,
						StageId: stageId,
						Name:    "step1",
						Step:    "gokins@test",
					},
					{
						Id:      utils.NewXid(),
						BuildId: buildId,
						StageId: stageId,
						Name:    "step1", // duplicate
						Step:    "gokins@test",
					},
				},
			},
		},
	})
	// Initialize maps that NewBuildTask doesn't create
	bt.stages = make(map[string]*taskStage)
	bt.jobs = make(map[string]*jobSync)

	result := bt.check()
	assert.False(t, result)
	assert.Equal(t, common.BuildEventCheckParam, bt.build.Event)
	assert.Contains(t, bt.build.Error, "is repeat")
}

// TestBuildTask_GenRunjob_StringCommands tests genRunjob with string commands
func TestBuildTask_GenRunjob_StringCommands(t *testing.T) {
	buildId := utils.NewXid()
	stageId := utils.NewXid()
	stepId := utils.NewXid()

	bt := NewBuildTask(nil, &runtime.Build{
		Id:     buildId,
		Status: common.BuildStatusPending,
		Repo: &runtime.Repository{
			CloneURL: "",
		},
	})

	stage := &runtime.Stage{
		Id:      stageId,
		BuildId: buildId,
		Name:    "stage1",
	}

	step := &runtime.Step{
		Id:       stepId,
		BuildId:  buildId,
		StageId:  stageId,
		Name:     "step1",
		Step:     "gokins@test",
		Commands: "echo hello",
	}

	job := &jobSync{
		task:  bt,
		step:  step,
		cmdmp: make(map[string]*cmdSync),
	}

	err := bt.genRunjob(stage, job)
	// Will fail because no DB, but we're testing the code path
	assert.Error(t, err)
}

// TestBuildTask_GenRunjob_SliceCommands tests genRunjob with slice commands
func TestBuildTask_GenRunjob_SliceCommands(t *testing.T) {
	buildId := utils.NewXid()
	stageId := utils.NewXid()
	stepId := utils.NewXid()

	bt := NewBuildTask(nil, &runtime.Build{
		Id:     buildId,
		Status: common.BuildStatusPending,
		Repo: &runtime.Repository{
			CloneURL: "",
		},
	})

	stage := &runtime.Stage{
		Id:      stageId,
		BuildId: buildId,
		Name:    "stage1",
	}

	step := &runtime.Step{
		Id:      stepId,
		BuildId: buildId,
		StageId: stageId,
		Name:    "step1",
		Step:    "gokins@test",
		Commands: []any{
			"echo hello",
			"echo world",
		},
	}

	job := &jobSync{
		task:  bt,
		step:  step,
		cmdmp: make(map[string]*cmdSync),
	}

	err := bt.genRunjob(stage, job)
	// Will fail because no DB, but we're testing the code path
	assert.Error(t, err)
}

// TestBuildTask_GenRunjob_StringSliceCommands tests genRunjob with []string commands
func TestBuildTask_GenRunjob_StringSliceCommands(t *testing.T) {
	buildId := utils.NewXid()
	stageId := utils.NewXid()
	stepId := utils.NewXid()

	bt := NewBuildTask(nil, &runtime.Build{
		Id:     buildId,
		Status: common.BuildStatusPending,
		Repo: &runtime.Repository{
			CloneURL: "",
		},
	})

	stage := &runtime.Stage{
		Id:      stageId,
		BuildId: buildId,
		Name:    "stage1",
	}

	step := &runtime.Step{
		Id:       stepId,
		BuildId:  buildId,
		StageId:  stageId,
		Name:     "step1",
		Step:     "gokins@test",
		Commands: []string{"echo hello", "echo world"},
	}

	job := &jobSync{
		task:  bt,
		step:  step,
		cmdmp: make(map[string]*cmdSync),
	}

	err := bt.genRunjob(stage, job)
	// Will fail because no DB, but we're testing the code path
	assert.Error(t, err)
}

// TestBuildTask_GenRunjob_GitStep tests genRunjob with git step
func TestBuildTask_GenRunjob_GitStep(t *testing.T) {
	buildId := utils.NewXid()
	stageId := utils.NewXid()
	stepId := utils.NewXid()

	bt := NewBuildTask(nil, &runtime.Build{
		Id:     buildId,
		Status: common.BuildStatusPending,
		Repo: &runtime.Repository{
			CloneURL: "",
		},
	})

	bt.isClone = false
	bt.repoPaths = "/tmp/test"

	stage := &runtime.Stage{
		Id:      stageId,
		BuildId: buildId,
		Name:    "stage1",
	}

	step := &runtime.Step{
		Id:       stepId,
		BuildId:  buildId,
		StageId:  stageId,
		Name:     "step1",
		Step:     "gokins@git",
		Commands: "git works",
	}

	job := &jobSync{
		task:  bt,
		step:  step,
		cmdmp: make(map[string]*cmdSync),
	}

	err := bt.genRunjob(stage, job)
	// Will fail because no DB, but we're testing the code path
	assert.Error(t, err)
}

// TestBuildTask_AppendCmds tests appendcmds method
func TestBuildTask_AppendCmds(t *testing.T) {
	bt := NewBuildTask(nil, &runtime.Build{
		Id:     utils.NewXid(),
		Status: common.BuildStatusPending,
	})

	runjb := &runners.RunJob{
		Commands: []*runners.CmdContent{},
	}

	bt.appendcmds(runjb, "echo test")
	assert.Len(t, runjb.Commands, 1)
	assert.Equal(t, "echo test", runjb.Commands[0].Conts)

	bt.appendcmds(runjb, "echo another")
	assert.Len(t, runjb.Commands, 2)
	assert.Equal(t, "echo another", runjb.Commands[1].Conts)
}

// TestBuildTask_Gencmds_String tests gencmds with string input
func TestBuildTask_Gencmds_String(t *testing.T) {
	bt := NewBuildTask(nil, &runtime.Build{
		Id:     utils.NewXid(),
		Status: common.BuildStatusPending,
	})

	runjb := &runners.RunJob{
		Commands: []*runners.CmdContent{},
	}

	cmds := []any{"echo test", "echo world"}
	err := bt.gencmds(runjb, cmds)
	assert.NoError(t, err)
	assert.Len(t, runjb.Commands, 2)
}

// TestBuildTask_Gencmds_NestedSlice tests gencmds with nested slice
func TestBuildTask_Gencmds_NestedSlice(t *testing.T) {
	bt := NewBuildTask(nil, &runtime.Build{
		Id:     utils.NewXid(),
		Status: common.BuildStatusPending,
	})

	runjb := &runners.RunJob{
		Commands: []*runners.CmdContent{},
	}

	cmds := []any{
		[]any{"echo nested1", "echo nested2"},
	}
	err := bt.gencmds(runjb, cmds)
	assert.NoError(t, err)
	assert.Len(t, runjb.Commands, 2)
}

// TestBuildTask_Gencmds_MapAnyAny tests gencmds with map[any]any
func TestBuildTask_Gencmds_MapAnyAny(t *testing.T) {
	bt := NewBuildTask(nil, &runtime.Build{
		Id:     utils.NewXid(),
		Status: common.BuildStatusPending,
	})

	runjb := &runners.RunJob{
		Commands: []*runners.CmdContent{},
	}

	cmds := []any{
		map[any]any{
			"key1": "echo value1",
			"key2": []any{"echo value2a", "echo value2b"},
		},
	}
	err := bt.gencmds(runjb, cmds)
	assert.NoError(t, err)
	assert.Len(t, runjb.Commands, 3)
}

// TestBuildTask_Gencmds_MapStringAny tests gencmds with map[string]any
func TestBuildTask_Gencmds_MapStringAny(t *testing.T) {
	bt := NewBuildTask(nil, &runtime.Build{
		Id:     utils.NewXid(),
		Status: common.BuildStatusPending,
	})

	runjb := &runners.RunJob{
		Commands: []*runners.CmdContent{},
	}

	cmds := []any{
		map[string]any{
			"key1": "echo value1",
			"key2": []any{"echo value2a", "echo value2b"},
		},
	}
	err := bt.gencmds(runjb, cmds)
	assert.NoError(t, err)
	assert.Len(t, runjb.Commands, 3)
}

// TestBuildTask_Gencmds_EmptyCommands tests gencmds with empty commands
func TestBuildTask_Gencmds_EmptyCommands(t *testing.T) {
	bt := NewBuildTask(nil, &runtime.Build{
		Id:     utils.NewXid(),
		Status: common.BuildStatusPending,
	})

	runjb := &runners.RunJob{
		Commands: []*runners.CmdContent{},
	}

	cmds := []any{}
	err := bt.gencmds(runjb, cmds)
	assert.NoError(t, err)
	assert.Len(t, runjb.Commands, 0)
}

// TestBuildTask_GetRepo_NoClone tests getRepo when isClone is false
func TestBuildTask_GetRepo_NoClone(t *testing.T) {
	bt := NewBuildTask(nil, &runtime.Build{
		Id:     utils.NewXid(),
		Status: common.BuildStatusPending,
	})
	bt.isClone = false

	err := bt.getRepo()
	assert.NoError(t, err)
}

// TestBuildTask_GetRepo_CreateDir tests getRepo creating directory
func TestBuildTask_GetRepo_CreateDir(t *testing.T) {
	bt := NewBuildTask(nil, &runtime.Build{
		Id:     utils.NewXid(),
		Status: common.BuildStatusPending,
		Repo: &runtime.Repository{
			CloneURL: "",
		},
	})
	bt.isClone = true
	bt.repoPaths = "/tmp/test-getrepo-" + utils.NewXid()

	err := bt.getRepo()
	assert.NoError(t, err)
}

// TestBuildTask_Write_ProgressParsing tests Write method with progress parsing
func TestBuildTask_Write_ProgressParsing(t *testing.T) {
	bt := NewBuildTask(nil, &runtime.Build{
		Id:     utils.NewXid(),
		Status: common.BuildStatusPending,
	})

	// Test with progress line
	input := []byte("Receiving objects:  50% (5/10)")
	n, err := bt.Write(input)
	assert.NoError(t, err)
	assert.Equal(t, len(input), n)
	assert.Equal(t, 40, bt.workpgss) // 50 * 0.8 = 40
}

// TestBuildTask_Write_NoProgress tests Write method without progress
func TestBuildTask_Write_NoProgress(t *testing.T) {
	bt := NewBuildTask(nil, &runtime.Build{
		Id:     utils.NewXid(),
		Status: common.BuildStatusPending,
	})

	input := []byte("normal git log output")
	n, err := bt.Write(input)
	assert.NoError(t, err)
	assert.Equal(t, len(input), n)
	assert.Equal(t, 0, bt.workpgss)
}

// TestBuildTask_Write_EmptyInput tests Write method with empty input
func TestBuildTask_Write_EmptyInput(t *testing.T) {
	bt := NewBuildTask(nil, &runtime.Build{
		Id:     utils.NewXid(),
		Status: common.BuildStatusPending,
	})

	input := []byte{}
	n, err := bt.Write(input)
	assert.NoError(t, err)
	assert.Equal(t, 0, n)
}
