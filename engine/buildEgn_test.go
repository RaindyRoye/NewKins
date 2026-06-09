package engine

import (
	"container/list"
	"fmt"
	"sync"
	"testing"

	"github.com/gokins/core/runtime"
)

func TestBuildEnginePutGet(t *testing.T) {
	c := &BuildEngine{
		taskw: list.New(),
		tasks: make(map[string]*BuildTask),
	}
	bd := &runtime.Build{Id: "build-1"}
	c.Put(bd)
	if c.taskw.Len() != 1 {
		t.Fatalf("expected task queue length 1, got %d", c.taskw.Len())
	}
}

func TestBuildEngineGetEmpty(t *testing.T) {
	c := &BuildEngine{
		taskw: list.New(),
		tasks: make(map[string]*BuildTask),
	}
	_, ok := c.Get("")
	if ok {
		t.Fatal("expected Get with empty ID to return false")
	}
	_, ok = c.Get("nonexistent")
	if ok {
		t.Fatal("expected Get with nonexistent ID to return false")
	}
}

func TestBuildEngineGetExisting(t *testing.T) {
	c := &BuildEngine{
		taskw: list.New(),
		tasks: make(map[string]*BuildTask),
	}
	bt := &BuildTask{build: &runtime.Build{Id: "build-42"}}
	c.tasks["build-42"] = bt

	got, ok := c.Get("build-42")
	if !ok {
		t.Fatal("expected Get to find existing task")
	}
	if got != bt {
		t.Fatal("expected Get to return the same task")
	}
}

func TestBuildEngineConcurrentGet(t *testing.T) {
	c := &BuildEngine{
		taskw: list.New(),
		tasks: make(map[string]*BuildTask),
	}
	// Populate some tasks
	for i := 0; i < 100; i++ {
		id := fmt.Sprintf("build-%d", i)
		c.tasks[id] = &BuildTask{
			build: &runtime.Build{Id: id},
		}
	}

	var wg sync.WaitGroup
	// Concurrent reads should not block each other with RLock
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				c.Get("build-0")
				c.Get("nonexistent")
				c.Get("build-50")
			}
		}()
	}
	wg.Wait()
}

func TestBuildEngineConcurrentGetAndPut(t *testing.T) {
	c := &BuildEngine{
		taskw: list.New(),
		tasks: make(map[string]*BuildTask),
	}

	var wg sync.WaitGroup
	// Concurrent reads via Get
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			for j := 0; j < 50; j++ {
				c.Get(fmt.Sprintf("build-%d", j))
			}
		}(i)
	}
	// Concurrent writes to task queue via Put
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 10; j++ {
				c.Put(&runtime.Build{Id: "new-build"})
			}
		}()
	}
	wg.Wait()

	if c.taskw.Len() != 100 {
		t.Fatalf("expected 100 items in task queue, got %d", c.taskw.Len())
	}
}
