package gitlabapi

import (
	"strconv"
	"testing"
	"time"

	"github.com/gokins/gokins/bean/thirdbean"
)

func makeGitlabRepo(id int, name, path, pathWithNamespace, webURL, ownerUsername, nsPath string) *thirdbean.ResultGitlabRepo {
	r := &thirdbean.ResultGitlabRepo{
		Id:                id,
		Name:              name,
		Path:              path,
		PathWithNamespace: pathWithNamespace,
		WebUrl:            webURL,
	}
	r.Owner.Username = ownerUsername
	r.Namespace.Path = nsPath
	return r
}

func TestConvertRepositoryList_Gitlab(t *testing.T) {
	input := []*thirdbean.ResultGitlabRepo{
		makeGitlabRepo(123, "repo1", "repo1", "owner/repo1", "https://gitlab.com/owner/repo1", "owner", "owner"),
		makeGitlabRepo(456, "repo2", "repo2", "owner/repo2", "https://gitlab.com/owner/repo2", "owner", "owner"),
	}

	result := convertRepositoryList(input)

	if len(result) != 2 {
		t.Fatalf("expected 2 repositories, got %d", len(result))
	}

	if result[0].Id != "123" {
		t.Errorf("expected repo1 id '123', got %s", result[0].Id)
	}
	if result[0].Name != "repo1" {
		t.Errorf("expected repo1 name 'repo1', got %s", result[0].Name)
	}
	if result[0].Path != "repo1" {
		t.Errorf("expected repo1 path 'repo1', got %s", result[0].Path)
	}
	if result[0].FullName != "owner/repo1" {
		t.Errorf("expected repo1 full name 'owner/repo1', got %s", result[0].FullName)
	}
	if result[0].Owner != "owner" {
		t.Errorf("expected repo1 owner 'owner', got %s", result[0].Owner)
	}
	if result[0].Namespace != "owner" {
		t.Errorf("expected repo1 namespace 'owner', got %s", result[0].Namespace)
	}
	if result[0].RepoType != "gitlab" {
		t.Errorf("expected repo1 type 'gitlab', got %s", result[0].RepoType)
	}
}

func TestConvertRepository_Gitlab(t *testing.T) {
	input := makeGitlabRepo(789, "test-repo", "test-repo", "group/test-repo", "https://gitlab.com/group/test-repo", "testowner", "group")

	result := convertRepository(input)

	if result.Id != strconv.Itoa(789) {
		t.Errorf("expected id '789', got %s", result.Id)
	}
	if result.Name != "test-repo" {
		t.Errorf("expected name 'test-repo', got %s", result.Name)
	}
	if result.Path != "test-repo" {
		t.Errorf("expected path 'test-repo', got %s", result.Path)
	}
	if result.FullName != "group/test-repo" {
		t.Errorf("expected full name 'group/test-repo', got %s", result.FullName)
	}
	if result.Owner != "testowner" {
		t.Errorf("expected owner 'testowner', got %s", result.Owner)
	}
	if result.Namespace != "group" {
		t.Errorf("expected namespace 'group', got %s", result.Namespace)
	}
	if result.HtmlURL != "https://gitlab.com/group/test-repo" {
		t.Errorf("expected html url 'https://gitlab.com/group/test-repo', got %s", result.HtmlURL)
	}
	if result.RepoType != "gitlab" {
		t.Errorf("expected type 'gitlab', got %s", result.RepoType)
	}
}

func TestConvertBranchList_Gitlab(t *testing.T) {
	input := []*thirdbean.ResultGitlabRepoBranch{
		{Name: "main"},
		{Name: "develop"},
		{Name: "feature/test"},
	}

	result := convertBranchList(input)

	if len(result) != 3 {
		t.Fatalf("expected 3 branches, got %d", len(result))
	}

	expected := []string{"main", "develop", "feature/test"}
	for i, name := range expected {
		if result[i].Name != name {
			t.Errorf("branch[%d]: expected name %q, got %q", i, name, result[i].Name)
		}
	}
}

func TestConvertBranch_Gitlab(t *testing.T) {
	input := &thirdbean.ResultGitlabRepoBranch{Name: "main"}

	result := convertBranch(input)

	if result.Name != "main" {
		t.Errorf("expected branch name 'main', got %s", result.Name)
	}
}

func TestConvertHookList_Gitlab(t *testing.T) {
	now := time.Now()
	input := []*thirdbean.ResultGetGitlabHook{
		{Id: 1, Url: "https://example.com/hook1", CreatedAt: now},
		{Id: 2, Url: "https://example.com/hook2", CreatedAt: now.Add(time.Hour)},
	}

	result := convertHookList(input)

	if len(result) != 2 {
		t.Fatalf("expected 2 hooks, got %d", len(result))
	}

	if result[0].Id != 1 {
		t.Errorf("expected hook id 1, got %v", result[0].Id)
	}
	if result[0].Url != "https://example.com/hook1" {
		t.Errorf("expected hook url 'https://example.com/hook1', got %s", result[0].Url)
	}
	if result[1].Id != 2 {
		t.Errorf("expected hook id 2, got %v", result[1].Id)
	}
}

func TestConvertHook_Gitlab(t *testing.T) {
	now := time.Now()
	input := &thirdbean.ResultGetGitlabHook{
		Id:        42,
		Url:       "https://example.com/webhook",
		CreatedAt: now,
	}

	result := convertHook(input)

	if result.Id != 42 {
		t.Errorf("expected hook id 42, got %v", result.Id)
	}
	if result.Url != "https://example.com/webhook" {
		t.Errorf("expected hook url 'https://example.com/webhook', got %s", result.Url)
	}
	if result.CreatedAt != now {
		t.Errorf("expected hook created at %v, got %v", now, result.CreatedAt)
	}
}

func TestConvertEmptyLists_Gitlab(t *testing.T) {
	repos := convertRepositoryList([]*thirdbean.ResultGitlabRepo{})
	if len(repos) != 0 {
		t.Errorf("expected 0 repositories, got %d", len(repos))
	}

	branches := convertBranchList([]*thirdbean.ResultGitlabRepoBranch{})
	if len(branches) != 0 {
		t.Errorf("expected 0 branches, got %d", len(branches))
	}

	hooks := convertHookList([]*thirdbean.ResultGetGitlabHook{})
	if len(hooks) != 0 {
		t.Errorf("expected 0 hooks, got %d", len(hooks))
	}
}

func TestConvertHookList_Nil_Gitlab(t *testing.T) {
	result := convertHookList(nil)
	if len(result) != 0 {
		t.Errorf("expected 0 hooks for nil input, got %d", len(result))
	}
}

func TestConvertRepositoryList_LargeList_Gitlab(t *testing.T) {
	var input []*thirdbean.ResultGitlabRepo
	for i := 0; i < 100; i++ {
		input = append(input, makeGitlabRepo(i, "repo"+strconv.Itoa(i), "repo"+strconv.Itoa(i), "group/repo"+strconv.Itoa(i), "https://gitlab.com/group/repo"+strconv.Itoa(i), "owner", "group"))
	}

	result := convertRepositoryList(input)

	if len(result) != 100 {
		t.Fatalf("expected 100 repositories, got %d", len(result))
	}
	for i := 0; i < 100; i++ {
		if result[i].Id != strconv.Itoa(i) {
			t.Errorf("repo[%d]: expected id %q, got %s", i, strconv.Itoa(i), result[i].Id)
		}
	}
}
