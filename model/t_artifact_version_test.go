package model

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/gokins/core/common"
	"github.com/gokins/gokins/comm"
	hbtp "github.com/mgr9525/HyperByte-Transfer-Protocol"
)

func TestTArtifactVersion_ReadFiles_NonExistent(t *testing.T) {
	// Use a temp dir so comm.WorkPath doesn't interfere
	tmpDir := t.TempDir()
	comm.WorkPath = tmpDir

	av := &TArtifactVersion{Id: "nonexistent-id"}
	err := av.ReadFiles()
	if err == nil {
		t.Fatal("expected error for non-existent artifact directory, got nil")
	}
	// Error should be wrapped with context
	if av.Files != nil {
		t.Error("expected Files to be nil on error")
	}
}

func TestTArtifactVersion_ReadFiles_Empty(t *testing.T) {
	tmpDir := t.TempDir()
	comm.WorkPath = tmpDir

	// Create the artifact directory (empty)
	artId := "test-art-id"
	artDir := filepath.Join(tmpDir, common.PathArtifacts, artId)
	if err := os.MkdirAll(artDir, 0750); err != nil {
		t.Fatalf("failed to create test dir: %v", err)
	}

	av := &TArtifactVersion{Id: artId}
	err := av.ReadFiles()
	if err != nil {
		t.Fatalf("unexpected error for empty artifact dir: %v", err)
	}
	if av.Files == nil {
		// Empty dir returns nil slice, which is fine
		return
	}
	if len(av.Files) != 0 {
		t.Errorf("expected 0 files, got %d", len(av.Files))
	}
}

func TestTArtifactVersion_ReadFiles_WithFiles(t *testing.T) {
	tmpDir := t.TempDir()
	comm.WorkPath = tmpDir

	artId := "test-art-with-files"
	artDir := filepath.Join(tmpDir, common.PathArtifacts, artId)
	if err := os.MkdirAll(artDir, 0750); err != nil {
		t.Fatalf("failed to create test dir: %v", err)
	}

	// Create some files
	if err := os.WriteFile(filepath.Join(artDir, "file1.txt"), []byte("hello"), 0600); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}
	if err := os.WriteFile(filepath.Join(artDir, "file2.bin"), []byte("binary"), 0600); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	av := &TArtifactVersion{Id: artId}
	err := av.ReadFiles()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(av.Files) != 2 {
		t.Fatalf("expected 2 files, got %d", len(av.Files))
	}
}

func TestTArtifactVersion_ReadFiles_WithSubdirs(t *testing.T) {
	tmpDir := t.TempDir()
	comm.WorkPath = tmpDir

	artId := "test-art-subdirs"
	artDir := filepath.Join(tmpDir, common.PathArtifacts, artId)
	subDir := filepath.Join(artDir, "subdir")
	if err := os.MkdirAll(subDir, 0750); err != nil {
		t.Fatalf("failed to create test dir: %v", err)
	}

	// Create file in root and subdirectory
	if err := os.WriteFile(filepath.Join(artDir, "root.txt"), []byte("root"), 0600); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}
	if err := os.WriteFile(filepath.Join(subDir, "nested.txt"), []byte("nested"), 0600); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	av := &TArtifactVersion{Id: artId}
	err := av.ReadFiles()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Should have root.txt and subdir (2 entries at top level)
	if len(av.Files) != 2 {
		t.Fatalf("expected 2 top-level entries, got %d", len(av.Files))
	}

	// Find the directory entry and verify it has children
	var dirEntry map[string]interface{}
	for _, f := range av.Files {
		if isDir, ok := f["dir"].(bool); ok && isDir {
			dirEntry = f
			break
		}
	}
	if dirEntry == nil {
		t.Fatal("expected to find a directory entry")
	}
	children, ok := dirEntry["child"].([]hbtp.Map)
	if !ok {
		t.Fatal("expected child to be []hbtp.Map")
	}
	if len(children) != 1 {
		t.Errorf("expected 1 child in subdir, got %d", len(children))
	}
}

func TestTArtifactVersion_ReadDir_NotExist(t *testing.T) {
	av := &TArtifactVersion{}
	_, err := av.readDir("/nonexistent/path/that/does/not/exist")
	if err == nil {
		t.Fatal("expected error for non-existent directory, got nil")
	}
}
