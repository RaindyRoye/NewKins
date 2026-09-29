package util

import (
	"context"
	"errors"
	"fmt"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
)

// ErrInvalidGitHash is returned when the provided hash string is not a valid git hash.
var ErrInvalidGitHash = errors.New("invalid git hash")

// ErrRepositoryNil is returned when a nil repository is passed to git operations.
var ErrRepositoryNil = errors.New("repository is nil")

func CloneRepo(path string, option *git.CloneOptions, ctx context.Context) (*git.Repository, error) {
	return git.PlainCloneContext(ctx,
		path,
		false,
		option,
	)
}

func CheckOutHash(repository *git.Repository, hash string) error {
	if !plumbing.IsHash(hash) {
		return fmt.Errorf("checkout: %w: %q is not a valid git hash", ErrInvalidGitHash, hash)
	}
	options := &git.CheckoutOptions{
		Force: true,
	}
	options.Hash = plumbing.NewHash(hash)
	return CheckOut(repository, options)
}

func CheckOut(repository *git.Repository, option *git.CheckoutOptions) error {
	if repository == nil {
		return fmt.Errorf("checkout: %w", ErrRepositoryNil)
	}
	worktree, err := repository.Worktree()
	if err != nil {
		return fmt.Errorf("get worktree: %w", err)
	}
	err = worktree.Checkout(option)
	if err != nil {
		return fmt.Errorf("checkout: %w", err)
	}
	return nil
}

func GetLogsHash(repository *git.Repository, hash string) (object.CommitIter, error) {
	h := plumbing.NewHash(hash)
	options := &git.LogOptions{From: h}
	return GetLogs(repository, options)
}

func GetLogs(repository *git.Repository, option *git.LogOptions) (object.CommitIter, error) {
	if repository == nil {
		return nil, fmt.Errorf("get logs: %w", ErrRepositoryNil)
	}
	return repository.Log(option)
}
