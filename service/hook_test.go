package service

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestParseHook_UnknownType(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/hook", strings.NewReader("{}")) //nolint:noctx // test request
	_, err := parseHook("unknown", req, "secret")
	if err == nil {
		t.Fatal("expected error for unknown hook type, got nil")
	}
	if !strings.Contains(err.Error(), "unknown webhook type") {
		t.Errorf("error = %q, want containing \"unknown webhook type\"", err.Error())
	}
}

func TestParseHook_UnknownType_Empty(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/hook", strings.NewReader("{}")) //nolint:noctx // test request
	_, err := parseHook("", req, "secret")
	if err == nil {
		t.Fatal("expected error for empty hook type, got nil")
	}
}

func TestParseHook_CaseInsensitive(t *testing.T) {
	// All these should NOT return "unknown webhook type" error
	// (they may fail for other reasons like missing headers, which is expected)
	types := []string{"gitee", "GITEE", "Gitee", "github", "GITHUB", "GitHub",
		"gitlab", "GITLAB", "GitLab", "gitea", "GITEA", "Gitea", "giteepremium", "GiteePremium"}
	for _, hookType := range types {
		t.Run(hookType, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/hook", strings.NewReader("{}")) //nolint:noctx // test request
			_, err := parseHook(hookType, req, "")
			if err != nil && strings.Contains(err.Error(), "unknown webhook type") {
				t.Errorf("parseHook(%q) should recognize this type, got: %v", hookType, err)
			}
		})
	}
}

func TestParseHook_GiteeRoutesToGiteeParser(t *testing.T) {
	// Without the proper Gitee event header, the parser should return an error
	// but NOT "unknown webhook type"
	req := httptest.NewRequest(http.MethodPost, "/hook", strings.NewReader("{}"))
	_, err := parseHook("gitee", req, "")
	if err != nil && strings.Contains(err.Error(), "unknown webhook type") {
		t.Errorf("gitee should not return unknown type error: %v", err)
	}
}

func TestParseHook_GiteePremiumRoutesToGiteeParser(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/hook", strings.NewReader("{}"))
	_, err := parseHook("giteepremium", req, "")
	if err != nil && strings.Contains(err.Error(), "unknown webhook type") {
		t.Errorf("giteepremium should not return unknown type error: %v", err)
	}
}

func TestParseHook_GithubRoutesToGithubParser(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/hook", strings.NewReader("{}"))
	_, err := parseHook("github", req, "")
	if err != nil && strings.Contains(err.Error(), "unknown webhook type") {
		t.Errorf("github should not return unknown type error: %v", err)
	}
}

func TestParseHook_GitlabRoutesToGitlabParser(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/hook", strings.NewReader("{}"))
	_, err := parseHook("gitlab", req, "")
	if err != nil && strings.Contains(err.Error(), "unknown webhook type") {
		t.Errorf("gitlab should not return unknown type error: %v", err)
	}
}

func TestParseHook_GiteaRoutesToGiteaParser(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/hook", strings.NewReader("{}"))
	_, err := parseHook("gitea", req, "")
	if err != nil && strings.Contains(err.Error(), "unknown webhook type") {
		t.Errorf("gitea should not return unknown type error: %v", err)
	}
}

func TestParseHook_ErrorMessagesIncludeType(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/hook", strings.NewReader("{}"))
	_, err := parseHook("bitbucket", req, "")
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "bitbucket") {
		t.Errorf("error should include the unknown type name, got: %q", err.Error())
	}
}
