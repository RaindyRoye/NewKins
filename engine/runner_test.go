package engine

import (
	"container/list"
	"testing"

	"github.com/gokins/gokins/comm"
	"github.com/gokins/runner/runners"
)

// setupTestManager initializes Mgr.buildEgn for testing and returns a cleanup function.
func setupTestManager(t *testing.T) func() {
	t.Helper()
	origBuildEgn := Mgr.buildEgn
	Mgr.buildEgn = &BuildEngine{
		taskw: list.New(),
		tasks: make(map[string]*BuildTask),
	}
	return func() {
		Mgr.buildEgn = origBuildEgn
	}
}

// --- ServerInfo ---

func TestServerInfo(t *testing.T) {
	comm.Cfg.Server.Host = "https://example.com"
	comm.Cfg.Server.DownToken = "test-token-123"
	br := &baseRunner{}
	info, err := br.ServerInfo()
	if err != nil {
		t.Fatalf("ServerInfo() error: %v", err)
	}
	if info == nil {
		t.Fatal("ServerInfo() returned nil")
	}
	if info.WebHost != "https://example.com" {
		t.Errorf("WebHost = %q, want %q", info.WebHost, "https://example.com")
	}
	if info.DownToken != "test-token-123" {
		t.Errorf("DownToken = %q, want %q", info.DownToken, "test-token-123")
	}
}

// --- CheckCancel ---

func TestCheckCancel_BuildNotFound(t *testing.T) {
	cleanup := setupTestManager(t)
	defer cleanup()

	br := &baseRunner{}
	if !br.CheckCancel("nonexistent-build") {
		t.Error("CheckCancel should return true when build not found")
	}
}

// --- FindJobId ---

func TestFindJobId_EmptyBuildID(t *testing.T) {
	br := &baseRunner{}
	_, ok := br.FindJobId("", "stage", "step")
	if ok {
		t.Error("FindJobId should return false with empty buildID")
	}
}

func TestFindJobId_EmptyStageName(t *testing.T) {
	br := &baseRunner{}
	_, ok := br.FindJobId("build-1", "", "step")
	if ok {
		t.Error("FindJobId should return false with empty stage name")
	}
}

func TestFindJobId_EmptyStepName(t *testing.T) {
	br := &baseRunner{}
	_, ok := br.FindJobId("build-1", "stage", "")
	if ok {
		t.Error("FindJobId should return false with empty step name")
	}
}

func TestFindJobId_AllEmpty(t *testing.T) {
	br := &baseRunner{}
	_, ok := br.FindJobId("", "", "")
	if ok {
		t.Error("FindJobId should return false with all empty params")
	}
}

func TestFindJobId_BuildNotFound(t *testing.T) {
	cleanup := setupTestManager(t)
	defer cleanup()

	br := &baseRunner{}
	_, ok := br.FindJobId("nonexistent", "stage", "step")
	if ok {
		t.Error("FindJobId should return false when build not found")
	}
}

// --- ReadDir validation ---

func TestReadDir_EmptyBuildID(t *testing.T) {
	br := &baseRunner{}
	_, err := br.ReadDir(1, "", "some/path")
	if err == nil {
		t.Error("ReadDir should return error with empty buildID")
	}
}

func TestReadDir_EmptyPath(t *testing.T) {
	br := &baseRunner{}
	_, err := br.ReadDir(1, "build-1", "")
	if err == nil {
		t.Error("ReadDir should return error with empty path")
	}
}

func TestReadDir_BothEmpty(t *testing.T) {
	br := &baseRunner{}
	_, err := br.ReadDir(1, "", "")
	if err == nil {
		t.Error("ReadDir should return error with both empty")
	}
}

func TestReadDir_BuildNotFound(t *testing.T) {
	cleanup := setupTestManager(t)
	defer cleanup()

	br := &baseRunner{}
	_, err := br.ReadDir(1, "nonexistent", "path")
	if err == nil {
		t.Error("ReadDir should return error when build not found")
	}
}

