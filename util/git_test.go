package util

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
)

// setupTestRepo creates a bare repo (origin) and a working clone (clone) with
// at least two commits on main, returning both paths plus the commit hashes.
// The caller must invoke the returned cleanup function (usually via defer).
func setupTestRepo(t *testing.T) (cloneDir string, hashes []plumbing.Hash, cleanup func()) {
	t.Helper()

	root := t.TempDir()
	originDir := filepath.Join(root, "origin")
	cloneDir = filepath.Join(root, "clone")

	// 1. Create a bare "origin" repo.
	origin, err := git.PlainInit(originDir, true)
	if err != nil {
		t.Fatalf("init origin: %v", err)
	}

	// 2. Create a separate worktree repo, commit two files, and push to origin.
	workDir := filepath.Join(root, "work")
	work, err := git.PlainInit(workDir, false)
	if err != nil {
		t.Fatalf("init work: %v", err)
	}
	// Point work's "origin" remote at the bare repo.
	if _, err := work.CreateRemote(&config.RemoteConfig{
		Name: "origin",
		URLs: []string{originDir},
	}); err != nil {
		t.Fatalf("create remote: %v", err)
	}

	writeFile := func(name, body string) {
		t.Helper()
		p := filepath.Join(workDir, name)
		if err := os.WriteFile(p, []byte(body), 0o600); err != nil {
			t.Fatalf("write %s: %v", name, err)
		}
	}

	wt, err := work.Worktree()
	if err != nil {
		t.Fatalf("worktree: %v", err)
	}
	sig := &object.Signature{Name: "t", Email: "t@t", When: time.Now()}

	writeFile("a.txt", "first")
	if _, err := wt.Add("a.txt"); err != nil {
		t.Fatalf("add a: %v", err)
	}
	h1, err := wt.Commit("first commit", &git.CommitOptions{Author: sig, Committer: sig})
	if err != nil {
		t.Fatalf("commit 1: %v", err)
	}
	hashes = append(hashes, h1)

	writeFile("b.txt", "second")
	if _, err := wt.Add("b.txt"); err != nil {
		t.Fatalf("add b: %v", err)
	}
	h2, err := wt.Commit("second commit", &git.CommitOptions{Author: sig, Committer: sig})
	if err != nil {
		t.Fatalf("commit 2: %v", err)
	}
	hashes = append(hashes, h2)

	if err := work.Push(&git.PushOptions{RemoteName: "origin"}); err != nil {
		t.Fatalf("push: %v", err)
	}

	// 3. Clone origin into cloneDir so we have a real working copy to exercise.
	if _, err := git.PlainClone(cloneDir, false, &git.CloneOptions{URL: originDir}); err != nil {
		t.Fatalf("clone: %v", err)
	}

	// Silence the unused-origin warning (we still need it to exist on disk).
	_ = origin

	return cloneDir, hashes, func() { /* TempDir auto-cleans */ }
}

// TestCheckOutHash_Valid exercises the happy path of CheckOutHash against a
// real git repository.
func TestCheckOutHash_Valid(t *testing.T) {
	cloneDir, hashes, cleanup := setupTestRepo(t)
	defer cleanup()

	repo, err := git.PlainOpen(cloneDir)
	if err != nil {
		t.Fatalf("open: %v", err)
	}

	// Checkout the FIRST commit (h1) so we can verify HEAD actually moves.
	if err := CheckOutHash(repo, hashes[0].String()); err != nil {
		t.Fatalf("CheckOutHash: %v", err)
	}
	head, err := repo.Head()
	if err != nil {
		t.Fatalf("head: %v", err)
	}
	if head.Hash() != hashes[0] {
		t.Fatalf("expected HEAD=%s, got %s", hashes[0], head.Hash())
	}
}

// TestCheckOutHash_InvalidHash reuses the existing unit-level assertion but
// now lives alongside the happy path for symmetry.
func TestCheckOutHash_InvalidHash(t *testing.T) {
	err := CheckOutHash(nil, "not-a-valid-hash")
	if err == nil {
		t.Fatal("expected error for invalid hash, got nil")
	}
	if !strings.Contains(err.Error(), "not a valid git hash") {
		t.Fatalf("expected 'not a valid git hash', got %q", err.Error())
	}
}

