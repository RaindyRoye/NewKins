package giteaapi

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"

	"github.com/gokins/gokins/bean/thirdbean"
	"github.com/gokins/gokins/thirdapi"
	"github.com/sirupsen/logrus"
)

type RepositoryService struct {
	client *wrapper
}

// GetRepos
/*
  visibility : 公开(public)、私有(private)或者所有(all)，默认: 所有(all)
  affiliation : owner(授权用户拥有的仓库)、collaborator(授权用户为仓库成员)、organization_member(授权用户为仓库所在组织并有访问仓库权限)、
    enterprise_member(授权用户所在企业并有访问仓库权限)、admin(所有有权限的，包括所管理的组织中所有仓库、所管理的企业的所有仓库)。 可以用逗号分隔符组合。
    如: owner, organization_member 或 owner, collaborator, organization_member
  type : 筛选用户仓库: 其创建(owner)、个人(personal)、其为成员(member)、公开(public)、私有(private)，不能与 visibility 或 affiliation 参数一并使用，否则会报 422 错误
  sort : 排序方式: 创建时间(created)，更新时间(updated)，最后推送时间(pushed)，仓库所属与名称(full_name)。默认: full_name
  direction : 如果sort参数为full_name，用升序(asc)。否则降序(desc)
  q : 搜索关键字
  page : 当前的页码
  perPage : 每页的数量，最大为 100
*/
func (s *RepositoryService) GetRepos(ctx context.Context, accessToken, username, types, sort, direction string, page, perPage int) (*thirdapi.RepositoryPage, error) {
	parse, err := s.client.BaseURL.Parse(s.client.BaseURL.String() + fmt.Sprintf(ApiGiteaGetRepos, page, perPage))
	if err != nil {
		return nil, fmt.Errorf("gitea GetRepos: parse URL: %w", err)
	}
	logrus.Debugf("Gitea Api GetRepos url : %v", parse.String())
	req, err := http.NewRequestWithContext(ctx, "GET", parse.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("gitea GetRepos: create request: %w", err)
	}
	req.Header.Set("Authorization", "token "+accessToken)
	resp, err := s.client.HttpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("gitea GetRepos: HTTP request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("gitea api GetRepos failed (status %d)", resp.StatusCode)
	}
	repos, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("gitea GetRepos: read response body: %w", err)
	}
	var repoList []*thirdbean.ResultGiteaRepo
	err = json.Unmarshal(repos, &repoList)
	if err != nil {
		return nil, fmt.Errorf("gitea GetRepos: unmarshal response: %w", err)
	}
	lk := resp.Header.Get("x-total-count")
	var totalPages int64 = 1
	if lk != "" {
		totalCount, parseErr := strconv.ParseInt(lk, 10, 64)
		if parseErr != nil {
			return nil, fmt.Errorf("gitea GetRepos: parse total count: %w", parseErr)
		}
		totalPages = totalCount / int64(perPage)
		if totalCount%int64(perPage) > 0 {
			totalPages++
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
	parse, err := s.client.BaseURL.Parse(s.client.BaseURL.String() + fmt.Sprintf(ApiGiteaDeleteHooks, owner, repo, hookID))
	if err != nil {
		return fmt.Errorf("gitea DeleteHooks: parse URL: %w", err)
	}
	request, err := http.NewRequestWithContext(ctx, "DELETE", parse.String(), nil)
	if err != nil {
		return fmt.Errorf("gitea DeleteHooks: create request: %w", err)
	}
	request.Header.Set("Authorization", "token "+accessToken)
	logrus.Debugf("Gitea Api DeleteHooks url : %v", parse)
	resp, err := s.client.HttpClient.Do(request)
	if err != nil {
		return fmt.Errorf("gitea DeleteHooks: HTTP request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	all, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("gitea DeleteHooks: read response body: %w", err)
	}

	if resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("gitea api DeleteHooks failed (status %d): %s", resp.StatusCode, string(all))
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
	parse, err := s.client.BaseURL.Parse(s.client.BaseURL.String() + fmt.Sprintf(ApiGiteaCreateHooks, owner, repo))
	if err != nil {
		return nil, fmt.Errorf("gitea CreateWebHooks: parse URL: %w", err)
	}
	logrus.Debugf("Gitea Api CreateWebHooks url : %s", parse.String())
	m := map[string]any{}
	m["url"] = backURL
	m["content_type"] = "json"
	m["secret"] = password
	obj := map[string]any{}
	obj["config"] = m
	obj["type"] = "gitea"
	obj["active"] = true
	marshal, err := json.Marshal(obj)
	if err != nil {
		return nil, fmt.Errorf("gitea CreateWebHooks: marshal request body: %w", err)
	}
	logrus.Debugf("CreateWebHooks json %s", string(marshal))
	request, err := http.NewRequestWithContext(ctx, "POST", parse.String(), bytes.NewBuffer(marshal))
	if err != nil {
		return nil, fmt.Errorf("gitea CreateWebHooks: create request: %w", err)
	}
	request.Header.Set("Authorization", "token "+accessToken)
	request.Header.Set("Content-Type", "application/json")
	resp, err := s.client.HttpClient.Do(request)
	if err != nil {
		return nil, fmt.Errorf("gitea CreateWebHooks: HTTP request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	all, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("gitea CreateWebHooks: read response body: %w", err)
	}
	if resp.StatusCode != http.StatusCreated {
		return nil, fmt.Errorf("gitea api CreateWebHooks failed (status %d): %s", resp.StatusCode, string(all))
	}
	k := &thirdbean.ResultGetGiteaHook{}
	err = json.Unmarshal(all, k)
	if err != nil {
		return nil, fmt.Errorf("gitea CreateWebHooks: unmarshal response: %w", err)
	}
	return convertHook(k), nil
}

func (s *RepositoryService) GetRepoBranches(ctx context.Context, accessToken, owner, repo string) ([]*thirdapi.RepositoryBranch, error) {
	parse, err := s.client.BaseURL.Parse(s.client.BaseURL.String() + fmt.Sprintf(ApiGiteaGetRepoBranches, owner, repo))
	if err != nil {
		return nil, fmt.Errorf("gitea GetRepoBranches: parse URL: %w", err)
	}
	logrus.Debugf("Gitea Api GetRepoBranches url : %v", parse)
	req, err := http.NewRequestWithContext(ctx, "GET", parse.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("gitea GetRepoBranches: create request: %w", err)
	}
	req.Header.Set("Authorization", "token "+accessToken)
	resp, err := s.client.HttpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("gitea GetRepoBranches: HTTP request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	all, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("gitea GetRepoBranches: read response body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("gitea api GetRepoBranches failed (status %d): %s", resp.StatusCode, string(all))
	}
	var branchList []*thirdbean.ResultGiteaRepoBranch
	err = json.Unmarshal(all, &branchList)
	if err != nil {
		return nil, fmt.Errorf("gitea GetRepoBranches: unmarshal response: %w", err)
	}
	return convertBranchList(branchList), nil
}

func (s *RepositoryService) GetPullQuest(ctx context.Context, accessToken, owner, repo string, index int) ([]byte, error) {
	parse, err := s.client.BaseURL.Parse(s.client.BaseURL.String() + fmt.Sprintf(ApiGiteaGetPullRequest, owner, repo, index))
	if err != nil {
		return nil, fmt.Errorf("gitea GetPullQuest: parse URL: %w", err)
	}
	logrus.Debugf("Gitea Api GetPullQuest url : %v", parse)
	req, err := http.NewRequestWithContext(ctx, "GET", parse.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("gitea GetPullQuest: create request: %w", err)
	}
	req.Header.Set("Authorization", "token "+accessToken)
	resp, err := s.client.HttpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("gitea GetPullQuest: HTTP request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	bys, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("gitea GetPullQuest: read response body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("gitea api GetPullQuest failed (status %d): %s", resp.StatusCode, string(bys))
	}
	return bys, nil
}

func (s *RepositoryService) GetWebHooks(ctx context.Context, accessToken, owner, repo string, page, perPage int) ([]*thirdapi.RepositoryHook, error) {
	parse, err := s.client.BaseURL.Parse(s.client.BaseURL.String() + fmt.Sprintf(ApiGiteaGetHooks, owner, repo, page, perPage))
	if err != nil {
		return nil, fmt.Errorf("gitea GetWebHooks: parse URL: %w", err)
	}
	logrus.Debugf("Gitea Api GetWebHooks url : %v", parse)
	req, err := http.NewRequestWithContext(ctx, "GET", parse.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("gitea GetWebHooks: create request: %w", err)
	}
	req.Header.Set("Authorization", "token "+accessToken)
	resp, err := s.client.HttpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("gitea GetWebHooks: HTTP request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	all, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("gitea GetWebHooks: read response body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("gitea api GetWebHooks failed (status %d): %s", resp.StatusCode, string(all))
	}

	hs := make([]*thirdbean.ResultGetGiteaHook, 0)
	err = json.Unmarshal(all, &hs)
	if err != nil {
		return nil, fmt.Errorf("gitea GetWebHooks: unmarshal response: %w", err)
	}
	return convertHookList(hs), nil
}

func convertRepositoryList(ls []*thirdbean.ResultGiteaRepo) []*thirdapi.Repository {
	repos := make([]*thirdapi.Repository, 0, len(ls))
	for _, v := range ls {
		repos = append(repos, convertRepository(v))
	}
	return repos
}

func convertRepository(from *thirdbean.ResultGiteaRepo) *thirdapi.Repository {
	return &thirdapi.Repository{
		Id:        strconv.Itoa(from.Id),
		Owner:     from.Owner.Login,
		Name:      from.Name,
		Path:      from.Name,
		Namespace: from.Owner.Login,
		FullName:  from.FullName,
		HtmlURL:   from.HtmlUrl,
		RepoType:  "gitea",
	}
}
func convertBranchList(ls []*thirdbean.ResultGiteaRepoBranch) []*thirdapi.RepositoryBranch {
	repos := make([]*thirdapi.RepositoryBranch, 0, len(ls))
	for _, v := range ls {
		repos = append(repos, convertBranch(v))
	}
	return repos
}

func convertBranch(from *thirdbean.ResultGiteaRepoBranch) *thirdapi.RepositoryBranch {
	return &thirdapi.RepositoryBranch{
		Name: from.Name,
	}
}
func convertHookList(ls []*thirdbean.ResultGetGiteaHook) []*thirdapi.RepositoryHook {
	repos := make([]*thirdapi.RepositoryHook, 0, len(ls))
	for _, v := range ls {
		repos = append(repos, convertHook(v))
	}
	return repos
}

func convertHook(from *thirdbean.ResultGetGiteaHook) *thirdapi.RepositoryHook {
	return &thirdapi.RepositoryHook{
		Id:        from.Id,
		Url:       from.Config.Url,
		CreatedAt: from.CreatedAt,
	}
}
