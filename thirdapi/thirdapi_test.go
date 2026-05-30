package thirdapi

import (
	"encoding/json"
	"net/http"
	"net/url"
	"testing"
	"time"
)

func TestRepositoryJSONMarshal(t *testing.T) {
	repo := Repository{
		Id:        "123",
		Owner:     "user",
		Name:      "test-repo",
		Path:      "/user/test-repo",
		Namespace: "user",
		FullName:  "user/test-repo",
		HtmlURL:   "https://example.com/user/test-repo",
		RepoType:  "github", // Note: json:"-" tag means this is NOT serialized
	}

	data, err := json.Marshal(repo)
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}

	var decoded Repository
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}

	if decoded.Id != "123" {
		t.Errorf("expected Id '123', got %q", decoded.Id)
	}
	if decoded.Owner != "user" {
		t.Errorf("expected Owner 'user', got %q", decoded.Owner)
	}
	if decoded.FullName != "user/test-repo" {
		t.Errorf("expected FullName 'user/test-repo', got %q", decoded.FullName)
	}

	// RepoType has json:"-" tag, so it should NOT be serialized
	var rawMap map[string]interface{}
	if err := json.Unmarshal(data, &rawMap); err != nil {
		t.Fatalf("unmarshal to map failed: %v", err)
	}
	if _, exists := rawMap["repoType"]; exists {
		t.Error("RepoType should not be serialized (json:\"-\" tag)")
	}
}

func TestRepositoryJSONUnmarshal(t *testing.T) {
	jsonStr := `{
		"id": "456",
		"owner": "org",
		"name": "api-repo",
		"fullName": "org/api-repo",
		"htmlURL": "https://example.com/org/api-repo"
	}`

	var repo Repository
	if err := json.Unmarshal([]byte(jsonStr), &repo); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}

	if repo.Id != "456" {
		t.Errorf("expected Id '456', got %q", repo.Id)
	}
	if repo.Owner != "org" {
		t.Errorf("expected Owner 'org', got %q", repo.Owner)
	}
	if repo.Name != "api-repo" {
		t.Errorf("expected Name 'api-repo', got %q", repo.Name)
	}
}

func TestRepositoryPage(t *testing.T) {
	page := &RepositoryPage{
		TotalPages: 5,
		Ropes: []*Repository{
			{Id: "1", Name: "repo1"},
			{Id: "2", Name: "repo2"},
			{Id: "3", Name: "repo3"},
		},
	}

	if page.TotalPages != 5 {
		t.Errorf("expected TotalPages 5, got %d", page.TotalPages)
	}
	if len(page.Ropes) != 3 {
		t.Errorf("expected 3 repos, got %d", len(page.Ropes))
	}
	if page.Ropes[0].Name != "repo1" {
		t.Errorf("expected first repo name 'repo1', got %q", page.Ropes[0].Name)
	}
}

func TestRepositoryPageEmpty(t *testing.T) {
	page := &RepositoryPage{
		TotalPages: 0,
		Ropes:      nil,
	}

	if page.TotalPages != 0 {
		t.Errorf("expected TotalPages 0, got %d", page.TotalPages)
	}
	if page.Ropes != nil {
		t.Error("expected Ropes to be nil")
	}
}

func TestRepositoryBranch(t *testing.T) {
	branch := RepositoryBranch{Name: "main"}
	if branch.Name != "main" {
		t.Errorf("expected name 'main', got %q", branch.Name)
	}
}

func TestRepositoryBranchJSON(t *testing.T) {
	jsonStr := `{"name": "feature/test"}`
	var branch RepositoryBranch
	if err := json.Unmarshal([]byte(jsonStr), &branch); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	if branch.Name != "feature/test" {
		t.Errorf("expected name 'feature/test', got %q", branch.Name)
	}
}

func TestRepositoryHook(t *testing.T) {
	now := time.Now()
	hook := RepositoryHook{
		Id:        42,
		Url:       "https://example.com/webhook",
		CreatedAt: now,
	}

	if hook.Id != 42 {
		t.Errorf("expected Id 42, got %d", hook.Id)
	}
	if hook.Url != "https://example.com/webhook" {
		t.Errorf("expected URL 'https://example.com/webhook', got %q", hook.Url)
	}
}

