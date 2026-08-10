package githubapi

import (
	"bytes"
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
	parse, err := s.client.BaseURL.Parse(s.client.BaseURL.String() + fmt.Sprintf(ApiGithubGetRepos, types, sort, direction, page, perPage))
	if err != nil {
		return nil, fmt.Errorf("github GetRepos: parse URL: %w", err)
	}
	logrus.Debugf("Github Api GetRepos url : %v", parse.String())
	req, err := http.NewRequestWithContext(ctx, "GET", parse.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("github GetRepos: create request: %w", err)
	}
	req.Header.Set("Authorization", "token "+accessToken)
	resp, err := s.client.HttpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("github GetRepos: HTTP request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return nil, thirdapi.NewAPIError("github", "GetRepos", resp.StatusCode, "")
	}
	repos, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("github GetRepos: read response body: %w", err)
	}
	var repoList []*thirdbean.ResultGithubRepo
	err = json.Unmarshal(repos, &repoList)
	if err != nil {
		return nil, fmt.Errorf("github GetRepos: unmarshal response: %w", err)
	}
	lk := resp.Header.Get("Link")
	var totalPages int64 = 1
	if lk != "" {
		splits := strings.Split(lk, ", ")
		for _, v := range splits {
			if strings.Contains(v, `rel="last"`) {
				// Extract URL from format: <https://...>; rel="last"
				start := strings.Index(v, "<")
				end := strings.Index(v, ">")
				if start >= 0 && end > start {
					urlStr := v[start+1 : end]
					p, errs := url.Parse(urlStr)
					if errs != nil {
						logrus.Errorf("Github Api GetRepos url Parse err : %v", errs)
					} else {
						get := p.Query().Get("page")
						totalPages, err = strconv.ParseInt(get, 10, 64)
						if err != nil {
							return nil, fmt.Errorf("github GetRepos: parse total pages: %w", err)
						}
					}
				}
			}
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
	parse, err := s.client.BaseURL.Parse(s.client.BaseURL.String() + fmt.Sprintf(ApiGithubDeleteHooks, owner, repo, hookID))
	if err != nil {
		return fmt.Errorf("github DeleteHooks: parse URL: %w", err)
	}
	request, err := http.NewRequestWithContext(ctx, "DELETE", parse.String(), nil)
	if err != nil {
		return fmt.Errorf("github DeleteHooks: create request: %w", err)
	}
	request.Header.Set("Authorization", "token "+accessToken)
	logrus.Debugf("Github Api DeleteHooks url : %v", parse)
	resp, err := s.client.HttpClient.Do(request)
	if err != nil {
		return fmt.Errorf("github DeleteHooks: HTTP request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	all, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("github DeleteHooks: read response body: %w", err)
	}

	if resp.StatusCode != http.StatusNoContent {
		return thirdapi.NewAPIError("github", "DeleteHooks", resp.StatusCode, string(all))
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
	parse, err := s.client.BaseURL.Parse(s.client.BaseURL.String() + fmt.Sprintf(ApiGithubCreateHooks, owner, repo))
	if err != nil {
		return nil, fmt.Errorf("github CreateWebHooks: parse URL: %w", err)
	}
	logrus.Debugf("Github Api CreateWebHooks url : %s", parse.String())
	m := map[string]any{}
	m["url"] = backURL
	m["content_type"] = "json"
	m["secret"] = password
	m["token"] = accessToken
	obj := map[string]map[string]any{}
	obj["config"] = m
	marshal, err := json.Marshal(obj)
	if err != nil {
		return nil, fmt.Errorf("github CreateWebHooks: marshal request body: %w", err)
	}
	logrus.Debugf("CreateWebHooks json %s", string(marshal))
	request, err := http.NewRequestWithContext(ctx, "POST", parse.String(), bytes.NewBuffer(marshal))
	if err != nil {
		return nil, fmt.Errorf("github CreateWebHooks: create request: %w", err)
	}
	request.Header.Set("Authorization", "token "+accessToken)
	request.Header.Set("Content-Type", "application/json")
	resp, err := s.client.HttpClient.Do(request)
	if err != nil {
		return nil, fmt.Errorf("github CreateWebHooks: HTTP request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	all, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("github CreateWebHooks: read response body: %w", err)
	}
	if resp.StatusCode != http.StatusCreated {
		return nil, thirdapi.NewAPIError("github", "CreateWebHooks", resp.StatusCode, string(all))
	}
	k := &thirdbean.ResultGetGithubHook{}
	err = json.Unmarshal(all, k)
	if err != nil {
		return nil, fmt.Errorf("github CreateWebHooks: unmarshal response: %w", err)
	}
	return convertHook(k), nil
}

func (s *RepositoryService) GetRepoBranches(ctx context.Context, accessToken, owner, repo string) ([]*thirdapi.RepositoryBranch, error) {
	parse, err := s.client.BaseURL.Parse(s.client.BaseURL.String() + fmt.Sprintf(ApiGithubGetRepoBranches, owner, repo))
	if err != nil {
		return nil, fmt.Errorf("github GetRepoBranches: parse URL: %w", err)
	}
	logrus.Debugf("Github Api GetRepoBranches url : %v", parse)
	req, err := http.NewRequestWithContext(ctx, "GET", parse.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("github GetRepoBranches: create request: %w", err)
	}
	req.Header.Set("Authorization", "token "+accessToken)
	resp, err := s.client.HttpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("github GetRepoBranches: HTTP request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	all, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("github GetRepoBranches: read response body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, thirdapi.NewAPIError("github", "GetRepoBranches", resp.StatusCode, string(all))
	}
	var branchList []*thirdbean.ResultGithubRepoBranch
	err = json.Unmarshal(all, &branchList)
	if err != nil {
		return nil, fmt.Errorf("github GetRepoBranches: unmarshal response: %w", err)
	}
	return convertBranchList(branchList), nil
}

func (s *RepositoryService) GetWebHooks(ctx context.Context, accessToken, owner, repo string, page, perPage int) ([]*thirdapi.RepositoryHook, error) {
	parse, err := s.client.BaseURL.Parse(s.client.BaseURL.String() + fmt.Sprintf(ApiGithubGetHooks, owner, repo, page, perPage))
	if err != nil {
		return nil, fmt.Errorf("github GetWebHooks: parse URL: %w", err)
	}
	logrus.Debugf("Github Api GetWebHooks url : %v", parse)
	req, err := http.NewRequestWithContext(ctx, "GET", parse.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("github GetWebHooks: create request: %w", err)
	}
	req.Header.Set("Authorization", "token "+accessToken)
	resp, err := s.client.HttpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("github GetWebHooks: HTTP request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	all, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("github GetWebHooks: read response body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, thirdapi.NewAPIError("github", "GetWebHooks", resp.StatusCode, string(all))
	}

	hs := make([]*thirdbean.ResultGetGithubHook, 0)
	err = json.Unmarshal(all, &hs)
	if err != nil {
		return nil, fmt.Errorf("github GetWebHooks: unmarshal response: %w", err)
	}
	return convertHookList(hs), nil
}

func convertRepositoryList(ls []*thirdbean.ResultGithubRepo) []*thirdapi.Repository {
	repos := make([]*thirdapi.Repository, 0, len(ls))
	for _, v := range ls {
		repos = append(repos, convertRepository(v))
	}
	return repos
}

func convertRepository(from *thirdbean.ResultGithubRepo) *thirdapi.Repository {
	return &thirdapi.Repository{
		Id:        strconv.Itoa(from.Id),
		Owner:     from.Owner.Login,
		Name:      from.Name,
		Path:      from.Name,
		Namespace: from.Owner.Login,
		FullName:  from.FullName,
		HtmlURL:   from.HtmlUrl,
		RepoType:  "github",
	}
}
func convertBranchList(ls []*thirdbean.ResultGithubRepoBranch) []*thirdapi.RepositoryBranch {
	repos := make([]*thirdapi.RepositoryBranch, 0, len(ls))
	for _, v := range ls {
		repos = append(repos, convertBranch(v))
	}
	return repos
}

func convertBranch(from *thirdbean.ResultGithubRepoBranch) *thirdapi.RepositoryBranch {
	return &thirdapi.RepositoryBranch{
		Name: from.Name,
	}
}
func convertHookList(ls []*thirdbean.ResultGetGithubHook) []*thirdapi.RepositoryHook {
	repos := make([]*thirdapi.RepositoryHook, 0, len(ls))
	for _, v := range ls {
		repos = append(repos, convertHook(v))
	}
	return repos
}

func convertHook(from *thirdbean.ResultGetGithubHook) *thirdapi.RepositoryHook {
	return &thirdapi.RepositoryHook{
		Id:        from.Id,
		Url:       from.Config.Url,
		CreatedAt: from.CreatedAt,
	}
}
