package githubapi

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"testing"
)

func TestGithubRepos(t *testing.T) {
	u := fmt.Sprintf(ApiGithubGetRepos, "all", "full_name", "desc", "1", "10")
	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, u, nil) //nolint:noctx // test-only HTTP request
	if err != nil {
		fmt.Println(fmt.Errorf("GitHub Api :%v err : %v", u, err))
		return
	}
	resp, err := http.DefaultClient.Do(req) //nolint:gosec // G107: test-only HTTP request
	if err != nil {
		fmt.Println(fmt.Errorf("GitHub Api :%v err : %v", u, err))
		return
	}
	all, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Println(fmt.Errorf("GitHub ReadAll :%v err : %v", u, err))
		return
	}
	fmt.Println(string(all))
}

func TestGiteeCode(t *testing.T) {
	u := fmt.Sprintf("https://github.com/login/oauth/authorize&client_id=%s", "102c5b2608655a5b7683")
	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, u, nil) //nolint:noctx // test-only HTTP request
	if err != nil {
		fmt.Println(fmt.Errorf("GitHub Api :%v err : %v", u, err))
		return
	}
	resp, err := http.DefaultClient.Do(req) //nolint:gosec // G107: test-only HTTP request
	if err != nil {
		fmt.Println(fmt.Errorf("GitHub Api :%v err : %v", u, err))
		return
	}
	all, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Println(fmt.Errorf("GitHub ReadAll :%v err : %v", u, err))
		return
	}
	fmt.Println(string(all))
}
