package gitlabapi

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"testing"
)

func TestGiteeContents(t *testing.T) {
	u := fmt.Sprintf(ApiGitlabGetRepos, "SuperHeroJim", "gokins-test", ".gokins", "1065cd3f8791b97224a823c954a0ec98")
	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, u, nil) //nolint:noctx // test-only HTTP request
	if err != nil {
		t.Logf("GitLab Api :%s err : %v", u, err)
		return
	}
	resp, err := http.DefaultClient.Do(req) //nolint:gosec // G107: test-only HTTP request
	if err != nil {
		t.Logf("GitLab Api :%s err : %v", u, err)
		return
	}
	all, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Logf("GitLab ReadAll :%s err : %v", u, err)
		return
	}
	t.Log(string(all))
}

func TestGiteeCode(t *testing.T) {
	u := fmt.Sprintf("https://gitlab.com/login/oauth/authorize&client_id=%s", "102c5b2608655a5b7683")
	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, u, nil) //nolint:noctx // test-only HTTP request
	if err != nil {
		t.Logf("GitLab Api :%s err : %v", u, err)
		return
	}
	resp, err := http.DefaultClient.Do(req) //nolint:gosec // G107: test-only HTTP request
	if err != nil {
		t.Logf("GitLab Api :%s err : %v", u, err)
		return
	}
	all, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Logf("GitLab ReadAll :%s err : %v", u, err)
		return
	}
	t.Log(string(all))
}
