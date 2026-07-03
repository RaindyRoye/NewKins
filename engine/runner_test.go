package engine

import (
	"errors"
	"fmt"
	"testing"
)

// TestSentinelErrors verifies that sentinel errors can be detected with errors.Is
func TestSentinelErrors(t *testing.T) {
	tests := []struct {
		name     string
		sentinel error
		wrapped  error
	}{
		{"ErrBuildNotFound", ErrBuildNotFound, fmt.Errorf("update: %w: build \"123\"", ErrBuildNotFound)},
		{"ErrJobNotFound", ErrJobNotFound, fmt.Errorf("update: %w: job \"456\" in build \"123\"", ErrJobNotFound)},
		{"ErrCmdNotFound", ErrCmdNotFound, fmt.Errorf("updateCmd: %w: cmd \"789\" in job \"456\"", ErrCmdNotFound)},
		{"ErrInvalidInput", ErrInvalidInput, fmt.Errorf("readDir: %w: buildID and path must not be empty", ErrInvalidInput)},
		{"ErrArtifactoryNotFound", ErrArtifactoryNotFound, fmt.Errorf("findArtVersionId: %w: artifactory \"repo1\"", ErrArtifactoryNotFound)},
		{"ErrArtifactNotFound", ErrArtifactNotFound, fmt.Errorf("findArtVersionId: %w: package \"pkg1\"", ErrArtifactNotFound)},
		{"ErrPluginNotFound", ErrPluginNotFound, fmt.Errorf("%w: myplugin", ErrPluginNotFound)},
		{"ErrTriggerNotFound", ErrTriggerNotFound, fmt.Errorf("%w: trigger-123", ErrTriggerNotFound)},
		{"ErrNoJobAvailable", ErrNoJobAvailable, fmt.Errorf("%w for runner \"r1\" with plugins [p1] after 5s", ErrNoJobAvailable)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if !errors.Is(tt.wrapped, tt.sentinel) {
				t.Errorf("errors.Is(%v, %v) = false, want true", tt.wrapped, tt.sentinel)
			}
		})
	}
}

// TestSentinelErrorUniqueness verifies that sentinel errors are distinct
func TestSentinelErrorUniqueness(t *testing.T) {
	sentinels := []error{
		ErrBuildNotFound,
		ErrJobNotFound,
		ErrCmdNotFound,
		ErrInvalidInput,
		ErrArtifactoryNotFound,
		ErrArtifactNotFound,
		ErrPluginNotFound,
		ErrTriggerNotFound,
		ErrNoJobAvailable,
	}

	for i, s1 := range sentinels {
		for j, s2 := range sentinels {
			if i != j && errors.Is(s1, s2) {
				t.Errorf("sentinel %v should not match %v", s1, s2)
			}
		}
	}
}

func TestBaseRunner_ServerInfo(t *testing.T) {
	r := &baseRunner{}
	info, err := r.ServerInfo()
	if err != nil {
		t.Errorf("ServerInfo() error = %v", err)
	}
	if info == nil {
		t.Error("ServerInfo() returned nil")
	}
}

func TestBaseRunner_FindJobId_EmptyParams(t *testing.T) {
	r := &baseRunner{}
	tests := []struct {
		name    string
		buildID string
		stgNm   string
		stpNm   string
	}{
		{"empty buildID", "", "stage", "step"},
		{"empty stage", "build", "", "step"},
		{"empty step", "build", "stage", ""},
		{"all empty", "", "", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			id, ok := r.FindJobId(tt.buildID, tt.stgNm, tt.stpNm)
			if ok {
				t.Errorf("FindJobId() = (%s, true), want false", id)
			}
		})
	}
}

func TestBaseRunner_GetEnv_EmptyParams(t *testing.T) {
	r := &baseRunner{}
	tests := []struct {
		name    string
		buildID string
		jobId   string
		key     string
	}{
		{"empty jobId", "build", "", "key"},
		{"empty key", "build", "job", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			val, ok := r.GetEnv(tt.buildID, tt.jobId, tt.key)
			if ok {
				t.Errorf("GetEnv() = (%s, true), want false", val)
			}
		})
	}
}

func TestBaseRunner_ReadDir_EmptyParams(t *testing.T) {
	r := &baseRunner{}
	_, err := r.ReadDir(1, "", "path")
	if err == nil {
		t.Error("ReadDir() with empty buildID should error")
	}
	_, err = r.ReadDir(1, "build", "")
	if err == nil {
		t.Error("ReadDir() with empty path should error")
	}
}

func TestBaseRunner_ReadFile_EmptyParams(t *testing.T) {
	r := &baseRunner{}
	_, _, err := r.ReadFile(1, "", "path", 0)
	if err == nil {
		t.Error("ReadFile() with empty buildID should error")
	}
	_, _, err = r.ReadFile(1, "build", "", 0)
	if err == nil {
		t.Error("ReadFile() with empty path should error")
	}
}

func TestBaseRunner_StatFile_EmptyParams(t *testing.T) {
	r := &baseRunner{}
	_, err := r.StatFile(1, "build", "", "dir", "path")
	if err == nil {
		t.Error("StatFile() with empty jobId should error")
	}
	_, err = r.StatFile(1, "build", "job", "dir", "")
	if err == nil {
		t.Error("StatFile() with empty path should error")
	}
}

func TestBaseRunner_UploadFile_EmptyParams(t *testing.T) {
	r := &baseRunner{}
	_, err := r.UploadFile(1, "build", "", "dir", "path", 0)
	if err == nil {
		t.Error("UploadFile() with empty jobId should error")
	}
	_, err = r.UploadFile(1, "build", "job", "dir", "", 0)
	if err == nil {
		t.Error("UploadFile() with empty path should error")
	}
}

func TestBaseRunner_FindArtVersionId_EmptyParams(t *testing.T) {
	r := &baseRunner{}
	_, err := r.FindArtVersionId("", "idnt", "name")
	if err == nil {
		t.Error("FindArtVersionId() with empty buildID should error")
	}
}

func TestBaseRunner_NewArtVersionId_EmptyParams(t *testing.T) {
	r := &baseRunner{}
	_, err := r.NewArtVersionId("", "idnt", "name")
	if err == nil {
		t.Error("NewArtVersionId() with empty buildID should error")
	}
}

func TestBaseRunner_GenEnv_NilParams(t *testing.T) {
	r := &baseRunner{}
	err := r.GenEnv("build", "", nil)
	if err == nil {
		t.Error("GenEnv() with empty jobId should error")
	}
}

// Test that errors.Is works with wrapped errors
func TestErrorWrapping(t *testing.T) {
	baseErr := errors.New("base error")
	wrapped := errors.Join(errors.New("context"), baseErr)
	if !errors.Is(wrapped, baseErr) {
		t.Error("errors.Is should find base error in chain")
	}
}
