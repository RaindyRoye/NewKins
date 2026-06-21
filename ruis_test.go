package main

import (
	"testing"

	"github.com/gokins/gokins/comm"
)

// TestVersion verifies that the version string is set and not empty.
func TestVersion(t *testing.T) {
	if comm.Version == "" {
		t.Error("comm.Version should not be empty")
	}
	t.Logf("version: %s, buildtime: %s, commit: %s", comm.Version, comm.BuildTime, comm.GitCommit)
}
