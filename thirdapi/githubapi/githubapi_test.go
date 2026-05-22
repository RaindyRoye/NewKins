package githubapi

import (
	"fmt"
	"io"
	"net/http"
	"testing"
)

func TestGithubRepos(t *testing.T) {
	u := fmt.Sprintf(ApiGithubGetRepos, "all", "full_name", "desc", "1", "10")
	resp, err := http.Get(u)
	if err != nil {
		fmt.Println(fmt.Errorf("Gitee Api :%v err : %v", u, err))
		return
	}
	all, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Println(fmt.Errorf("Gitee ReadAll :%v err : %v", u, err))
		return
	}
	fmt.Println(string(all))
}

func TestGiteeCode(t *testing.T) {
	u := fmt.Sprintf("https://github.com/login/oauth/authorize&client_id=%s", "102c5b2608655a5b7683")
	resp, err := http.Get(u)
	if err != nil {
		fmt.Println(fmt.Errorf("Gitee Api :%v err : %v", u, err))
		return
	}
	all, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Println(fmt.Errorf("Gitee ReadAll :%v err : %v", u, err))
		return
	}
	fmt.Println(string(all))
}
