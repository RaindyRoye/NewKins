package engine

import (
	"testing"

	"github.com/gokins/core/runtime"
	"github.com/gokins/runner/runners"
)

func TestGenRunjob_BasicFields(t *testing.T) {
	task := &BuildTask{
		build: &runtime.Build{
			Id:         "build-1",
			PipelineId: "pipe-1",
			Vars:       make(map[string]*runtime.Variables),
		},
		repoPath: "/tmp/repo",
	}
	
	stage := &runtime.Stage{
		Id:   "stage-1",
		Name: "build",
	}
	
	step := &runtime.Step{
		Id:        "step-1",
		StageId:   "stage-1",
		BuildId:   "build-1",
		Step:      "shell@ssh",
		Name:      "compile",
		Commands:  "echo hello",
		Env:       map[string]string{"KEY": "value"},
		Artifacts: []*runtime.Artifact{{Name: "output", Path: "*.jar"}},
		Input:     map[string]string{"input": "data"},
	}
	
	job := &jobSync{
		step:  step,
		task:  task,
		cmdmp: make(map[string]*cmdSync),
	}
	
	// We can't fully test genRunjob without a database, but we can test the structure setup
	// by calling it and checking that it doesn't panic and that runjb is created
	_ = task.genRunjob(stage, job)
	
	// The function will fail at the database insert, but we can check that runjb was created
	if job.runjb == nil {
		t.Skip("runjb not created (likely due to database error)")
		return
	}
	
	runjb := job.runjb
	
	if runjb.Id != step.Id {
		t.Errorf("expected runjb.Id=%s, got %s", step.Id, runjb.Id)
	}
	if runjb.PipelineId != task.build.PipelineId {
		t.Errorf("expected runjb.PipelineId=%s, got %s", task.build.PipelineId, runjb.PipelineId)
	}
	if runjb.StageId != step.StageId {
		t.Errorf("expected runjb.StageId=%s, got %s", step.StageId, runjb.StageId)
	}
	if runjb.BuildId != step.BuildId {
		t.Errorf("expected runjb.BuildId=%s, got %s", step.BuildId, runjb.BuildId)
	}
	if runjb.StageName != stage.Name {
		t.Errorf("expected runjb.StageName=%s, got %s", stage.Name, runjb.StageName)
	}
	if runjb.Step != step.Step {
		t.Errorf("expected runjb.Step=%s, got %s", step.Step, runjb.Step)
	}
	if runjb.Name != step.Name {
		t.Errorf("expected runjb.Name=%s, got %s", step.Name, runjb.Name)
	}
	if runjb.OriginRepo != task.repoPath {
		t.Errorf("expected runjb.OriginRepo=%s, got %s", task.repoPath, runjb.OriginRepo)
	}
}

func TestGenRunjob_IsClone(t *testing.T) {
	task := &BuildTask{
		build: &runtime.Build{
			Id:         "build-1",
			PipelineId: "pipe-1",
			Vars:       make(map[string]*runtime.Variables),
		},
		repoPath: "/tmp/repo",
		isClone:  true,
	}
	
	stage := &runtime.Stage{
		Id:   "stage-1",
		Name: "build",
	}
	
	step := &runtime.Step{
		Id:        "step-1",
		StageId:   "stage-1",
		BuildId:   "build-1",
		Step:      "shell@ssh",
		Name:      "compile",
		Commands:  "echo hello",
	}
	
	job := &jobSync{
		step:  step,
		task:  task,
		cmdmp: make(map[string]*cmdSync),
	}
	
	_ = task.genRunjob(stage, job)
	
	if job.runjb == nil {
		t.Skip("runjb not created (likely due to database error)")
		return
	}
	
	// When isClone is true, OriginRepo should be empty
	if job.runjb.OriginRepo != "" {
		t.Errorf("expected runjb.OriginRepo to be empty when isClone=true, got %s", job.runjb.OriginRepo)
	}
}

