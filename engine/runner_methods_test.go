package engine

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/gokins/core/runtime"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestServerInfo verifies that ServerInfo returns a valid struct without error
func TestServerInfo(t *testing.T) {
	br := &baseRunner{}
	info, err := br.ServerInfo()
	require.NoError(t, err)
	require.NotNil(t, info)
}

// TestFindJobId_Validation verifies FindJobId returns false for invalid inputs
func TestFindJobId_Validation(t *testing.T) {
	br := &baseRunner{}
	cases := []struct {
		name    string
		buildID string
		stage   string
		step    string
	}{
		{"empty buildID", "", "stage", "step"},
		{"empty stage", "b1", "", "step"},
		{"empty step", "b1", "stage", ""},
		{"build not found", "nonexistent", "stage", "step"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, ok := br.FindJobId(tc.buildID, tc.stage, tc.step)
			assert.False(t, ok)
		})
	}
}

// TestFindJobId_StageAndStepNotFound verifies correct behavior for missing stage/step
func TestFindJobId_StageAndStepNotFound(t *testing.T) {
	buildEgn := &BuildEngine{tasks: make(map[string]*BuildTask)}
	task := &BuildTask{
		build: &runtime.Build{Id: "build123"},
		stages: map[string]*taskStage{
			"stage1": {
				stage: &runtime.Stage{Name: "stage1"},
				jobs: map[string]*jobSync{
					"job1": {step: &runtime.Step{Id: "job1", Name: "compile"}},
				},
			},
		},
	}
	buildEgn.tasks["build123"] = task
	Mgr.buildEgn = buildEgn
	defer func() { Mgr.buildEgn = nil }()

	br := &baseRunner{}
	// Stage not found
	_, ok := br.FindJobId("build123", "nonexistent-stage", "compile")
	assert.False(t, ok)
	// Step not found
	_, ok = br.FindJobId("build123", "stage1", "nonexistent-step")
	assert.False(t, ok)
	// Success
	jobID, ok := br.FindJobId("build123", "stage1", "compile")
	assert.True(t, ok)
	assert.Equal(t, "job1", jobID)
}

// TestReadDir_Validation verifies ReadDir returns appropriate errors for invalid inputs
func TestReadDir_Validation(t *testing.T) {
	br := &baseRunner{}
	// Empty buildID
	_, err := br.ReadDir(1, "", "/path")
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrEmptyParams))
	// Empty path
	_, err = br.ReadDir(1, "b1", "")
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrEmptyParams))
	// Build not found
	_, err = br.ReadDir(1, "nonexistent", "/path")
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrBuildNotFound))
}

// TestReadDir_InvalidFSType_WithRepoPath verifies ReadDir returns error for invalid FS type
// when repoPath is set (error propagates because build.repoPath is non-empty)
func TestReadDir_InvalidFSType_WithRepoPath(t *testing.T) {
	buildEgn := &BuildEngine{tasks: make(map[string]*BuildTask)}
	task := &BuildTask{
		build:     &runtime.Build{Id: "b1"},
		buildPath: "/tmp/b1",
		repoPaths: "/tmp/b1/repo",
		repoPath:  "https://example.com/repo.git",
	}
	buildEgn.tasks["b1"] = task
	Mgr.buildEgn = buildEgn
	defer func() { Mgr.buildEgn = nil }()

	br := &baseRunner{}
	_, err := br.ReadDir(99, "b1", "/path")
	require.Error(t, err)
}

// TestReadDir_InvalidFSType_NoRepoPath verifies ReadDir returns nil,nil when repoPath is empty
func TestReadDir_InvalidFSType_NoRepoPath(t *testing.T) {
	buildEgn := &BuildEngine{tasks: make(map[string]*BuildTask)}
	task := &BuildTask{
		build:     &runtime.Build{Id: "b2"},
		buildPath: "/tmp/b2",
		repoPaths: "/tmp/b2/repo",
		repoPath:  "",
	}
	buildEgn.tasks["b2"] = task
	Mgr.buildEgn = buildEgn
	defer func() { Mgr.buildEgn = nil }()

	br := &baseRunner{}
	entries, err := br.ReadDir(99, "b2", "/path")
	assert.NoError(t, err)
	assert.Nil(t, entries)
}

// TestReadDir_Success verifies ReadDir returns directory entries correctly
func TestReadDir_Success(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "readir-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "file1.txt"), []byte("c1"), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "file2.txt"), []byte("content2"), 0644))
	require.NoError(t, os.Mkdir(filepath.Join(tmpDir, "subdir"), 0755))

	buildEgn := &BuildEngine{tasks: make(map[string]*BuildTask)}
	buildEgn.tasks["b1"] = &BuildTask{
		build: &runtime.Build{Id: "b1"}, buildPath: tmpDir, repoPaths: tmpDir,
	}
	Mgr.buildEgn = buildEgn
	defer func() { Mgr.buildEgn = nil }()

	br := &baseRunner{}
	entries, err := br.ReadDir(1, "b1", ".")
	require.NoError(t, err)
	require.Len(t, entries, 3)

	names := make(map[string]bool)
	for _, e := range entries {
		names[e.Name] = true
		if e.Name == "subdir" {
			assert.True(t, e.IsDir)
		} else {
			assert.False(t, e.IsDir)
		}
	}
	assert.True(t, names["file1.txt"] && names["file2.txt"] && names["subdir"])
}

