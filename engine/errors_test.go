package engine

import (
	"container/list"
	"errors"
	"fmt"
	"strings"
	"testing"
)

// TestSentinelErrorValues verifies each sentinel error is non-nil, has a
// meaningful message, and is unique (no two sentinels share the same text).
func TestSentinelErrorValues(t *testing.T) {
	sentinels := []struct {
		name string
		err  error
	}{
		{"ErrBuildNotFound", ErrBuildNotFound},
		{"ErrJobNotFound", ErrJobNotFound},
		{"ErrCmdNotFound", ErrCmdNotFound},
		{"ErrInvalidFSType", ErrInvalidFSType},
		{"ErrEmptyParams", ErrEmptyParams},
		{"ErrArtifactoryNotFound", ErrArtifactoryNotFound},
		{"ErrArtifactNotFound", ErrArtifactNotFound},
		{"ErrPermissionDenied", ErrPermissionDenied},
		{"ErrPluginNotFound", ErrPluginNotFound},
	}

	seen := make(map[string]string) // message -> name
	for _, s := range sentinels {
		if s.err == nil {
			t.Fatalf("%s must not be nil", s.name)
		}
		msg := s.err.Error()
		if msg == "" {
			t.Fatalf("%s must have a non-empty message", s.name)
		}
		if prev, ok := seen[msg]; ok {
			t.Fatalf("%s and %s share the same message %q", s.name, prev, msg)
		}
		seen[msg] = s.name
	}
}

// TestSentinelErrorsIsChain ensures that wrapping a sentinel with
// fmt.Errorf("%w", ...) preserves errors.Is() identity.
func TestSentinelErrorsIsChain(t *testing.T) {
	sentinels := []error{
		ErrBuildNotFound,
		ErrJobNotFound,
		ErrCmdNotFound,
		ErrInvalidFSType,
		ErrEmptyParams,
		ErrArtifactoryNotFound,
		ErrArtifactNotFound,
		ErrPermissionDenied,
		ErrPluginNotFound,
	}

	for _, s := range sentinels {
		// Single wrap
		wrapped := fmt.Errorf("context: %w", s)
		if !errors.Is(wrapped, s) {
			t.Errorf("errors.Is(wrapped, %v) = false after single wrap", s)
		}

		// Double wrap
		doubleWrapped := fmt.Errorf("outer: %w", wrapped)
		if !errors.Is(doubleWrapped, s) {
			t.Errorf("errors.Is(doubleWrapped, %v) = false after double wrap", s)
		}
	}
}

// TestSentinelErrorsIsNegative verifies that unrelated errors do NOT match
// any sentinel via errors.Is().
func TestSentinelErrorsIsNegative(t *testing.T) {
	unrelated := errors.New("something completely different")

	sentinels := []error{
		ErrBuildNotFound,
		ErrJobNotFound,
		ErrCmdNotFound,
		ErrInvalidFSType,
		ErrEmptyParams,
		ErrArtifactoryNotFound,
		ErrArtifactNotFound,
		ErrPermissionDenied,
		ErrPluginNotFound,
	}

	for _, s := range sentinels {
		if errors.Is(unrelated, s) {
			t.Errorf("errors.Is(unrelated, %v) should be false", s)
		}
		wrapped := fmt.Errorf("wrap: %w", unrelated)
		if errors.Is(wrapped, s) {
			t.Errorf("errors.Is(wrapped unrelated, %v) should be false", s)
		}
	}
}