// --- ReadFile validation ---

func TestReadFile_EmptyBuildID(t *testing.T) {
	br := &baseRunner{}
	_, _, err := br.ReadFile(1, "", "some/path", 0)
	if err == nil {
		t.Error("ReadFile should return error with empty buildID")
	}
}

func TestReadFile_EmptyPath(t *testing.T) {
	br := &baseRunner{}
	_, _, err := br.ReadFile(1, "build-1", "", 0)
	if err == nil {
		t.Error("ReadFile should return error with empty path")
	}
}

func TestReadFile_BuildNotFound(t *testing.T) {
	cleanup := setupTestManager(t)
	defer cleanup()

	br := &baseRunner{}
	_, _, err := br.ReadFile(1, "nonexistent", "path", 0)
	if err == nil {
		t.Error("ReadFile should return error when build not found")
	}
}

// --- GetEnv validation ---

func TestGetEnv_EmptyJobId(t *testing.T) {
	br := &baseRunner{}
	_, ok := br.GetEnv("build-1", "", "KEY")
	if ok {
		t.Error("GetEnv should return false with empty jobId")
	}
}

func TestGetEnv_EmptyKey(t *testing.T) {
	br := &baseRunner{}
	_, ok := br.GetEnv("build-1", "job-1", "")
	if ok {
		t.Error("GetEnv should return false with empty key")
	}
}

func TestGetEnv_BuildNotFound(t *testing.T) {
	cleanup := setupTestManager(t)
	defer cleanup()

	br := &baseRunner{}
	_, ok := br.GetEnv("nonexistent", "job-1", "KEY")
	if ok {
		t.Error("GetEnv should return false when build not found")
	}
}

// --- GenEnv validation ---

func TestGenEnv_EmptyJobId(t *testing.T) {
	br := &baseRunner{}
	err := br.GenEnv("build-1", "", nil)
	if err == nil {
		t.Error("GenEnv should return error with empty jobId")
	}
}

func TestGenEnv_NilEnv(t *testing.T) {
	br := &baseRunner{}
	err := br.GenEnv("build-1", "job-1", nil)
	if err == nil {
		t.Error("GenEnv should return error with nil env")
	}
}

func TestGenEnv_BuildNotFound(t *testing.T) {
	cleanup := setupTestManager(t)
	defer cleanup()

	br := &baseRunner{}
	err := br.GenEnv("nonexistent", "job-1", map[string]string{"KEY": "val"})
	if err == nil {
		t.Error("GenEnv should return error when build not found")
	}
}

// --- StatFile validation ---

func TestStatFile_EmptyJobId(t *testing.T) {
	br := &baseRunner{}
	_, err := br.StatFile(1, "build-1", "", "dir", "path")
	if err == nil {
		t.Error("StatFile should return error with empty jobId")
	}
}

func TestStatFile_EmptyPath(t *testing.T) {
	br := &baseRunner{}
	_, err := br.StatFile(1, "build-1", "job-1", "dir", "")
	if err == nil {
		t.Error("StatFile should return error with empty path")
	}
}

func TestStatFile_BuildNotFound(t *testing.T) {
	cleanup := setupTestManager(t)
	defer cleanup()

	br := &baseRunner{}
	_, err := br.StatFile(1, "nonexistent", "job-1", "dir", "path")
	if err == nil {
		t.Error("StatFile should return error when build not found")
	}
}

// --- UploadFile validation ---

func TestUploadFile_EmptyJobId(t *testing.T) {
	br := &baseRunner{}
	_, err := br.UploadFile(1, "build-1", "", "dir", "path", 0)
	if err == nil {
		t.Error("UploadFile should return error with empty jobId")
	}
}

func TestUploadFile_EmptyPath(t *testing.T) {
	br := &baseRunner{}
	_, err := br.UploadFile(1, "build-1", "job-1", "dir", "", 0)
	if err == nil {
		t.Error("UploadFile should return error with empty path")
	}
}