func TestRepositoryHookJSON(t *testing.T) {
	jsonStr := `{
		"id": 10,
		"url": "https://hooks.example.com/callback",
		"createdAt": "2024-01-15T10:30:00Z"
	}`

	var hook RepositoryHook
	if err := json.Unmarshal([]byte(jsonStr), &hook); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	if hook.Id != 10 {
		t.Errorf("expected Id 10, got %d", hook.Id)
	}
	if hook.Url != "https://hooks.example.com/callback" {
		t.Errorf("expected URL, got %q", hook.Url)
	}
	if hook.CreatedAt.IsZero() {
		t.Error("expected CreatedAt to be parsed")
	}
}

func TestClientCreation(t *testing.T) {
	baseURL, _ := url.Parse("https://api.example.com")
	client := &Client{
		HttpClient: &http.Client{},
		BaseURL:    baseURL,
	}

	if client.HttpClient == nil {
		t.Error("expected HttpClient to be set")
	}
	if client.BaseURL.String() != "https://api.example.com" {
		t.Errorf("expected BaseURL 'https://api.example.com', got %q", client.BaseURL.String())
	}
	if client.Repositories != nil {
		t.Error("expected Repositories to be nil by default")
	}
}

// mockRepoService implements RepositoryService for interface verification.
type mockRepoService struct{}

func (m *mockRepoService) GetRepos(accessToken, username, types, sort, direction string, page, perPage int) (*RepositoryPage, error) {
	return &RepositoryPage{TotalPages: 1, Ropes: []*Repository{{Id: "1"}}}, nil
}
func (m *mockRepoService) DeleteHooks(accessToken, owner, repo, hookId string) error {
	return nil
}
func (m *mockRepoService) CreateWebHooks(accessToken, owner, repo, backUrl, password string) (*RepositoryHook, error) {
	return &RepositoryHook{Id: 1}, nil
}
func (m *mockRepoService) GetRepoBranches(accessToken, owner, repo string) ([]*RepositoryBranch, error) {
	return []*RepositoryBranch{{Name: "main"}}, nil
}
func (m *mockRepoService) GetWebHooks(accessToken, owner, repo string, page, perPage int) ([]*RepositoryHook, error) {
	return []*RepositoryHook{{Id: 1}}, nil
}

func TestRepositoryServiceInterface(t *testing.T) {
	// Verify mockRepoService implements RepositoryService at compile time
	var _ RepositoryService = &mockRepoService{}

	svc := &mockRepoService{}

	// Test GetRepos
	page, err := svc.GetRepos("token", "user", "", "", "", 1, 10)
	if err != nil {
		t.Fatalf("GetRepos failed: %v", err)
	}
	if page.TotalPages != 1 {
		t.Errorf("expected 1 page, got %d", page.TotalPages)
	}

	// Test DeleteHooks
	if err := svc.DeleteHooks("token", "owner", "repo", "1"); err != nil {
		t.Fatalf("DeleteHooks failed: %v", err)
	}

	// Test CreateWebHooks
	hook, err := svc.CreateWebHooks("token", "owner", "repo", "https://callback.com", "pass")
	if err != nil {
		t.Fatalf("CreateWebHooks failed: %v", err)
	}
	if hook.Id != 1 {
		t.Errorf("expected hook ID 1, got %d", hook.Id)
	}

	// Test GetRepoBranches
	branches, err := svc.GetRepoBranches("token", "owner", "repo")
	if err != nil {
		t.Fatalf("GetRepoBranches failed: %v", err)
	}
	if len(branches) != 1 {
		t.Errorf("expected 1 branch, got %d", len(branches))
	}

	// Test GetWebHooks
	hooks, err := svc.GetWebHooks("token", "owner", "repo", 1, 10)
	if err != nil {
		t.Fatalf("GetWebHooks failed: %v", err)
	}
	if len(hooks) != 1 {
		t.Errorf("expected 1 hook, got %d", len(hooks))
	}
}

func TestClientWithRepositoryService(t *testing.T) {
	baseURL, _ := url.Parse("https://api.example.com")
	svc := &mockRepoService{}
	client := &Client{
		HttpClient:   &http.Client{},
		BaseURL:      baseURL,
		Repositories: svc,
	}

	if client.Repositories == nil {
		t.Fatal("expected Repositories to be set")
	}

	page, err := client.Repositories.GetRepos("token", "user", "", "", "", 1, 10)
	if err != nil {
		t.Fatalf("GetRepos via client failed: %v", err)
	}
	if len(page.Ropes) != 1 {
		t.Errorf("expected 1 repo, got %d", len(page.Ropes))
	}
}