// TestBaseRunner_ErrorsWrapSentinels exercises each baseRunner method that
// returns errors and verifies the returned error wraps the expected sentinel.
// Methods that access the build engine require a minimal BuildEngine to be
// initialized in Mgr to avoid nil-pointer panics.
func TestBaseRunner_ErrorsWrapSentinels(t *testing.T) {
	r := &baseRunner{}

	// Initialize a minimal BuildEngine so Mgr.buildEgn.Get() doesn't panic.
	Mgr.buildEgn = &BuildEngine{
		taskw: list.New(),
		tasks: make(map[string]*BuildTask),
	}
	defer func() { Mgr.buildEgn = nil }()

	t.Run("ReadDir_empty_buildID", func(t *testing.T) {
		_, err := r.ReadDir(1, "", "path")
		if err == nil {
			t.Fatal("expected error")
		}
		if !errors.Is(err, ErrEmptyParams) {
			t.Errorf("expected ErrEmptyParams, got: %v", err)
		}
	})

	t.Run("ReadDir_empty_path", func(t *testing.T) {
		_, err := r.ReadDir(1, "build", "")
		if err == nil {
			t.Fatal("expected error")
		}
		if !errors.Is(err, ErrEmptyParams) {
			t.Errorf("expected ErrEmptyParams, got: %v", err)
		}
	})

	t.Run("ReadDir_unknown_build", func(t *testing.T) {
		_, err := r.ReadDir(1, "nonexistent-build", "path")
		if err == nil {
			t.Fatal("expected error")
		}
		if !errors.Is(err, ErrBuildNotFound) {
			t.Errorf("expected ErrBuildNotFound, got: %v", err)
		}
	})

	t.Run("ReadFile_empty_buildID", func(t *testing.T) {
		_, _, err := r.ReadFile(1, "", "path", 0)
		if err == nil {
			t.Fatal("expected error")
		}
		if !errors.Is(err, ErrEmptyParams) {
			t.Errorf("expected ErrEmptyParams, got: %v", err)
		}
	})

	t.Run("ReadFile_empty_path", func(t *testing.T) {
		_, _, err := r.ReadFile(1, "build", "", 0)
		if err == nil {
			t.Fatal("expected error")
		}
		if !errors.Is(err, ErrEmptyParams) {
			t.Errorf("expected ErrEmptyParams, got: %v", err)
		}
	})

	t.Run("ReadFile_unknown_build", func(t *testing.T) {
		_, _, err := r.ReadFile(1, "nonexistent-build", "path", 0)
		if err == nil {
			t.Fatal("expected error")
		}
		if !errors.Is(err, ErrBuildNotFound) {
			t.Errorf("expected ErrBuildNotFound, got: %v", err)
		}
	})

	t.Run("StatFile_empty_jobId", func(t *testing.T) {
		_, err := r.StatFile(1, "build", "", "dir", "path")
		if err == nil {
			t.Fatal("expected error")
		}
		if !errors.Is(err, ErrEmptyParams) {
			t.Errorf("expected ErrEmptyParams, got: %v", err)
		}
	})

	t.Run("StatFile_empty_path", func(t *testing.T) {
		_, err := r.StatFile(1, "build", "job", "dir", "")
		if err == nil {
			t.Fatal("expected error")
		}
		if !errors.Is(err, ErrEmptyParams) {
			t.Errorf("expected ErrEmptyParams, got: %v", err)
		}
	})

	t.Run("StatFile_unknown_build", func(t *testing.T) {
		_, err := r.StatFile(1, "nonexistent-build", "job", "dir", "path")
		if err == nil {
			t.Fatal("expected error")
		}
		if !errors.Is(err, ErrBuildNotFound) {
			t.Errorf("expected ErrBuildNotFound, got: %v", err)
		}
	})

	t.Run("UploadFile_empty_jobId", func(t *testing.T) {
		_, err := r.UploadFile(1, "build", "", "dir", "path", 0)
		if err == nil {
			t.Fatal("expected error")
		}
		if !errors.Is(err, ErrEmptyParams) {
			t.Errorf("expected ErrEmptyParams, got: %v", err)
		}
	})

	t.Run("UploadFile_empty_path", func(t *testing.T) {
		_, err := r.UploadFile(1, "build", "job", "dir", "", 0)
		if err == nil {
			t.Fatal("expected error")
		}
		if !errors.Is(err, ErrEmptyParams) {
			t.Errorf("expected ErrEmptyParams, got: %v", err)
		}
	})

	t.Run("UploadFile_unknown_build", func(t *testing.T) {
		_, err := r.UploadFile(1, "nonexistent-build", "job", "dir", "path", 0)
		if err == nil {
			t.Fatal("expected error")
		}
		if !errors.Is(err, ErrBuildNotFound) {
			t.Errorf("expected ErrBuildNotFound, got: %v", err)
		}
	})

	t.Run("FindArtVersionId_empty_buildID", func(t *testing.T) {
		_, err := r.FindArtVersionId("", "idnt", "name")
		if err == nil {
			t.Fatal("expected error")
		}
		if !errors.Is(err, ErrEmptyParams) {
			t.Errorf("expected ErrEmptyParams, got: %v", err)
		}
	})

	t.Run("FindArtVersionId_empty_idnt", func(t *testing.T) {
		_, err := r.FindArtVersionId("build", "", "name")
		if err == nil {
			t.Fatal("expected error")
		}
		if !errors.Is(err, ErrEmptyParams) {
			t.Errorf("expected ErrEmptyParams, got: %v", err)
		}
	})

	t.Run("FindArtVersionId_unknown_build", func(t *testing.T) {
		_, err := r.FindArtVersionId("nonexistent-build", "idnt", "name")
		if err == nil {
			t.Fatal("expected error")
		}
		if !errors.Is(err, ErrBuildNotFound) {
			t.Errorf("expected ErrBuildNotFound, got: %v", err)
		}
	})

	t.Run("NewArtVersionId_empty_buildID", func(t *testing.T) {
		_, err := r.NewArtVersionId("", "idnt", "name")
		if err == nil {
			t.Fatal("expected error")
		}
		if !errors.Is(err, ErrEmptyParams) {
			t.Errorf("expected ErrEmptyParams, got: %v", err)
		}
	})

	t.Run("NewArtVersionId_unknown_build", func(t *testing.T) {
		_, err := r.NewArtVersionId("nonexistent-build", "idnt", "name")
		if err == nil {
			t.Fatal("expected error")
		}
		if !errors.Is(err, ErrBuildNotFound) {
			t.Errorf("expected ErrBuildNotFound, got: %v", err)
		}
	})

	t.Run("GenEnv_empty_jobId", func(t *testing.T) {
		err := r.GenEnv("build", "", nil)
		if err == nil {
			t.Fatal("expected error")
		}
		if !errors.Is(err, ErrEmptyParams) {
			t.Errorf("expected ErrEmptyParams, got: %v", err)
		}
	})

	t.Run("GenEnv_unknown_build", func(t *testing.T) {
		env := make(map[string]string)
		err := r.GenEnv("nonexistent-build", "job", env)
		if err == nil {
			t.Fatal("expected error")
		}
		if !errors.Is(err, ErrBuildNotFound) {
			t.Errorf("expected ErrBuildNotFound, got: %v", err)
		}
	})
}

// TestBaseRunner_ErrorMessagesContainContext verifies that error messages
// include the relevant IDs so they remain useful in logs.
func TestBaseRunner_ErrorMessagesContainContext(t *testing.T) {
	r := &baseRunner{}

	// Initialize a minimal BuildEngine so Mgr.buildEgn.Get() doesn't panic.
	Mgr.buildEgn = &BuildEngine{
		taskw: list.New(),
		tasks: make(map[string]*BuildTask),
	}
	defer func() { Mgr.buildEgn = nil }()

	_, err := r.ReadDir(1, "", "somepath")
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "somepath") {
		t.Errorf("error should contain path 'somepath': %v", err)
	}

	_, _, err = r.ReadFile(1, "mybuild", "myfile", 0)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "mybuild") {
		t.Errorf("error should contain buildID 'mybuild': %v", err)
	}

	_, err = r.FindArtVersionId("b1", "ident1", "art@1.0")
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "b1") {
		t.Errorf("error should contain buildID 'b1': %v", err)
	}
}