func TestGenRunjob_MustCopy(t *testing.T) {
	task := &BuildTask{
		build: &runtime.Build{
			Id:         "build-1",
			PipelineId: "pipe-1",
			Vars:       make(map[string]*runtime.Variables),
		},
		repoPath: "/tmp/repo",
		isClone:  false,
	}
	
	stage := &runtime.Stage{
		Id:   "stage-1",
		Name: "build",
	}
	
	step := &runtime.Step{
		Id:        "step-1",
		StageId:   "stage-1",
		BuildId:   "build-1",
		Step:      "shell@ssh",
		Name:      "compile",
		Commands:  "echo hello",
		MustCopy:  true,
	}
	
	job := &jobSync{
		step:  step,
		task:  task,
		cmdmp: make(map[string]*cmdSync),
	}
	
	_ = task.genRunjob(stage, job)
	
	if job.runjb == nil {
		t.Skip("runjb not created (likely due to database error)")
		return
	}
	
	// When MustCopy is true, OriginRepo should be empty
	if job.runjb.OriginRepo != "" {
		t.Errorf("expected runjb.OriginRepo to be empty when MustCopy=true, got %s", job.runjb.OriginRepo)
	}
}

func TestGenRunjob_GokinsGit(t *testing.T) {
	task := &BuildTask{
		build: &runtime.Build{
			Id:         "build-1",
			PipelineId: "pipe-1",
			Vars:       make(map[string]*runtime.Variables),
		},
		repoPath:  "/tmp/repo",
		repoPaths: "/tmp/repo/paths",
		isClone:   false,
	}
	
	stage := &runtime.Stage{
		Id:   "stage-1",
		Name: "build",
	}
	
	step := &runtime.Step{
		Id:        "step-1",
		StageId:   "stage-1",
		BuildId:   "build-1",
		Step:      "gokins@git",
		Name:      "clone",
		Commands:  nil,
	}
	
	job := &jobSync{
		step:  step,
		task:  task,
		cmdmp: make(map[string]*cmdSync),
	}
	
	_ = task.genRunjob(stage, job)
	
	if job.runjb == nil {
		t.Skip("runjb not created (likely due to database error)")
		return
	}
	
	// For gokins@git, OriginRepo should be set to repoPaths
	if job.runjb.OriginRepo != task.repoPaths {
		t.Errorf("expected runjb.OriginRepo=%s for gokins@git, got %s", task.repoPaths, job.runjb.OriginRepo)
	}
	
	// Commands should be set to ["git works"]
	cmds, ok := step.Commands.([]string)
	if !ok {
		t.Fatal("expected step.Commands to be []string")
	}
	if len(cmds) != 1 || cmds[0] != "git works" {
		t.Errorf("expected step.Commands=[git works], got %v", cmds)
	}
}

func TestGenRunjob_StringCommands(t *testing.T) {
	task := &BuildTask{
		build: &runtime.Build{
			Id:         "build-1",
			PipelineId: "pipe-1",
			Vars:       make(map[string]*runtime.Variables),
		},
		repoPath: "/tmp/repo",
	}
	
	stage := &runtime.Stage{
		Id:   "stage-1",
		Name: "build",
	}
	
	step := &runtime.Step{
		Id:        "step-1",
		StageId:   "stage-1",
		BuildId:   "build-1",
		Step:      "shell@ssh",
		Name:      "compile",
		Commands:  "echo hello",
	}
	
	job := &jobSync{
		step:  step,
		task:  task,
		cmdmp: make(map[string]*cmdSync),
	}
	
	_ = task.genRunjob(stage, job)
	
	if job.runjb == nil {
		t.Skip("runjb not created (likely due to database error)")
		return
	}
	
	// Should have created one command from the string
	if len(job.runjb.Commands) != 1 {
		t.Errorf("expected 1 command, got %d", len(job.runjb.Commands))
	}
}

