package giteaapi

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"testing"
)

func TestGiteaRepos(t *testing.T) {
	u := fmt.Sprintf(ApiGiteaGetRepos, 1, 10)
	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, u, nil)
	if err != nil {
		t.Logf("Gitea Api :%s err : %v", u, err)
		return
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Logf("Gitea Api :%s err : %v", u, err)
		return
	}
	all, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Logf("Gitea ReadAll :%s err : %v", u, err)
		return
	}
	t.Log(string(all))
}

func TestGiteeCode(t *testing.T) {
	u := fmt.Sprintf("https://gitea.com/login/oauth/authorize&client_id=%s", "102c5b2608655a5b7683")
	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, u, nil)
	if err != nil {
		t.Logf("Gitea Api :%s err : %v", u, err)
		return
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Logf("Gitea Api :%s err : %v", u, err)
		return
	}
	all, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Logf("Gitea ReadAll :%s err : %v", u, err)
		return
	}
	t.Log(string(all))
}
