package engine

import (
	"container/list"
	"testing"

	"github.com/gokins/core/runtime"
)

func TestBuildEngine_Get_NilEngine(t *testing.T) {
	var egn *BuildEngine
	task, ok := egn.Get("build1")
	if ok {
		t.Error("Get() on nil engine should return false")
	}
	if task != nil {
		t.Error("Get() on nil engine should return nil task")
	}
}

func TestBuildEngine_Get_EmptyBuildID(t *testing.T) {
	egn := &BuildEngine{
		taskw: list.New(),
		tasks: make(map[string]*BuildTask),
	}
	task, ok := egn.Get("")
	if ok {
		t.Error("Get() with empty buildID should return false")
	}
	if task != nil {
		t.Error("Get() with empty buildID should return nil task")
	}
}

func TestBuildEngine_Get_NonExistent(t *testing.T) {
	egn := &BuildEngine{
		taskw: list.New(),
		tasks: make(map[string]*BuildTask),
	}
	task, ok := egn.Get("nonexistent")
	if ok {
		t.Error("Get() with nonexistent buildID should return false")
	}
	if task != nil {
		t.Error("Get() with nonexistent buildID should return nil task")
	}
}

func TestBuildEngine_Get_Existing(t *testing.T) {
	egn := &BuildEngine{
		taskw: list.New(),
		tasks: make(map[string]*BuildTask),
	}
	bt := &BuildTask{
		build: &runtime.Build{Id: "build1"},
	}
	egn.tasks["build1"] = bt
	
	task, ok := egn.Get("build1")
	if !ok {
		t.Error("Get() with existing buildID should return true")
	}
	if task != bt {
		t.Error("Get() should return the correct BuildTask")
	}
}

func TestBuildEngine_Put(t *testing.T) {
	egn := &BuildEngine{
		taskw: list.New(),
		tasks: make(map[string]*BuildTask),
	}
	bd := &runtime.Build{Id: "build1"}
	egn.Put(bd)
	
	if egn.taskw.Len() != 1 {
		t.Errorf("Put() should add to taskw queue, got length %d", egn.taskw.Len())
	}
}

func TestBuildEngine_Put_Multiple(t *testing.T) {
	egn := &BuildEngine{
		taskw: list.New(),
		tasks: make(map[string]*BuildTask),
	}
	egn.Put(&runtime.Build{Id: "build1"})
	egn.Put(&runtime.Build{Id: "build2"})
	egn.Put(&runtime.Build{Id: "build3"})
	
	if egn.taskw.Len() != 3 {
		t.Errorf("Put() should add multiple items, got length %d", egn.taskw.Len())
	}
}

func TestBuildEngine_Stop_Empty(t *testing.T) {
	egn := &BuildEngine{
		taskw: list.New(),
		tasks: make(map[string]*BuildTask),
	}
	egn.Stop() // Should not panic
}

func TestBuildEngine_Stop_WithTasks(t *testing.T) {
	egn := &BuildEngine{
		taskw: list.New(),
		tasks: make(map[string]*BuildTask),
	}
	bt1 := &BuildTask{
		build: &runtime.Build{Id: "build1"},
	}
	bt2 := &BuildTask{
		build: &runtime.Build{Id: "build2"},
	}
	egn.tasks["build1"] = bt1
	egn.tasks["build2"] = bt2
	
	egn.Stop() // Should not panic and should stop all tasks
}
