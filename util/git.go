package util

import (
	"context"
	"fmt"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
)

func CloneRepo(path string, option *git.CloneOptions, ctx context.Context) (*git.Repository, error) {
	return git.PlainCloneContext(ctx,
		path,
		false,
		option,
	)
}

func CheckOutHash(repository *git.Repository, hash string) error {
	if !plumbing.IsHash(hash) {
		return fmt.Errorf("checkout: %q is not a valid git hash", hash)
	}
	options := &git.CheckoutOptions{
		Force: true,
	}
	options.Hash = plumbing.NewHash(hash)
	return CheckOut(repository, options)
}

func CheckOut(repository *git.Repository, option *git.CheckoutOptions) error {
	if repository == nil {
		return fmt.Errorf("checkout: repository is nil")
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
		return nil, fmt.Errorf("get logs: repository is nil")
	}
	iter, err := repository.Log(option)
	if err != nil {
		return nil, fmt.Errorf("get logs: %w", err)
	}
	return iter, nil
}