// TestCheckOutHash_NilRepo verifies the nil guard.
func TestCheckOutHash_NilRepo(t *testing.T) {
	err := CheckOutHash(nil, "0123456789abcdef0123456789abcdef01234567")
	if err == nil {
		t.Fatal("expected error for nil repo")
	}
	if !strings.Contains(err.Error(), "repository is nil") {
		t.Fatalf("expected 'repository is nil', got %q", err.Error())
	}
}

// TestCheckOut_Valid covers CheckOut using a real repository.
func TestCheckOut_Valid(t *testing.T) {
	cloneDir, hashes, cleanup := setupTestRepo(t)
	defer cleanup()

	repo, err := git.PlainOpen(cloneDir)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	if err := CheckOut(repo, &git.CheckoutOptions{
		Hash:  hashes[0],
		Force: true,
	}); err != nil {
		t.Fatalf("CheckOut: %v", err)
	}
	head, err := repo.Head()
	if err != nil {
		t.Fatalf("head: %v", err)
	}
	if head.Hash() != hashes[0] {
		t.Fatalf("expected HEAD=%s, got %s", hashes[0], head.Hash())
	}
}

// TestCheckOut_NilRepository verifies the nil guard.
func TestCheckOut_NilRepository(t *testing.T) {
	err := CheckOut(nil, nil)
	if err == nil {
		t.Fatal("expected error for nil repository, got nil")
	}
	if err.Error() != "checkout: repository is nil" {
		t.Fatalf("expected 'checkout: repository is nil', got %q", err.Error())
	}
}

// TestGetLogsHash covers the helper that builds LogOptions from a hash string.
func TestGetLogsHash(t *testing.T) {
	cloneDir, hashes, cleanup := setupTestRepo(t)
	defer cleanup()

	repo, err := git.PlainOpen(cloneDir)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	iter, err := GetLogsHash(repo, hashes[1].String())
	if err != nil {
		t.Fatalf("GetLogsHash: %v", err)
	}
	defer iter.Close()

	// Walk commits - we expect to see the target hash among them.
	seen := false
	_ = iter.ForEach(func(c *object.Commit) error {
		if c.Hash == hashes[1] {
			seen = true
		}
		return nil
	})
	if !seen {
		t.Fatalf("expected to find commit %s in log", hashes[1])
	}
}

// TestGetLogs_NilRepository verifies the nil guard.
func TestGetLogs_NilRepository(t *testing.T) {
	_, err := GetLogs(nil, nil)
	if err == nil {
		t.Fatal("expected error for nil repository, got nil")
	}
	if err.Error() != "get logs: repository is nil" {
		t.Fatalf("expected 'get logs: repository is nil', got %q", err.Error())
	}
}

// TestGetLogs_Valid exercises the real log iteration.
func TestGetLogs_Valid(t *testing.T) {
	cloneDir, hashes, cleanup := setupTestRepo(t)
	defer cleanup()

	repo, err := git.PlainOpen(cloneDir)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	iter, err := GetLogs(repo, &git.LogOptions{From: hashes[1]})
	if err != nil {
		t.Fatalf("GetLogs: %v", err)
	}
	defer iter.Close()

	count := 0
	_ = iter.ForEach(func(c *object.Commit) error {
		count++
		return nil
	})
	if count < 2 {
		t.Fatalf("expected at least 2 commits in log, got %d", count)
	}
}

// TestCloneRepo_Valid exercises CloneRepo against a real origin.
func TestCloneRepo_Valid(t *testing.T) {
	cloneDir, _, cleanup := setupTestRepo(t)
	defer cleanup()

	target := filepath.Join(t.TempDir(), "clone2")
	repo, err := CloneRepo(target, &git.CloneOptions{URL: cloneDir}, context.Background())
	if err != nil {
		t.Fatalf("CloneRepo: %v", err)
	}
	if repo == nil {
		t.Fatal("expected non-nil repo")
	}
	// Verify we can open HEAD - proves the clone is real.
	if _, err := repo.Head(); err != nil {
		t.Fatalf("head: %v", err)
	}
}

// TestCloneRepo_CancelledCtx ensures context cancellation is honored.
func TestCloneRepo_CancelledCtx(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately

	_, err := CloneRepo(filepath.Join(t.TempDir(), "dead"), &git.CloneOptions{URL: "/nonexistent"}, ctx)
	if err == nil {
		t.Fatal("expected error for canceled context")
	}
	if !errors.Is(err, context.Canceled) {
		// go-git wraps context errors, but they should still be unwrappable.
		if !strings.Contains(err.Error(), "context canceled") {
			t.Fatalf("expected context.Canceled in error chain, got: %v", err)
		}
	}
}