// TestReadFile_Validation verifies ReadFile returns appropriate errors for invalid inputs
func TestReadFile_Validation(t *testing.T) {
	br := &baseRunner{}
	// Empty buildID
	_, _, err := br.ReadFile(1, "", "/path", 0)
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrEmptyParams))
	// Empty path
	_, _, err = br.ReadFile(1, "b1", "", 0)
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrEmptyParams))
	// Build not found
	_, _, err = br.ReadFile(1, "nonexistent", "/path", 0)
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrBuildNotFound))
}

// TestReadFile_InvalidFSType verifies ReadFile returns ErrInvalidFSType for unknown FS codes
func TestReadFile_InvalidFSType(t *testing.T) {
	buildEgn := &BuildEngine{tasks: make(map[string]*BuildTask)}
	buildEgn.tasks["b1"] = &BuildTask{
		build: &runtime.Build{Id: "b1"}, buildPath: "/tmp/b1", repoPaths: "/tmp/b1/repo",
	}
	Mgr.buildEgn = buildEgn
	defer func() { Mgr.buildEgn = nil }()

	br := &baseRunner{}
	_, _, err := br.ReadFile(99, "b1", "/path", 0)
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrInvalidFSType))
}

// TestReadFile_FileNotFound verifies ReadFile returns error when file doesn't exist
func TestReadFile_FileNotFound(t *testing.T) {
	tmpDir := t.TempDir()
	buildEgn := &BuildEngine{tasks: make(map[string]*BuildTask)}
	buildEgn.tasks["b1"] = &BuildTask{
		build: &runtime.Build{Id: "b1"}, buildPath: tmpDir, repoPaths: tmpDir,
	}
	Mgr.buildEgn = buildEgn
	defer func() { Mgr.buildEgn = nil }()

	br := &baseRunner{}
	_, _, err := br.ReadFile(1, "b1", "nonexistent.txt", 0)
	require.Error(t, err)
}

// TestReadFile_Success_WithAndWithoutOffset verifies file read and seek behavior
func TestReadFile_Success_WithAndWithoutOffset(t *testing.T) {
	tmpDir := t.TempDir()
	content := "test file content\nwith multiple lines"
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "test.txt"), []byte(content), 0644))

	buildEgn := &BuildEngine{tasks: make(map[string]*BuildTask)}
	buildEgn.tasks["b1"] = &BuildTask{
		build: &runtime.Build{Id: "b1"}, buildPath: tmpDir, repoPaths: tmpDir,
	}
	Mgr.buildEgn = buildEgn
	defer func() { Mgr.buildEgn = nil }()

	br := &baseRunner{}

	// Read from start
	size, reader, err := br.ReadFile(1, "b1", "test.txt", 0)
	require.NoError(t, err)
	assert.Equal(t, int64(len(content)), size)
	buf := make([]byte, 1024)
	n, err := reader.Read(buf)
	require.NoError(t, err)
	assert.Equal(t, content, string(buf[:n]))
	reader.Close()

	// Read from offset 5 ("file content\nwith multiple lines")
	_, reader, err = br.ReadFile(1, "b1", "test.txt", 5)
	require.NoError(t, err)
	n, err = reader.Read(buf)
	require.NoError(t, err)
	assert.Equal(t, "file content\nwith multiple lines", string(buf[:n]))
	reader.Close()
}

// TestGetEnv_Validation verifies GetEnv returns false for invalid inputs
func TestGetEnv_Validation(t *testing.T) {
	br := &baseRunner{}
	cases := []struct {
		name  string
		jobID string
		key   string
	}{
		{"empty jobID", "", "key"},
		{"empty key", "job1", ""},
		{"build not found", "job1", "key"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, ok := br.GetEnv("build", tc.jobID, tc.key)
			assert.False(t, ok)
		})
	}
}

// TestGenEnv_Validation verifies GenEnv returns errors for invalid inputs
func TestGenEnv_Validation(t *testing.T) {
	br := &baseRunner{}
	// Empty jobID
	err := br.GenEnv("build", "", nil)
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrEmptyParams))
	// Nil env
	err = br.GenEnv("build", "job1", nil)
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrEmptyParams))
	// Build not found
	err = br.GenEnv("nonexistent", "job1", map[string]string{})
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrBuildNotFound))
}
