package giteeapi

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"

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
	parse, err := s.client.BaseURL.Parse(s.client.BaseURL.String() + fmt.Sprintf(ApiGiteeGetRepos, accessToken, types, sort, direction, page, perPage))
	if err != nil {
		return nil, fmt.Errorf("gitee GetRepos: parse URL: %w", err)
	}
	logrus.Debugf("Gitee Api GetRepos url : %v", parse.String())
	req, err := http.NewRequestWithContext(ctx, "GET", parse.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("gitee GetRepos: create request: %w", err)
	}
	resp, err := s.client.HttpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("gitee GetRepos: HTTP request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("gitee api GetRepos failed (status %d)", resp.StatusCode)
	}
	repos, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("gitee GetRepos: read response body: %w", err)
	}
	var repoList []*thirdbean.ResultGiteeRepo
	err = json.Unmarshal(repos, &repoList)
	if err != nil {
		return nil, fmt.Errorf("gitee GetRepos: unmarshal response: %w", err)
	}
	tp := resp.Header.Get("total_page")
	var totalPages int64 = 0
	if tp != "" {
		totalPages, err = strconv.ParseInt(tp, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("gitee GetRepos: parse total pages: %w", err)
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
	parse, err := s.client.BaseURL.Parse(s.client.BaseURL.String() + fmt.Sprintf(ApiGiteeDeleteHooks, owner, repo, hookID, accessToken))
	if err != nil {
		return fmt.Errorf("gitee DeleteHooks: parse URL: %w", err)
	}
	request, err := http.NewRequestWithContext(ctx, "DELETE", parse.String(), nil)
	if err != nil {
		return fmt.Errorf("gitee DeleteHooks: create request: %w", err)
	}
	logrus.Debugf("Gitee Api DeleteHooks url : %v", parse)
	resp, err := s.client.HttpClient.Do(request)
	if err != nil {
		return fmt.Errorf("gitee DeleteHooks: HTTP request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	all, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("gitee DeleteHooks: read response body: %w", err)
	}

	if resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("gitee api DeleteHooks failed (status %d): %s", resp.StatusCode, string(all))
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
	values := url.Values{}
	values.Add("access_token", accessToken)
	values.Add("url", backURL)
	values.Add("content", "0")
	values.Add("password", password)
	values.Add("push_events", "true")
	parse, err := s.client.BaseURL.Parse(s.client.BaseURL.String() + fmt.Sprintf(ApiGiteeCreateHooks, owner, repo))
	if err != nil {
		return nil, fmt.Errorf("gitee CreateWebHooks: parse URL: %w", err)
	}
	logrus.Debugf("Gitee Api CreateWebHooks url : %v", parse)
	request, err := http.NewRequestWithContext(ctx, "POST", parse.String(), strings.NewReader(values.Encode()))
	if err != nil {
		return nil, fmt.Errorf("gitee CreateWebHooks: create request: %w", err)
	}
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := s.client.HttpClient.Do(request)
	if err != nil {
		return nil, fmt.Errorf("gitee CreateWebHooks: HTTP request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	all, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("gitee CreateWebHooks: read response body: %w", err)
	}

	if resp.StatusCode != http.StatusCreated {
		return nil, fmt.Errorf("gitee api CreateWebHooks failed (status %d): %s", resp.StatusCode, string(all))
	}
	k := &thirdbean.ResultGetGiteeHook{}
	err = json.Unmarshal(all, k)
	if err != nil {
		return nil, fmt.Errorf("gitee CreateWebHooks: unmarshal response: %w", err)
	}
	return convertHook(k), nil
}

func (s *RepositoryService) GetRepoBranches(ctx context.Context, accessToken, owner, repo string) ([]*thirdapi.RepositoryBranch, error) {
	parse, err := s.client.BaseURL.Parse(s.client.BaseURL.String() + fmt.Sprintf(ApiGiteeGetRepoBranches, owner, repo, accessToken))
	if err != nil {
		return nil, fmt.Errorf("gitee GetRepoBranches: parse URL: %w", err)
	}
	logrus.Debugf("Gitee Api GetRepoBranches url : %v", parse)
	req, err := http.NewRequestWithContext(ctx, "GET", parse.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("gitee GetRepoBranches: create request: %w", err)
	}
	resp, err := s.client.HttpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("gitee GetRepoBranches: HTTP request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	all, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("gitee GetRepoBranches: read response body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("gitee api GetRepoBranches failed (status %d): %s", resp.StatusCode, string(all))
	}

	var branchList []*thirdbean.ResultGiteeRepoBranch
	err = json.Unmarshal(all, &branchList)
	if err != nil {
		return nil, fmt.Errorf("gitee GetRepoBranches: unmarshal response: %w", err)
	}
	return convertBranchList(branchList), nil
}

func (s *RepositoryService) GetWebHooks(ctx context.Context, accessToken, owner, repo string, page, perPage int) ([]*thirdapi.RepositoryHook, error) {
	parse, err := s.client.BaseURL.Parse(s.client.BaseURL.String() + fmt.Sprintf(ApiGiteeGetHooks, owner, repo, accessToken, page, perPage))
	if err != nil {
		return nil, fmt.Errorf("gitee GetWebHooks: parse URL: %w", err)
	}
	logrus.Debugf("Gitee Api GetWebHooks url : %v", parse)
	req, err := http.NewRequestWithContext(ctx, "GET", parse.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("gitee GetWebHooks: create request: %w", err)
	}
	resp, err := s.client.HttpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("gitee GetWebHooks: HTTP request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	all, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("gitee GetWebHooks: read response body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("gitee api GetWebHooks failed (status %d): %s", resp.StatusCode, string(all))
	}
	hs := make([]*thirdbean.ResultGetGiteeHook, 0)
	err = json.Unmarshal(all, &hs)
	if err != nil {
		return nil, fmt.Errorf("gitee GetWebHooks: unmarshal response: %w", err)
	}
	return convertHookList(hs), nil
}

func convertHookList(ls []*thirdbean.ResultGetGiteeHook) []*thirdapi.RepositoryHook {
	repos := make([]*thirdapi.RepositoryHook, 0, len(ls))
	for _, v := range ls {
		repos = append(repos, convertHook(v))
	}
	return repos
}

func convertHook(from *thirdbean.ResultGetGiteeHook) *thirdapi.RepositoryHook {
	return &thirdapi.RepositoryHook{
		Id:        from.Id,
		Url:       from.Url,
		CreatedAt: from.CreatedAt,
	}
}

func convertBranchList(ls []*thirdbean.ResultGiteeRepoBranch) []*thirdapi.RepositoryBranch {
	repos := make([]*thirdapi.RepositoryBranch, 0, len(ls))
	for _, v := range ls {
		repos = append(repos, convertBranch(v))
	}
	return repos
}

func convertBranch(from *thirdbean.ResultGiteeRepoBranch) *thirdapi.RepositoryBranch {
	return &thirdapi.RepositoryBranch{
		Name: from.Name,
	}
}

func convertRepositoryList(ls []*thirdbean.ResultGiteeRepo) []*thirdapi.Repository {
	repos := make([]*thirdapi.Repository, 0, len(ls))
	for _, v := range ls {
		repos = append(repos, convertRepository(v))
	}
	return repos
}

func convertRepository(from *thirdbean.ResultGiteeRepo) *thirdapi.Repository {
	return &thirdapi.Repository{
		Id:        strconv.FormatInt(from.Id, 10),
		Owner:     from.Owner.Login,
		Name:      from.Name,
		Path:      from.Path,
		Namespace: from.Namespace.Path,
		FullName:  from.FullName,
		HtmlURL:   from.HtmlUrl,
		RepoType:  "gitee",
	}
}