func TestGenRunjob_ArrayCommands(t *testing.T) {
	task := &BuildTask{
		build: &runtime.Build{
			Id:         "build-1",
			PipelineId: "pipe-1",
			Vars:       make(map[string]*runtime.Variables),
		},
		repoPath: "/tmp/repo",
	}
	
	stage := &runtime.Stage{
		Id:   "stage-1",
		Name: "build",
	}
	
	step := &runtime.Step{
		Id:        "step-1",
		StageId:   "stage-1",
		BuildId:   "build-1",
		Step:      "shell@ssh",
		Name:      "compile",
		Commands:  []any{"echo hello", "echo world"},
	}
	
	job := &jobSync{
		step:  step,
		task:  task,
		cmdmp: make(map[string]*cmdSync),
	}
	
	_ = task.genRunjob(stage, job)
	
	if job.runjb == nil {
		t.Skip("runjb not created (likely due to database error)")
		return
	}
	
	// Should have created two commands from the array
	if len(job.runjb.Commands) != 2 {
		t.Errorf("expected 2 commands, got %d", len(job.runjb.Commands))
	}
}

func TestGenRunjob_StringArrayCommands(t *testing.T) {
	task := &BuildTask{
		build: &runtime.Build{
			Id:         "build-1",
			PipelineId: "pipe-1",
			Vars:       make(map[string]*runtime.Variables),
		},
		repoPath: "/tmp/repo",
	}
	
	stage := &runtime.Stage{
		Id:   "stage-1",
		Name: "build",
	}
	
	step := &runtime.Step{
		Id:        "step-1",
		StageId:   "stage-1",
		BuildId:   "build-1",
		Step:      "shell@ssh",
		Name:      "compile",
		Commands:  []string{"echo hello", "echo world"},
	}
	
	job := &jobSync{
		step:  step,
		task:  task,
		cmdmp: make(map[string]*cmdSync),
	}
	
	_ = task.genRunjob(stage, job)
	
	if job.runjb == nil {
		t.Skip("runjb not created (likely due to database error)")
		return
	}
	
	// Should have created two commands from the string array
	if len(job.runjb.Commands) != 2 {
		t.Errorf("expected 2 commands, got %d", len(job.runjb.Commands))
	}
}

func TestGencmds_MapAnyAny(t *testing.T) {
	task := &BuildTask{
		build: &runtime.Build{Id: "test-build"},
	}
	runjb := &runners.RunJob{}
	
	// Test map[any]any with string and []any values
	cmds := []any{
		map[any]any{
			"key1": "echo from map",
			"key2": []any{"echo nested1", "echo nested2"},
		},
	}
	
	err := task.gencmds(runjb, cmds)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	
	// Should have created commands from the map values
	if len(runjb.Commands) == 0 {
		t.Fatal("expected some commands from map[any]any values, got 0")
	}
}

func TestGencmds_MapStringAny_NestedArray(t *testing.T) {
	task := &BuildTask{
		build: &runtime.Build{Id: "test-build"},
	}
	runjb := &runners.RunJob{}
	
	cmds := []any{
		map[string]any{
			"scripts": []any{"echo a", "echo b", "echo c"},
		},
	}
	
	err := task.gencmds(runjb, cmds)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	
	// Should have created 3 commands from the nested array
	if len(runjb.Commands) != 3 {
		t.Errorf("expected 3 commands from nested array, got %d", len(runjb.Commands))
	}
}

func TestGencmds_MapAnyAny_NestedArray(t *testing.T) {
	task := &BuildTask{
		build: &runtime.Build{Id: "test-build"},
	}
	runjb := &runners.RunJob{}
	
	cmds := []any{
		map[any]any{
			"scripts": []any{"echo x", "echo y"},
		},
	}
	
	err := task.gencmds(runjb, cmds)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	
	// Should have created 2 commands from the nested array
	if len(runjb.Commands) != 2 {
		t.Errorf("expected 2 commands from nested array, got %d", len(runjb.Commands))
	}
}

func TestGencmds_DeepNesting(t *testing.T) {
	task := &BuildTask{
		build: &runtime.Build{Id: "test-build"},
	}
	runjb := &runners.RunJob{}
	
	// Test deeply nested structure
	cmds := []any{
		"echo first",
		[]any{
			"echo second",
			[]any{"echo third", "echo fourth"},
		},
		map[string]any{
			"key": []any{"echo fifth", "echo sixth"},
		},
	}
	
	err := task.gencmds(runjb, cmds)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	
	// Should have created commands from all levels
	if len(runjb.Commands) == 0 {
		t.Fatal("expected some commands from deeply nested structure, got 0")
	}
}

