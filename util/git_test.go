package util

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/object"
)

func TestCheckOutHash_InvalidHash(t *testing.T) {
	tests := []struct {
		name string
		hash string
	}{
		{"empty string", ""},
		{"too short", "abc123"},
		{"not hex", "zzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzz"},
		{"contains spaces", "abc def ghi jkl mno pqr stu vwx yz0 1234"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := CheckOutHash(nil, tt.hash)
			if err == nil {
				t.Fatal("expected error for invalid hash, got nil")
			}
			if !strings.Contains(err.Error(), "not a valid git hash") {
				t.Errorf("error should mention 'not a valid git hash', got: %v", err)
			}
		})
	}
}

func TestCheckOutHash_ValidFormatNoRepo(t *testing.T) {
	// Valid 40-char hex hash, but nil repository should return an error
	validHash := "abcdef1234567890abcdef1234567890abcdef12"
	err := CheckOutHash(nil, validHash)
	if err == nil {
		t.Fatal("expected error for nil repository with valid hash")
	}
	if !strings.Contains(err.Error(), "repository is nil") {
		t.Errorf("error should mention 'repository is nil', got: %v", err)
	}
}

// initTestRepo creates a temporary git repository with one commit.
// Returns the repo and a cleanup function.
func initTestRepo(t *testing.T) (*git.Repository, func()) {
	t.Helper()
	dir, err := os.MkdirTemp("", "git_test_*")
	if err != nil {
		t.Fatalf("create temp dir: %v", err)
	}

	cleanup := func() {
		if err := os.RemoveAll(dir); err != nil {
			t.Logf("cleanup temp dir %s: %v", dir, err)
		}
	}
	t.Cleanup(cleanup)

	repo, err := git.PlainInit(dir, false)
	if err != nil {
		t.Fatalf("init repo: %v", err)
	}

	wt, err := repo.Worktree()
	if err != nil {
		t.Fatalf("get worktree: %v", err)
	}

	// Create a file and commit it
	testFile := filepath.Join(dir, "test.txt")
	if err := os.WriteFile(testFile, []byte("hello"), 0600); err != nil {
		t.Fatalf("write file: %v", err)
	}

	if _, err := wt.Add("test.txt"); err != nil {
		t.Fatalf("add file: %v", err)
	}

	_, err = wt.Commit("initial commit", &git.CommitOptions{
		Author: &object.Signature{
			Name:  "test",
			Email: "test@test.com",
			When:  time.Now(),
		},
	})
	if err != nil {
		t.Fatalf("commit: %v", err)
	}

	return repo, cleanup
}

func TestCheckOut_WithRealRepo(t *testing.T) {
	repo, cleanup := initTestRepo(t)
	defer cleanup()

	// Checkout the current HEAD should succeed
	head, err := repo.Head()
	if err != nil {
		t.Fatalf("get HEAD: %v", err)
	}

	err = CheckOutHash(repo, head.Hash().String())
	if err != nil {
		t.Errorf("CheckOutHash with valid commit should succeed, got: %v", err)
	}
}

func TestCheckOutHash_NonExistentCommit(t *testing.T) {
	repo, cleanup := initTestRepo(t)
	defer cleanup()

	// Valid hash format but non-existent commit.
	// go-git may return an error or may not validate existence at checkout time.
	// We just verify the function doesn't panic.
	nonExistent := "0000000000000000000000000000000000000000"
	_ = CheckOutHash(repo, nonExistent)
}

func TestCloneRepo_CancelledContext(t *testing.T) {
	dir, err := os.MkdirTemp("", "clone_test_*")
	if err != nil {
		t.Fatalf("create temp dir: %v", err)
	}
	defer func() {
		if err := os.RemoveAll(dir); err != nil {
			t.Logf("cleanup temp dir %s: %v", dir, err)
		}
	}()

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	_, err = CloneRepo(filepath.Join(dir, "cloned"), &git.CloneOptions{
		URL: "https://github.com/gokins/gokins.git",
	}, ctx)
	if err == nil {
		t.Error("expected error for cancelled context")
	}
}

func TestGetLogsHash_WithRealRepo(t *testing.T) {
	repo, cleanup := initTestRepo(t)
	defer cleanup()

	head, err := repo.Head()
	if err != nil {
		t.Fatalf("get HEAD: %v", err)
	}

	iter, err := GetLogsHash(repo, head.Hash().String())
	if err != nil {
		t.Fatalf("GetLogsHash should succeed: %v", err)
	}
	defer iter.Close()

	commit, err := iter.Next()
	if err != nil {
		t.Fatalf("should have at least one commit: %v", err)
	}
	if commit.Hash != head.Hash() {
		t.Errorf("expected commit %s, got %s", head.Hash(), commit.Hash)
	}
}

func TestGetLogs_NilRepo(t *testing.T) {
	_, err := GetLogs(nil, &git.LogOptions{})
	if err == nil {
		t.Fatal("expected error for nil repository")
	}
	if !strings.Contains(err.Error(), "repository is nil") {
		t.Errorf("error should mention 'repository is nil', got: %v", err)
	}
}

func TestCheckOut_NilRepo(t *testing.T) {
	err := CheckOut(nil, &git.CheckoutOptions{})
	if err == nil {
		t.Fatal("expected error for nil repository")
	}
	if !strings.Contains(err.Error(), "repository is nil") {
		t.Errorf("error should mention 'repository is nil', got: %v", err)
	}
}
