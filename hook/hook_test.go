package hook

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

// TestActionConstants verifies that Action constants have correct string values.
func TestActionConstants(t *testing.T) {
	tests := []struct {
		name     string
		action   Action
		expected string
	}{
		{"ActionOpen", ActionOpen, "open"},
		{"ActionOpened", ActionOpened, "opened"},
		{"ActionClose", ActionClose, "close"},
		{"ActionCreate", ActionCreate, "create"},
		{"ActionDelete", ActionDelete, "delete"},
		{"ActionSync", ActionSync, "sync"},
		{"ActionUpdate", ActionUpdate, "update"},
		{"ActionSynchronize", ActionSynchronize, "synchronize"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if string(tc.action) != tc.expected {
				t.Errorf("expected %q, got %q", tc.expected, tc.action)
			}
		})
	}
}

// TestActionType verifies Action is a string-based type.
func TestActionType(t *testing.T) {
	var a Action = "open"
	if a != ActionOpen {
		t.Errorf("expected ActionOpen, got %v", a)
	}
}

// TestEventConstants verifies event type constants are non-empty.
func TestEventConstants(t *testing.T) {
	eventTypes := []struct {
		name  string
		value string
	}{
		{"EventsTypeComment", EventsTypeComment},
		{"EventsTypePR", EventsTypePR},
		{"EventsTypePush", EventsTypePush},
		{"EventsTypeBuild", EventsTypeBuild},
		{"EventsTypeRebuild", EventsTypeRebuild},
	}

	for _, et := range eventTypes {
		t.Run(et.name, func(t *testing.T) {
			if et.value == "" {
				t.Errorf("%s should not be empty", et.name)
			}
		})
	}
}

// TestProviderEventHeaders verifies provider-specific event header constants.
func TestProviderEventHeaders(t *testing.T) {
	// Gitee headers
	if GiteeEvent == "" {
		t.Error("GiteeEvent header should not be empty")
	}
	if GiteeEventPush == "" {
		t.Error("GiteeEventPush should not be empty")
	}

	// GitHub headers
	if GithubEvent == "" {
		t.Error("GithubEvent header should not be empty")
	}
	if GithubEventPush == "" {
		t.Error("GithubEventPush should not be empty")
	}

	// GitLab headers
	if GitlabEvent == "" {
		t.Error("GitlabEvent header should not be empty")
	}
	if GitlabEventPush == "" {
		t.Error("GitlabEventPush should not be empty")
	}

	// Gitea headers
	if GiteaEvent == "" {
		t.Error("GiteaEvent header should not be empty")
	}
	if GiteaEventPush == "" {
		t.Error("GiteaEventPush should not be empty")
	}
}

// TestPushHookRepository verifies PushHook implements WebHook interface.
func TestPushHookRepository(t *testing.T) {
	repo := Repository{
		Id:       "1",
		Name:     "test-repo",
		FullName: "user/test-repo",
		Branch:   "main",
	}
	hook := &PushHook{
		Ref:    "refs/heads/main",
		Repo:   repo,
		Before: "abc123",
		After:  "def456",
		Commit: Commit{Sha: "def456", Message: "test"},
		Sender: User{UserName: "testuser"},
	}

	// Verify WebHook interface
	var _ WebHook = hook

	got := hook.Repository()
	if got.Id != repo.Id {
		t.Errorf("expected repo ID %q, got %q", repo.Id, got.Id)
	}
	if got.Name != repo.Name {
		t.Errorf("expected repo name %q, got %q", repo.Name, got.Name)
	}
	if got.FullName != repo.FullName {
		t.Errorf("expected full name %q, got %q", repo.FullName, got.FullName)
	}
}

// TestBranchHookRepository verifies BranchHook implements WebHook interface.
func TestBranchHookRepository(t *testing.T) {
	repo := Repository{
		Id:       "2",
		Name:     "branch-repo",
		FullName: "user/branch-repo",
	}
	hook := &BranchHook{
		Ref:    Reference{Name: "feature", Sha: "abc123"},
		Repo:   repo,
		Sender: User{UserName: "testuser"},
	}

	var _ WebHook = hook

	got := hook.Repository()
	if got.Id != repo.Id {
		t.Errorf("expected repo ID %q, got %q", repo.Id, got.Id)
	}
	if got.Name != repo.Name {
		t.Errorf("expected repo name %q, got %q", repo.Name, got.Name)
	}
}