func TestUploadFile_BuildNotFound(t *testing.T) {
	cleanup := setupTestManager(t)
	defer cleanup()

	br := &baseRunner{}
	_, err := br.UploadFile(1, "nonexistent", "job-1", "dir", "path", 0)
	if err == nil {
		t.Error("UploadFile should return error when build not found")
	}
}

// --- FindArtVersionId validation ---

func TestFindArtVersionId_EmptyBuildID(t *testing.T) {
	br := &baseRunner{}
	_, err := br.FindArtVersionId("", "idnt", "name")
	if err == nil {
		t.Error("FindArtVersionId should return error with empty buildID")
	}
}

func TestFindArtVersionId_EmptyIdentifier(t *testing.T) {
	br := &baseRunner{}
	_, err := br.FindArtVersionId("build-1", "", "name")
	if err == nil {
		t.Error("FindArtVersionId should return error with empty identifier")
	}
}

func TestFindArtVersionId_EmptyName(t *testing.T) {
	br := &baseRunner{}
	_, err := br.FindArtVersionId("build-1", "idnt", "")
	if err == nil {
		t.Error("FindArtVersionId should return error with empty name")
	}
}

func TestFindArtVersionId_WhitespaceOnlyName(t *testing.T) {
	br := &baseRunner{}
	_, err := br.FindArtVersionId("build-1", "idnt", "   ")
	if err == nil {
		t.Error("FindArtVersionId should return error with whitespace-only name")
	}
}

// --- NewArtVersionId validation ---

func TestNewArtVersionId_EmptyBuildID(t *testing.T) {
	br := &baseRunner{}
	_, err := br.NewArtVersionId("", "idnt", "name")
	if err == nil {
		t.Error("NewArtVersionId should return error with empty buildID")
	}
}

func TestNewArtVersionId_EmptyIdentifier(t *testing.T) {
	br := &baseRunner{}
	_, err := br.NewArtVersionId("build-1", "", "name")
	if err == nil {
		t.Error("NewArtVersionId should return error with empty identifier")
	}
}

func TestNewArtVersionId_EmptyName(t *testing.T) {
	br := &baseRunner{}
	_, err := br.NewArtVersionId("build-1", "idnt", "")
	if err == nil {
		t.Error("NewArtVersionId should return error with empty name")
	}
}

func TestNewArtVersionId_NameWithVersionStripped(t *testing.T) {
	// NewArtVersionId strips @version from name, so "name@1.0" becomes "name"
	// but if only "@" is provided, name becomes empty
	br := &baseRunner{}
	_, err := br.NewArtVersionId("build-1", "idnt", "@1.0")
	if err == nil {
		t.Error("NewArtVersionId should return error when name part is empty after stripping version")
	}
}

// --- Update validation ---

func TestUpdate_BuildNotFound(t *testing.T) {
	cleanup := setupTestManager(t)
	defer cleanup()

	br := &baseRunner{}
	err := br.Update(&runners.UpdateJobInfo{BuildId: "nonexistent", JobId: "job-1"})
	if err == nil {
		t.Error("Update should return error when build not found")
	}
}

// --- UpdateCmd validation ---

func TestUpdateCmd_BuildNotFound(t *testing.T) {
	cleanup := setupTestManager(t)
	defer cleanup()

	br := &baseRunner{}
	err := br.UpdateCmd("nonexistent", "job-1", "cmd-1", 1, 0)
	if err == nil {
		t.Error("UpdateCmd should return error when build not found")
	}
}

// --- PushOutLine validation ---

func TestPushOutLine_BuildNotFound(t *testing.T) {
	cleanup := setupTestManager(t)
	defer cleanup()

	br := &baseRunner{}
	err := br.PushOutLine("nonexistent", "job-1", "cmd-1", "output line", false)
	if err == nil {
		t.Error("PushOutLine should return error when build not found")
	}
}
