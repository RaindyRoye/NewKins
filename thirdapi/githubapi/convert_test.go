package githubapi

import (
	"strconv"
	"testing"
	"time"

	"github.com/gokins/gokins/bean/thirdbean"
)

func makeGithubRepo(id int, name, fullName, htmlURL, ownerLogin string) *thirdbean.ResultGithubRepo {
	r := &thirdbean.ResultGithubRepo{
		Id:       id,
		Name:     name,
		FullName: fullName,
		HtmlUrl:  htmlURL,
	}
	r.Owner.Login = ownerLogin
	return r
}

func TestConvertRepositoryList_Github(t *testing.T) {
	input := []*thirdbean.ResultGithubRepo{
		makeGithubRepo(123, "repo1", "owner/repo1", "https://github.com/owner/repo1", "owner"),
		makeGithubRepo(456, "repo2", "owner/repo2", "https://github.com/owner/repo2", "owner"),
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
	if result[0].FullName != "owner/repo1" {
		t.Errorf("expected repo1 full name 'owner/repo1', got %s", result[0].FullName)
	}
	if result[0].Owner != "owner" {
		t.Errorf("expected repo1 owner 'owner', got %s", result[0].Owner)
	}
	if result[0].RepoType != "github" {
		t.Errorf("expected repo1 type 'github', got %s", result[0].RepoType)
	}
}

func TestConvertRepository_Github(t *testing.T) {
	input := makeGithubRepo(789, "test-repo", "owner/test-repo", "https://github.com/owner/test-repo", "owner")

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
	if result.Namespace != "owner" {
		t.Errorf("expected namespace 'owner', got %s", result.Namespace)
	}
	if result.HtmlURL != "https://github.com/owner/test-repo" {
		t.Errorf("expected html url 'https://github.com/owner/test-repo', got %s", result.HtmlURL)
	}
	if result.RepoType != "github" {
		t.Errorf("expected type 'github', got %s", result.RepoType)
	}
}

func TestConvertBranchList_Github(t *testing.T) {
	input := []*thirdbean.ResultGithubRepoBranch{
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

func TestConvertBranch_Github(t *testing.T) {
	input := &thirdbean.ResultGithubRepoBranch{Name: "main"}

	result := convertBranch(input)

	if result.Name != "main" {
		t.Errorf("expected branch name 'main', got %s", result.Name)
	}
}

func TestConvertHookList_Github(t *testing.T) {
	now := time.Now()
	h1 := &thirdbean.ResultGetGithubHook{
		Id:        1,
		CreatedAt: now,
	}
	h1.Config.Url = "https://example.com/hook1"

	h2 := &thirdbean.ResultGetGithubHook{
		Id:        2,
		CreatedAt: now.Add(time.Hour),
	}
	h2.Config.Url = "https://example.com/hook2"

	result := convertHookList([]*thirdbean.ResultGetGithubHook{h1, h2})

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

func TestConvertHook_Github(t *testing.T) {
	now := time.Now()
	input := &thirdbean.ResultGetGithubHook{
		Id:        42,
		CreatedAt: now,
	}
	input.Config.Url = "https://example.com/webhook"

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

func TestConvertEmptyLists_Github(t *testing.T) {
	repos := convertRepositoryList([]*thirdbean.ResultGithubRepo{})
	if len(repos) != 0 {
		t.Errorf("expected 0 repositories, got %d", len(repos))
	}

	branches := convertBranchList([]*thirdbean.ResultGithubRepoBranch{})
	if len(branches) != 0 {
		t.Errorf("expected 0 branches, got %d", len(branches))
	}

	hooks := convertHookList([]*thirdbean.ResultGetGithubHook{})
	if len(hooks) != 0 {
		t.Errorf("expected 0 hooks, got %d", len(hooks))
	}
}

func TestConvertHookList_Nil(t *testing.T) {
	result := convertHookList(nil)
	if len(result) != 0 {
		t.Errorf("expected 0 hooks for nil input, got %d", len(result))
	}
}

func TestConvertRepositoryList_LargeList(t *testing.T) {
	var input []*thirdbean.ResultGithubRepo
	for i := 0; i < 100; i++ {
		input = append(input, makeGithubRepo(i, "repo"+strconv.Itoa(i), "owner/repo"+strconv.Itoa(i), "https://github.com/owner/repo"+strconv.Itoa(i), "owner"))
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
