package gitlabapi

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"

	"github.com/gokins/gokins/bean/thirdbean"
	"github.com/gokins/gokins/thirdapi"
	"github.com/sirupsen/logrus"
)

type RepositoryService struct {
	client *wrapper
}

func (s *RepositoryService) GetRepos(ctx context.Context, accessToken, username, types, sort, direction string, page, perPage int) (*thirdapi.RepositoryPage, error) {
	parse, err := s.client.BaseURL.Parse(s.client.BaseURL.String() + fmt.Sprintf(ApiGitlabGetRepos, username, types, page, perPage))
	if err != nil {
		return nil, fmt.Errorf("gitlab GetRepos: parse URL: %w", err)
	}
	logrus.Debugf("Gitlab Api GetRepos url : %v", parse.String())
	req, err := http.NewRequestWithContext(ctx, "GET", parse.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("gitlab GetRepos: create request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	resp, err := s.client.HttpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("gitlab GetRepos: HTTP request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, thirdapi.NewAPIError("gitlab", "GetRepos", resp.StatusCode, string(body))
	}
	repos, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("gitlab GetRepos: read response body: %w", err)
	}
	var repoList []*thirdbean.ResultGitlabRepo
	err = json.Unmarshal(repos, &repoList)
	if err != nil {
		return nil, fmt.Errorf("gitlab GetRepos: unmarshal response: %w", err)
	}
	tp := resp.Header.Get("X-Total-Pages")
	var totalPages int64 = 0
	if tp != "" {
		totalPages, err = strconv.ParseInt(tp, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("gitlab GetRepos: parse total pages: %w", err)
		}
	}
	list := convertRepositoryList(repoList)
	rp := &thirdapi.RepositoryPage{
		TotalPages: totalPages,
		Ropes:      list,
	}
	return rp, nil
}