// TestPullRequestHookRepository verifies PullRequestHook implements WebHook interface.
func TestPullRequestHookRepository(t *testing.T) {
	repo := Repository{
		Id:       "3",
		Name:     "pr-repo",
		FullName: "user/pr-repo",
	}
	targetRepo := Repository{
		Id:       "4",
		Name:     "target-repo",
		FullName: "user/target-repo",
	}
	hook := &PullRequestHook{
		Action:     ActionOpen,
		Repo:       repo,
		TargetRepo: targetRepo,
		PullRequest: PullRequest{
			Number: 42,
			Title:  "Test PR",
		},
		Sender: User{UserName: "prauthor"},
	}

	var _ WebHook = hook

	got := hook.Repository()
	if got.Id != repo.Id {
		t.Errorf("expected repo ID %q, got %q", repo.Id, got.Id)
	}
	if hook.TargetRepo.Id != targetRepo.Id {
		t.Errorf("expected target repo ID %q, got %q", targetRepo.Id, hook.TargetRepo.Id)
	}
	if hook.PullRequest.Number != 42 {
		t.Errorf("expected PR number 42, got %d", hook.PullRequest.Number)
	}
}

// TestPullRequestCommentHookRepository verifies PullRequestCommentHook implements WebHook interface.
func TestPullRequestCommentHookRepository(t *testing.T) {
	repo := Repository{
		Id:       "5",
		Name:     "comment-repo",
		FullName: "user/comment-repo",
	}
	hook := &PullRequestCommentHook{
		Action:     ActionCreate,
		Repo:       repo,
		TargetRepo: repo,
		PullRequest: PullRequest{
			Number: 10,
			Title:  "Commented PR",
		},
		Comment: Comment{
			Body:   "Nice work!",
			Author: User{UserName: "reviewer"},
		},
		Sender: User{UserName: "reviewer"},
	}

	var _ WebHook = hook

	got := hook.Repository()
	if got.Id != repo.Id {
		t.Errorf("expected repo ID %q, got %q", repo.Id, got.Id)
	}
	if hook.Comment.Body != "Nice work!" {
		t.Errorf("expected comment body 'Nice work!', got %q", hook.Comment.Body)
	}
	if hook.Comment.Author.UserName != "reviewer" {
		t.Errorf("expected comment author 'reviewer', got %q", hook.Comment.Author.UserName)
	}
}

// TestWebHookInterface verifies all hook types satisfy the WebHook interface at compile time.
func TestWebHookInterface(t *testing.T) {
	repo := Repository{Id: "1", Name: "test"}

	hooks := []WebHook{
		&PushHook{Repo: repo},
		&BranchHook{Repo: repo},
		&PullRequestHook{Repo: repo},
		&PullRequestCommentHook{Repo: repo},
	}

	for i, h := range hooks {
		got := h.Repository()
		if got.Id != "1" {
			t.Errorf("hook[%d]: expected repo ID '1', got %q", i, got.Id)
		}
	}
}

// TestSecretFuncType verifies SecretFunc type signature.
func TestSecretFuncType(t *testing.T) {
	fn := SecretFunc(func(webhook WebHook) (string, error) {
		return "mysecret", nil
	})

	hook := &PushHook{Repo: Repository{Id: "1"}}
	secret, err := fn(hook)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if secret != "mysecret" {
		t.Errorf("expected 'mysecret', got %q", secret)
	}
}

// TestRepositoryFields verifies Repository struct field access.
func TestRepositoryFields(t *testing.T) {
	repo := Repository{
		Id:          "42",
		Ref:         "refs/heads/main",
		Sha:         "abc123def456",
		CloneURL:    "https://example.com/repo.git",
		Branch:      "main",
		Description: "Test repo",
		FullName:    "user/repo",
		GitHttpURL:  "https://example.com/repo.git",
		GitShhURL:   "git@example.com:user/repo.git",
		GitURL:      "git://example.com/repo.git",
		HtmlURL:     "https://example.com/user/repo",
		SshURL:      "git@example.com:user/repo.git",
		Name:        "repo",
		Private:     true,
		URL:         "https://example.com/user/repo",
		Owner:       "user",
		RepoType:    "github",
		RepoOpenid:  "openid123",
	}

	if repo.Id != "42" {
		t.Errorf("expected Id '42', got %q", repo.Id)
	}
	if !repo.Private {
		t.Error("expected Private to be true")
	}
	if repo.RepoType != "github" {
		t.Errorf("expected RepoType 'github', got %q", repo.RepoType)
	}
}

