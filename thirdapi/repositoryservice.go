package thirdapi

import "context"

type (
	RepositoryService interface {
		GetRepos(ctx context.Context, accessToken, username, types, sort, direction string, page, per_page int) (*RepositoryPage, error)

		DeleteHooks(ctx context.Context, accessToken, owner, repo, hookId string) error

		CreateWebHooks(ctx context.Context, accessToken, owner, repo, backUrl, password string) (*RepositoryHook, error)

		GetRepoBranches(ctx context.Context, accessToken, owner, repo string) ([]*RepositoryBranch, error)

		GetWebHooks(ctx context.Context, accessToken, owner, repo string, page, per_page int) ([]*RepositoryHook, error)
	}
)
