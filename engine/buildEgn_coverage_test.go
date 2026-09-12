package engine

import (
	"container/list"
	"testing"
)

func TestBuildEngineStop(t *testing.T) {
	c := &BuildEngine{
		taskw: list.New(),
		tasks: make(map[string]*BuildTask),
	}
	// Add some tasks
	c.tasks["build-1"] = &BuildTask{}
	c.tasks["build-2"] = &BuildTask{}
	
	// Stop should not panic even with empty tasks
	c.Stop()
}

func TestBuildEngineNilGet(t *testing.T) {
	var c *BuildEngine
	_, ok := c.Get("test")
	if ok {
		t.Error("Get on nil BuildEngine should return false")
	}
}