// TestPullRequestFields verifies PullRequest struct field access.
func TestPullRequestFields(t *testing.T) {
	pr := PullRequest{
		Number: 99,
		Title:  "Fix bug",
		Body:   "This fixes the bug",
		Base:   Reference{Name: "main", Path: "refs/heads/main", Sha: "base123"},
		Head:   Reference{Name: "fix-branch", Path: "refs/heads/fix-branch", Sha: "head456"},
		Author: User{UserName: "developer"},
	}

	if pr.Number != 99 {
		t.Errorf("expected number 99, got %d", pr.Number)
	}
	if pr.Base.Name != "main" {
		t.Errorf("expected base branch 'main', got %q", pr.Base.Name)
	}
	if pr.Head.Sha != "head456" {
		t.Errorf("expected head sha 'head456', got %q", pr.Head.Sha)
	}
	if pr.Author.UserName != "developer" {
		t.Errorf("expected author 'developer', got %q", pr.Author.UserName)
	}
}

// TestWebhookServiceInterface verifies the WebhookService interface contract.
func TestWebhookServiceInterface(t *testing.T) {
	// Verify interface can be implemented
	var _ WebhookService = &mockWebhookService{}
}

type mockWebhookService struct{}

func (m *mockWebhookService) Parse(req *http.Request, fn SecretFunc) (WebHook, error) {
	return &PushHook{Repo: Repository{Id: "mock"}}, nil
}

// TestMockWebhookServiceParse tests the mock implementation for completeness.
func TestMockWebhookServiceParse(t *testing.T) {
	svc := &mockWebhookService{}
	req := httptest.NewRequest(http.MethodPost, "/webhook", nil)

	fn := func(webhook WebHook) (string, error) {
		return "secret", nil
	}

	wh, err := svc.Parse(req, fn)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if wh.Repository().Id != "mock" {
		t.Errorf("expected repo ID 'mock', got %q", wh.Repository().Id)
	}
}

// TestCommitFields verifies Commit struct field access.
func TestCommitFields(t *testing.T) {
	c := Commit{
		Sha:     "abc123",
		Message: "Initial commit",
		Link:    "https://example.com/commit/abc123",
	}

	if c.Sha != "abc123" {
		t.Errorf("expected sha 'abc123', got %q", c.Sha)
	}
	if c.Message != "Initial commit" {
		t.Errorf("expected message 'Initial commit', got %q", c.Message)
	}
	if c.Link != "https://example.com/commit/abc123" {
		t.Errorf("expected link, got %q", c.Link)
	}
}

// TestUserFields verifies User struct field access.
func TestUserFields(t *testing.T) {
	u := User{UserName: "testuser"}
	if u.UserName != "testuser" {
		t.Errorf("expected 'testuser', got %q", u.UserName)
	}
}

// TestReferenceFields verifies Reference struct field access.
func TestReferenceFields(t *testing.T) {
	ref := Reference{
		Name: "main",
		Path: "refs/heads/main",
		Sha:  "abc123",
	}
	if ref.Name != "main" {
		t.Errorf("expected name 'main', got %q", ref.Name)
	}
	if ref.Path != "refs/heads/main" {
		t.Errorf("expected path 'refs/heads/main', got %q", ref.Path)
	}
	if ref.Sha != "abc123" {
		t.Errorf("expected sha 'abc123', got %q", ref.Sha)
	}
}

// TestPushHookWithMultipleCommits verifies PushHook handles multiple commits.
func TestPushHookWithMultipleCommits(t *testing.T) {
	hook := &PushHook{
		Ref: "refs/heads/main",
		Commits: []Commit{
			{Sha: "aaa", Message: "first"},
			{Sha: "bbb", Message: "second"},
			{Sha: "ccc", Message: "third"},
		},
	}

	if len(hook.Commits) != 3 {
		t.Errorf("expected 3 commits, got %d", len(hook.Commits))
	}
	if hook.Commits[0].Sha != "aaa" {
		t.Errorf("expected first commit sha 'aaa', got %q", hook.Commits[0].Sha)
	}
	if hook.Commits[2].Message != "third" {
		t.Errorf("expected third commit message 'third', got %q", hook.Commits[2].Message)
	}
}