func (s *RepositoryService) DeleteHooks(ctx context.Context, accessToken, owner, repo, hookID string) error {
	escape := url.QueryEscape(owner + "/" + repo)
	parse, err := s.client.BaseURL.Parse(s.client.BaseURL.String() + fmt.Sprintf(ApiGitlabDeleteHooks, escape, hookID))
	if err != nil {
		return fmt.Errorf("gitlab DeleteHooks: parse URL: %w", err)
	}
	request, err := http.NewRequestWithContext(ctx, "DELETE", parse.String(), nil)
	if err != nil {
		return fmt.Errorf("gitlab DeleteHooks: create request: %w", err)
	}
	request.Header.Set("Authorization", "Bearer "+accessToken)
	logrus.Debugf("Gitlab Api DeleteHooks url : %v", parse)
	resp, err := s.client.HttpClient.Do(request)
	if err != nil {
		return fmt.Errorf("gitlab DeleteHooks: HTTP request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	all, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("gitlab DeleteHooks: read response body: %w", err)
	}

	if resp.StatusCode != http.StatusNoContent {
		return thirdapi.NewAPIError("gitlab", "DeleteHooks", resp.StatusCode, string(all))
	}
	return nil
}

// CreateWebHooks
/*
  owner : 仓库所属空间地址(企业、组织或个人的地址path)
  repo : 仓库路径(path)
  backURL : 回调地址
  password : webhook 密钥
*/
func (s *RepositoryService) CreateWebHooks(ctx context.Context, accessToken, owner, repo, backURL, password string) (*thirdapi.RepositoryHook, error) {
	escape := url.QueryEscape(owner + "/" + repo)
	parse, err := s.client.BaseURL.Parse(s.client.BaseURL.String() + fmt.Sprintf(ApiGitlabCreateHooks, escape))
	if err != nil {
		return nil, fmt.Errorf("gitlab CreateWebHooks: parse URL: %w", err)
	}
	logrus.Debugf("Gitlab Api CreateWebHooks url : %v", parse)
	m := map[string]any{}
	m["url"] = backURL
	m["token"] = password
	logrus.Infof("gitlab CreateWebHooks backURL : %s", backURL)
	marshal, err := json.Marshal(m)
	if err != nil {
		return nil, fmt.Errorf("gitlab CreateWebHooks: marshal request body: %w", err)
	}
	logrus.Infof("gitlab CreateWebHooks json : %s", string(marshal))
	request, err := http.NewRequestWithContext(ctx, "POST", parse.String(), bytes.NewBuffer(marshal))
	if err != nil {
		return nil, fmt.Errorf("gitlab CreateWebHooks: create request: %w", err)
	}
	request.Header.Set("Authorization", "Bearer "+accessToken)
	request.Header.Set("Content-Type", "application/json")
	resp, err := s.client.HttpClient.Do(request)
	if err != nil {
		return nil, fmt.Errorf("gitlab CreateWebHooks: HTTP request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	all, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("gitlab CreateWebHooks: read response body: %w", err)
	}

	if resp.StatusCode != http.StatusCreated {
		return nil, thirdapi.NewAPIError("gitlab", "CreateWebHooks", resp.StatusCode, string(all))
	}
	k := &thirdbean.ResultGetGitlabHook{}
	err = json.Unmarshal(all, k)
	if err != nil {
		return nil, fmt.Errorf("gitlab CreateWebHooks: unmarshal response: %w", err)
	}
	return convertHook(k), nil
}

func (s *RepositoryService) GetRepoBranches(ctx context.Context, accessToken, owner, repo string) ([]*thirdapi.RepositoryBranch, error) {
	escape := url.QueryEscape(owner + "/" + repo)
	parse, err := s.client.BaseURL.Parse(s.client.BaseURL.String() + fmt.Sprintf(ApiGitlabGetRepoBranches, escape))
	if err != nil {
		return nil, fmt.Errorf("gitlab GetRepoBranches: parse URL: %w", err)
	}
	logrus.Debugf("Gitlab Api GetRepoBranches url : %v", parse)
	req, err := http.NewRequestWithContext(ctx, "GET", parse.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("gitlab GetRepoBranches: create request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	resp, err := s.client.HttpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("gitlab GetRepoBranches: HTTP request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	all, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("gitlab GetRepoBranches: read response body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, thirdapi.NewAPIError("gitlab", "GetRepoBranches", resp.StatusCode, string(all))
	}
	var branchList []*thirdbean.ResultGitlabRepoBranch
	err = json.Unmarshal(all, &branchList)
	if err != nil {
		return nil, fmt.Errorf("gitlab GetRepoBranches: unmarshal response: %w", err)
	}
	return convertBranchList(branchList), nil
}

func (s *RepositoryService) GetWebHooks(ctx context.Context, accessToken, owner, repo string, page, perPage int) ([]*thirdapi.RepositoryHook, error) {
	escape := url.QueryEscape(owner + "/" + repo)
	parse, err := s.client.BaseURL.Parse(s.client.BaseURL.String() + fmt.Sprintf(ApiGitlabGetHooks, escape, page, perPage))
	if err != nil {
		return nil, fmt.Errorf("gitlab GetWebHooks: parse URL: %w", err)
	}
	logrus.Debugf("Gitlab Api GetWebHooks url : %v", parse)
	req, err := http.NewRequestWithContext(ctx, "GET", parse.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("gitlab GetWebHooks: create request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	resp, err := s.client.HttpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("gitlab GetWebHooks: HTTP request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	all, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("gitlab GetWebHooks: read response body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, thirdapi.NewAPIError("gitlab", "GetWebHooks", resp.StatusCode, string(all))
	}

	hs := make([]*thirdbean.ResultGetGitlabHook, 0)
	err = json.Unmarshal(all, &hs)
	if err != nil {
		return nil, fmt.Errorf("gitlab GetWebHooks: unmarshal response: %w", err)
	}
	return convertHookList(hs), nil
}

func convertRepositoryList(ls []*thirdbean.ResultGitlabRepo) []*thirdapi.Repository {
	repos := make([]*thirdapi.Repository, 0, len(ls))
	for _, v := range ls {
		repos = append(repos, convertRepository(v))
	}
	return repos
}

func convertRepository(from *thirdbean.ResultGitlabRepo) *thirdapi.Repository {
	return &thirdapi.Repository{
		Id:        strconv.Itoa(from.Id),
		Owner:     from.Owner.Username,
		Name:      from.Name,
		Path:      from.Path,
		Namespace: from.Namespace.Path,
		FullName:  from.PathWithNamespace,
		HtmlURL:   from.WebUrl,
		RepoType:  "gitlab",
	}
}
func convertBranchList(ls []*thirdbean.ResultGitlabRepoBranch) []*thirdapi.RepositoryBranch {
	repos := make([]*thirdapi.RepositoryBranch, 0, len(ls))
	for _, v := range ls {
		repos = append(repos, convertBranch(v))
	}
	return repos
}

func convertBranch(from *thirdbean.ResultGitlabRepoBranch) *thirdapi.RepositoryBranch {
	return &thirdapi.RepositoryBranch{
		Name: from.Name,
	}
}
func convertHookList(ls []*thirdbean.ResultGetGitlabHook) []*thirdapi.RepositoryHook {
	repos := make([]*thirdapi.RepositoryHook, 0, len(ls))
	for _, v := range ls {
		repos = append(repos, convertHook(v))
	}
	return repos
}

func convertHook(from *thirdbean.ResultGetGitlabHook) *thirdapi.RepositoryHook {
	return &thirdapi.RepositoryHook{
		Id:        from.Id,
		Url:       from.Url,
		CreatedAt: from.CreatedAt,
	}
}