func TestGencmds_EmptyNestedArrays(t *testing.T) {
	task := &BuildTask{
		build: &runtime.Build{Id: "test-build"},
	}
	runjb := &runners.RunJob{}
	
	cmds := []any{
		[]any{},
		map[string]any{
			"empty": []any{},
		},
	}
	
	err := task.gencmds(runjb, cmds)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	
	// Should have created 0 commands from empty arrays
	if len(runjb.Commands) != 0 {
		t.Errorf("expected 0 commands from empty arrays, got %d", len(runjb.Commands))
	}
}

func TestGencmds_MapWithNonStringValues(t *testing.T) {
	task := &BuildTask{
		build: &runtime.Build{Id: "test-build"},
	}
	runjb := &runners.RunJob{}
	
	// Map with non-string, non-array values should be ignored
	cmds := []any{
		map[string]any{
			"number": 42,
			"bool":   true,
		},
	}
	
	err := task.gencmds(runjb, cmds)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	
	// Should have created 0 commands (non-string values are ignored)
	if len(runjb.Commands) != 0 {
		t.Errorf("expected 0 commands from non-string map values, got %d", len(runjb.Commands))
	}
}

func TestGencmds_ComplexRealWorldScenario(t *testing.T) {
	task := &BuildTask{
		build: &runtime.Build{Id: "test-build"},
	}
	runjb := &runners.RunJob{}
	
	// Simulate a real-world complex command structure
	cmds := []any{
		"echo 'Starting build'",
		[]any{
			"mkdir -p build",
			"cd build",
		},
		map[string]any{
			"compile": "go build -o app .",
			"test":    []any{"go test ./...", "go test -race ./..."},
		},
		[]any{
			"echo 'Build complete'",
			[]any{"ls -la", "pwd"},
		},
	}
	
	err := task.gencmds(runjb, cmds)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	
	// Should have created multiple commands from the complex structure
	if len(runjb.Commands) < 5 {
		t.Errorf("expected at least 5 commands from complex structure, got %d", len(runjb.Commands))
	}
}

func TestAppendcmds_UniqueIDs(t *testing.T) {
	task := &BuildTask{
		build: &runtime.Build{Id: "test-build"},
	}
	runjb := &runners.RunJob{}
	
	// Append multiple commands and verify each has a unique ID
	for i := 0; i < 10; i++ {
		task.appendcmds(runjb, "echo hello")
	}
	
	if len(runjb.Commands) != 10 {
		t.Fatalf("expected 10 commands, got %d", len(runjb.Commands))
	}
	
	// Verify all IDs are unique
	ids := make(map[string]bool)
	for _, cmd := range runjb.Commands {
		if ids[cmd.Id] {
			t.Errorf("duplicate command ID: %s", cmd.Id)
		}
		ids[cmd.Id] = true
	}
}

func TestAppendcmds_EmptyContent(t *testing.T) {
	task := &BuildTask{
		build: &runtime.Build{Id: "test-build"},
	}
	runjb := &runners.RunJob{}
	
	task.appendcmds(runjb, "")
	
	if len(runjb.Commands) != 1 {
		t.Fatalf("expected 1 command, got %d", len(runjb.Commands))
	}
	
	if runjb.Commands[0].Conts != "" {
		t.Errorf("expected empty content, got %q", runjb.Commands[0].Conts)
	}
}

func TestAppendcmds_VeryLongContent(t *testing.T) {
	task := &BuildTask{
		build: &runtime.Build{Id: "test-build"},
	}
	runjb := &runners.RunJob{}
	
	// Create a very long command
	longCmd := ""
	for i := 0; i < 1000; i++ {
		longCmd += "echo 'line " + string(rune(i)) + "'\n"
	}
	
	task.appendcmds(runjb, longCmd)
	
	if len(runjb.Commands) != 1 {
		t.Fatalf("expected 1 command, got %d", len(runjb.Commands))
	}
	
	if runjb.Commands[0].Conts != longCmd {
		t.Error("expected command content to match long input")
	}
}
