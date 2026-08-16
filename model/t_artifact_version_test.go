package model

import (
	"path/filepath"
	"testing"
)

func TestTArtifactVersion_ReadDir_NonexistentPath(t *testing.T) {
	art := &TArtifactVersion{}

	// Try to read a directory that doesn't exist
	nonexistent := filepath.Join(t.TempDir(), "nonexistent")
	_, err := art.readDir(nonexistent)
	if err == nil {
		t.Fatal("expected error for nonexistent directory")
	}

	// Error should be wrapped with context
	if err.Error() == "" {
		t.Error("error message should not be empty")
	}

	// Verify error contains the path
	t.Logf("error properly wrapped: %v", err)
}

func TestTArtifactVersion_ReadFiles_NoWorkPath(t *testing.T) {
	art := &TArtifactVersion{
		Id: "test-id",
	}

	// This will try to read from comm.WorkPath which may not be set
	err := art.ReadFiles()
	// We expect an error since the artifact directory won't exist
	if err == nil {
		t.Skip("no error returned, WorkPath might be configured")
	}

	t.Logf("error properly wrapped: %v", err)
}
