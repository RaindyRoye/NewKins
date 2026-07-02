package gitlab

import (
	"crypto/hmac"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"hash"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gokins/gokins/hook"
	"github.com/gokins/gokins/util"
)

func Parse(req *http.Request, secret string) (wb hook.WebHook, err error) {
	defer util.RecoverResult(&err, "gitlab.Parse")
	data, err := io.ReadAll(
		io.LimitReader(req.Body, hook.MaxWebhookBodySize),
	)
	if err != nil {
		return nil, fmt.Errorf("gitlab: read request body: %w", err)
	}
	switch req.Header.Get(hook.GitlabEvent) {
	case hook.GitlabEventPush:
		wb, err = parsePushHook(data)
	case hook.GitlabEventNote:
		wb, err = parseCommentHook(data)
	case hook.GitlabEventPR:
		wb, err = parsePullRequestHook(data)
	default:
		return nil, fmt.Errorf("hook contains unknown header: %v", req.Header.Get(hook.GitlabEvent))
	}
	if err != nil {
		return nil, fmt.Errorf("gitlab: %w", err)
	}
	sig := req.Header.Get("X-Gitlab-Token")
	if secret != sig {
		return wb, errors.New("secret validation failed")
	}
	return wb, nil
}

func Validate(h func() hash.Hash, message, key []byte, signature string) bool {
	decoded, err := hex.DecodeString(signature)
	if err != nil {
		return false
	}
	return validate(h, message, key, decoded)
}

func validate(h func() hash.Hash, message, key, signature []byte) bool {
	mac := hmac.New(h, key)
	mac.Write(message)
	sum := mac.Sum(nil)
	return hmac.Equal(signature, sum)
}

func parseCommentHook(data []byte) (*hook.PullRequestCommentHook, error) {
	gp := new(gitlabCommentHook)
	err := json.Unmarshal(data, gp)
	if err != nil {
		return nil, fmt.Errorf("unmarshal gitlab comment hook: %w", err)
	}
	return convertCommentHook(gp)
}

func parsePullRequestHook(data []byte) (*hook.PullRequestHook, error) {
	gp := new(gitlabPRHook)
	err := json.Unmarshal(data, gp)
	if err != nil {
		return nil, fmt.Errorf("unmarshal gitlab PR hook: %w", err)
	}
	if gp.ObjectAttributes.Action != "" {
		if gp.ObjectAttributes.Action != hook.ActionOpen && gp.ObjectAttributes.Action != hook.ActionUpdate {
			return nil, fmt.Errorf("action is %v", gp.ObjectAttributes.Action)
		}
	}
	return convertPullRequestHook(gp), nil
}

func parsePushHook(data []byte) (*hook.PushHook, error) {
	gp := new(gitlabPushHook)
	err := json.Unmarshal(data, gp)
	if err != nil {
		return nil, fmt.Errorf("unmarshal gitlab push hook: %w", err)
	}
	return convertPushHook(gp), nil
}
func convertPullRequestHook(gp *gitlabPRHook) *hook.PullRequestHook {
	return &hook.PullRequestHook{
		Action: gp.ObjectAttributes.Action,
		Repo: hook.Repository{
			Ref:         gp.ObjectAttributes.SourceBranch,
			Sha:         gp.ObjectAttributes.LastCommit.Id,
			CloneURL:    gp.ObjectAttributes.Source.HttpUrl,
			Branch:      gp.ObjectAttributes.SourceBranch,
			Description: gp.ObjectAttributes.Source.Description,
			FullName:    gp.ObjectAttributes.Source.PathWithNamespace,
			GitHttpURL:  gp.ObjectAttributes.Source.GitHttpUrl,
			GitShhURL:   gp.ObjectAttributes.Source.SshUrl,
			GitURL:      gp.ObjectAttributes.Source.Url,
			HtmlURL:     gp.ObjectAttributes.Source.WebUrl,
			SshURL:      gp.ObjectAttributes.Source.SshUrl,
			Name:        gp.ObjectAttributes.Source.Name,
			URL:         gp.ObjectAttributes.Source.Url,
			Owner:       gp.User.Username,
			RepoType:    "gitlab",
			RepoOpenid:  strconv.Itoa(gp.Project.Id),
		},
		TargetRepo: hook.Repository{
			Ref:         gp.ObjectAttributes.TargetBranch,
			Sha:         gp.ObjectAttributes.LastCommit.Id,
			CloneURL:    gp.ObjectAttributes.Target.HttpUrl,
			Branch:      gp.ObjectAttributes.TargetBranch,
			Description: gp.ObjectAttributes.Target.Description,
			FullName:    gp.ObjectAttributes.Target.PathWithNamespace,
			GitHttpURL:  gp.ObjectAttributes.Target.GitHttpUrl,
			GitShhURL:   gp.ObjectAttributes.Target.SshUrl,
			GitURL:      gp.ObjectAttributes.Target.Url,
			HtmlURL:     gp.ObjectAttributes.Target.WebUrl,
			SshURL:      gp.ObjectAttributes.Target.SshUrl,
			Name:        gp.ObjectAttributes.Target.Name,
			URL:         gp.ObjectAttributes.Target.Url,
			Owner:       gp.User.Username,
			RepoType:    "gitlab",
			RepoOpenid:  strconv.Itoa(gp.Project.Id),
		},
		Sender: hook.User{
			UserName: gp.User.Username,
		},
		PullRequest: hook.PullRequest{
			Number: gp.ObjectAttributes.Iid,
			Title:  gp.ObjectAttributes.Title,
			Base: hook.Reference{
				Name: gp.ObjectAttributes.Source.Name,
				Sha:  gp.ObjectAttributes.LastCommit.Id,
			},
			Head: hook.Reference{
				Name: gp.ObjectAttributes.Target.Name,
				Sha:  gp.ObjectAttributes.LastCommit.Id,
			},
			Author: hook.User{
				UserName: gp.User.Username,
			},
			Created: time.Time{},
			Updated: time.Time{},
		},
	}
}
func convertPushHook(gp *gitlabPushHook) *hook.PushHook {
	branch := ""
	if gp.Ref != "" {
		if len(strings.Split(gp.Ref, "/")) > 2 {
			branch = strings.Split(gp.Ref, "/")[2]
		}
	}
	return &hook.PushHook{
		Ref: gp.Ref,
		Repo: hook.Repository{
			Ref:         gp.Ref,
			Sha:         gp.After,
			CloneURL:    gp.Project.HttpUrl,
			Branch:      branch,
			Description: gp.Repository.Description,
			FullName:    gp.Project.PathWithNamespace,
			GitHttpURL:  gp.Repository.GitHttpUrl,
			GitShhURL:   gp.Repository.GitSshUrl,
			GitURL:      gp.Repository.Url,
			SshURL:      gp.Project.SshUrl,
			Name:        gp.Repository.Name,
			URL:         gp.Repository.Url,
			Owner:       gp.UserUsername,
			RepoType:    "gitlab",
			RepoOpenid:  strconv.Itoa(gp.ProjectId),
		},
		Before: gp.Before,
		After:  gp.After,
		Sender: hook.User{
			UserName: gp.UserUsername,
		},
	}
}
func convertCommentHook(gp *gitlabCommentHook) (*hook.PullRequestCommentHook, error) {
	return &hook.PullRequestCommentHook{
		Action: hook.EventsTypeComment,
		Repo: hook.Repository{
			Ref:         gp.MergeRequest.SourceBranch,
			Sha:         gp.MergeRequest.LastCommit.Id,
			CloneURL:    gp.MergeRequest.Source.HttpUrl,
			Branch:      gp.MergeRequest.SourceBranch,
			Description: gp.MergeRequest.Source.Description,
			FullName:    gp.MergeRequest.Source.PathWithNamespace,
			GitHttpURL:  gp.MergeRequest.Source.GitHttpUrl,
			GitShhURL:   gp.MergeRequest.Source.SshUrl,
			GitURL:      gp.MergeRequest.Source.Url,
			HtmlURL:     gp.MergeRequest.Source.WebUrl,
			SshURL:      gp.MergeRequest.Source.SshUrl,
			Name:        gp.MergeRequest.Source.Name,
			URL:         gp.MergeRequest.Source.Url,
			Owner:       gp.User.Username,
			RepoType:    "gitlab",
			RepoOpenid:  strconv.Itoa(gp.Project.Id),
		},
		TargetRepo: hook.Repository{
			Ref:         gp.MergeRequest.TargetBranch,
			Sha:         gp.MergeRequest.LastCommit.Id,
			CloneURL:    gp.MergeRequest.Target.HttpUrl,
			Branch:      gp.MergeRequest.TargetBranch,
			Description: gp.MergeRequest.Target.Description,
			FullName:    gp.MergeRequest.Target.PathWithNamespace,
			GitHttpURL:  gp.MergeRequest.Target.GitHttpUrl,
			GitShhURL:   gp.MergeRequest.Target.SshUrl,
			GitURL:      gp.MergeRequest.Target.Url,
			HtmlURL:     gp.MergeRequest.Target.WebUrl,
			SshURL:      gp.MergeRequest.Target.SshUrl,
			Name:        gp.MergeRequest.Target.Name,
			URL:         gp.MergeRequest.Target.Url,
			Owner:       gp.User.Username,
			RepoType:    "gitlab",
			RepoOpenid:  strconv.Itoa(gp.Project.Id),
		},
		Comment: hook.Comment{
			Body: gp.ObjectAttributes.Note,
			Author: hook.User{
				UserName: gp.User.Username,
			},
		},
		PullRequest: hook.PullRequest{
			Number: gp.MergeRequest.Iid,
			Title:  gp.MergeRequest.Title,
			Base: hook.Reference{
				Name: gp.MergeRequest.Source.Name,
				Sha:  gp.MergeRequest.LastCommit.Id,
			},
			Head: hook.Reference{
				Name: gp.MergeRequest.Target.Name,
				Sha:  gp.MergeRequest.LastCommit.Id,
			},
			Author: hook.User{
				UserName: gp.User.Username,
			},
			Created: time.Time{},
			Updated: time.Time{},
		},
		Sender: hook.User{
			UserName: gp.User.Username,
		},
	}, nil
}

type gitlabPushHook struct {
	ObjectKind   string `json:"object_kind"`
	EventName    string `json:"event_name"`
	Before       string `json:"before"`
	After        string `json:"after"`
	Ref          string `json:"ref"`
	CheckoutSha  string `json:"checkout_sha"`
	Message      any    `json:"message"`
	UserId       int    `json:"user_id"`
	UserName     string `json:"user_name"`
	UserUsername string `json:"user_username"`
	UserEmail    string `json:"user_email"`
	UserAvatar   string `json:"user_avatar"`
	ProjectId    int    `json:"project_id"`
	Project      struct {
		Id                int    `json:"id"`
		Name              string `json:"name"`
		Description       string `json:"description"`
		WebUrl            string `json:"web_url"`
		AvatarUrl         any    `json:"avatar_url"`
		GitSshUrl         string `json:"git_ssh_url"`
		GitHttpUrl        string `json:"git_http_url"`
		Namespace         string `json:"namespace"`
		VisibilityLevel   int    `json:"visibility_level"`
		PathWithNamespace string `json:"path_with_namespace"`
		DefaultBranch     string `json:"default_branch"`
		CiConfigPath      string `json:"ci_config_path"`
		Homepage          string `json:"homepage"`
		Url               string `json:"url"`
		SshUrl            string `json:"ssh_url"`
		HttpUrl           string `json:"http_url"`
	} `json:"project"`
	Commits []struct {
		Id        string    `json:"id"`
		Message   string    `json:"message"`
		Title     string    `json:"title"`
		Timestamp time.Time `json:"timestamp"`
		Url       string    `json:"url"`
		Author    struct {
			Name  string `json:"name"`
			Email string `json:"email"`
		} `json:"author"`
		Added    []any    `json:"added"`
		Modified []string `json:"modified"`
		Removed  []any    `json:"removed"`
	} `json:"commits"`
	TotalCommitsCount int `json:"total_commits_count"`
	PushOptions       struct {
	} `json:"push_options"`
	Repository struct {
		Name            string `json:"name"`
		Url             string `json:"url"`
		Description     string `json:"description"`
		Homepage        string `json:"homepage"`
		GitHttpUrl      string `json:"git_http_url"`
		GitSshUrl       string `json:"git_ssh_url"`
		VisibilityLevel int    `json:"visibility_level"`
	} `json:"repository"`
}
type gitlabPRHook struct {
	ObjectKind string `json:"object_kind"`
	EventType  string `json:"event_type"`
	User       struct {
		Id        int    `json:"id"`
		Name      string `json:"name"`
		Username  string `json:"username"`
		AvatarUrl string `json:"avatar_url"`
		Email     string `json:"email"`
	} `json:"user"`
	Project struct {
		Id                int    `json:"id"`
		Name              string `json:"name"`
		Description       string `json:"description"`
		WebUrl            string `json:"web_url"`
		AvatarUrl         any    `json:"avatar_url"`
		GitSshUrl         string `json:"git_ssh_url"`
		GitHttpUrl        string `json:"git_http_url"`
		Namespace         string `json:"namespace"`
		VisibilityLevel   int    `json:"visibility_level"`
		PathWithNamespace string `json:"path_with_namespace"`
		DefaultBranch     string `json:"default_branch"`
		CiConfigPath      string `json:"ci_config_path"`
		Homepage          string `json:"homepage"`
		Url               string `json:"url"`
		SshUrl            string `json:"ssh_url"`
		HttpUrl           string `json:"http_url"`
	} `json:"project"`
	ObjectAttributes struct {
		AssigneeId     any    `json:"assignee_id"`
		AuthorId       int    `json:"author_id"`
		CreatedAt      string `json:"created_at"`
		Description    string `json:"description"`
		HeadPipelineId any    `json:"head_pipeline_id"`
		Id             int    `json:"id"`
		Iid            int64  `json:"iid"`
		LastEditedAt   any    `json:"last_edited_at"`
		LastEditedById any    `json:"last_edited_by_id"`
		MergeCommitSha any    `json:"merge_commit_sha"`
		MergeError     any    `json:"merge_error"`
		MergeParams    struct {
			ForceRemoveSourceBranch string `json:"force_remove_source_branch"`
		} `json:"merge_params"`
		MergeStatus               string `json:"merge_status"`
		MergeUserId               any    `json:"merge_user_id"`
		MergeWhenPipelineSucceeds bool   `json:"merge_when_pipeline_succeeds"`
		MilestoneId               any    `json:"milestone_id"`
		SourceBranch              string `json:"source_branch"`
		SourceProjectId           int    `json:"source_project_id"`
		StateId                   int    `json:"state_id"`
		TargetBranch              string `json:"target_branch"`
		TargetProjectId           int    `json:"target_project_id"`
		TimeEstimate              int    `json:"time_estimate"`
		Title                     string `json:"title"`
		UpdatedAt                 string `json:"updated_at"`
		UpdatedById               any    `json:"updated_by_id"`
		Url                       string `json:"url"`
		Source                    struct {
			Id                int    `json:"id"`
			Name              string `json:"name"`
			Description       string `json:"description"`
			WebUrl            string `json:"web_url"`
			AvatarUrl         any    `json:"avatar_url"`
			GitSshUrl         string `json:"git_ssh_url"`
			GitHttpUrl        string `json:"git_http_url"`
			Namespace         string `json:"namespace"`
			VisibilityLevel   int    `json:"visibility_level"`
			PathWithNamespace string `json:"path_with_namespace"`
			DefaultBranch     string `json:"default_branch"`
			CiConfigPath      string `json:"ci_config_path"`
			Homepage          string `json:"homepage"`
			Url               string `json:"url"`
			SshUrl            string `json:"ssh_url"`
			HttpUrl           string `json:"http_url"`
		} `json:"source"`
		Target struct {
			Id                int    `json:"id"`
			Name              string `json:"name"`
			Description       string `json:"description"`
			WebUrl            string `json:"web_url"`
			AvatarUrl         any    `json:"avatar_url"`
			GitSshUrl         string `json:"git_ssh_url"`
			GitHttpUrl        string `json:"git_http_url"`
			Namespace         string `json:"namespace"`
			VisibilityLevel   int    `json:"visibility_level"`
			PathWithNamespace string `json:"path_with_namespace"`
			DefaultBranch     string `json:"default_branch"`
			CiConfigPath      string `json:"ci_config_path"`
			Homepage          string `json:"homepage"`
			Url               string `json:"url"`
			SshUrl            string `json:"ssh_url"`
			HttpUrl           string `json:"http_url"`
		} `json:"target"`
		LastCommit struct {
			Id        string    `json:"id"`
			Message   string    `json:"message"`
			Title     string    `json:"title"`
			Timestamp time.Time `json:"timestamp"`
			Url       string    `json:"url"`
			Author    struct {
				Name  string `json:"name"`
				Email string `json:"email"`
			} `json:"author"`
		} `json:"last_commit"`
		WorkInProgress      bool   `json:"work_in_progress"`
		TotalTimeSpent      int    `json:"total_time_spent"`
		TimeChange          int    `json:"time_change"`
		HumanTotalTimeSpent any    `json:"human_total_time_spent"`
		HumanTimeChange     any    `json:"human_time_change"`
		HumanTimeEstimate   any    `json:"human_time_estimate"`
		AssigneeIds         []any  `json:"assignee_ids"`
		State               string `json:"state"`
		Action              string `json:"action"`
	} `json:"object_attributes"`
	Labels  []any `json:"labels"`
	Changes struct {
		MergeStatus struct {
			Previous string `json:"previous"`
			Current  string `json:"current"`
		} `json:"merge_status"`
	} `json:"changes"`
	Repository struct {
		Name        string `json:"name"`
		Url         string `json:"url"`
		Description string `json:"description"`
		Homepage    string `json:"homepage"`
	} `json:"repository"`
}
type gitlabCommentHook struct {
	ObjectKind string `json:"object_kind"`
	EventType  string `json:"event_type"`
	User       struct {
		Id        int    `json:"id"`
		Name      string `json:"name"`
		Username  string `json:"username"`
		AvatarUrl string `json:"avatar_url"`
		Email     string `json:"email"`
	} `json:"user"`
	ProjectId int `json:"project_id"`
	Project   struct {
		Id                int    `json:"id"`
		Name              string `json:"name"`
		Description       string `json:"description"`
		WebUrl            string `json:"web_url"`
		AvatarUrl         any    `json:"avatar_url"`
		GitSshUrl         string `json:"git_ssh_url"`
		GitHttpUrl        string `json:"git_http_url"`
		Namespace         string `json:"namespace"`
		VisibilityLevel   int    `json:"visibility_level"`
		PathWithNamespace string `json:"path_with_namespace"`
		DefaultBranch     string `json:"default_branch"`
		CiConfigPath      string `json:"ci_config_path"`
		Homepage          string `json:"homepage"`
		Url               string `json:"url"`
		SshUrl            string `json:"ssh_url"`
		HttpUrl           string `json:"http_url"`
	} `json:"project"`
	ObjectAttributes struct {
		Attachment       any    `json:"attachment"`
		AuthorId         int    `json:"author_id"`
		ChangePosition   any    `json:"change_position"`
		CommitId         any    `json:"commit_id"`
		CreatedAt        string `json:"created_at"`
		DiscussionId     string `json:"discussion_id"`
		Id               int    `json:"id"`
		LineCode         any    `json:"line_code"`
		Note             string `json:"note"`
		NoteableId       int    `json:"noteable_id"`   //nolint:misspell // GitLab API uses "noteable"
		NoteableType     string `json:"noteable_type"` //nolint:misspell // GitLab API uses "noteable"
		OriginalPosition any    `json:"original_position"`
		Position         any    `json:"position"`
		ProjectId        int    `json:"project_id"`
		ResolvedAt       any    `json:"resolved_at"`
		ResolvedById     any    `json:"resolved_by_id"`
		ResolvedByPush   any    `json:"resolved_by_push"`
		StDiff           any    `json:"st_diff"`
		System           bool   `json:"system"`
		Type             any    `json:"type"`
		UpdatedAt        string `json:"updated_at"`
		UpdatedById      any    `json:"updated_by_id"`
		Description      string `json:"description"`
		Url              string `json:"url"`
	} `json:"object_attributes"`
	Repository struct {
		Name        string `json:"name"`
		Url         string `json:"url"`
		Description string `json:"description"`
		Homepage    string `json:"homepage"`
	} `json:"repository"`
	MergeRequest struct {
		AssigneeId     any    `json:"assignee_id"`
		AuthorId       int    `json:"author_id"`
		CreatedAt      string `json:"created_at"`
		Description    string `json:"description"`
		HeadPipelineId any    `json:"head_pipeline_id"`
		Id             int    `json:"id"`
		Iid            int64  `json:"iid"`
		LastEditedAt   any    `json:"last_edited_at"`
		LastEditedById any    `json:"last_edited_by_id"`
		MergeCommitSha any    `json:"merge_commit_sha"`
		MergeError     any    `json:"merge_error"`
		MergeParams    struct {
			ForceRemoveSourceBranch string `json:"force_remove_source_branch"`
		} `json:"merge_params"`
		MergeStatus               string `json:"merge_status"`
		MergeUserId               any    `json:"merge_user_id"`
		MergeWhenPipelineSucceeds bool   `json:"merge_when_pipeline_succeeds"`
		MilestoneId               any    `json:"milestone_id"`
		SourceBranch              string `json:"source_branch"`
		SourceProjectId           int    `json:"source_project_id"`
		StateId                   int    `json:"state_id"`
		TargetBranch              string `json:"target_branch"`
		TargetProjectId           int    `json:"target_project_id"`
		TimeEstimate              int    `json:"time_estimate"`
		Title                     string `json:"title"`
		UpdatedAt                 string `json:"updated_at"`
		UpdatedById               any    `json:"updated_by_id"`
		Url                       string `json:"url"`
		Source                    struct {
			Id                int    `json:"id"`
			Name              string `json:"name"`
			Description       string `json:"description"`
			WebUrl            string `json:"web_url"`
			AvatarUrl         any    `json:"avatar_url"`
			GitSshUrl         string `json:"git_ssh_url"`
			GitHttpUrl        string `json:"git_http_url"`
			Namespace         string `json:"namespace"`
			VisibilityLevel   int    `json:"visibility_level"`
			PathWithNamespace string `json:"path_with_namespace"`
			DefaultBranch     string `json:"default_branch"`
			CiConfigPath      string `json:"ci_config_path"`
			Homepage          string `json:"homepage"`
			Url               string `json:"url"`
			SshUrl            string `json:"ssh_url"`
			HttpUrl           string `json:"http_url"`
		} `json:"source"`
		Target struct {
			Id                int    `json:"id"`
			Name              string `json:"name"`
			Description       string `json:"description"`
			WebUrl            string `json:"web_url"`
			AvatarUrl         any    `json:"avatar_url"`
			GitSshUrl         string `json:"git_ssh_url"`
			GitHttpUrl        string `json:"git_http_url"`
			Namespace         string `json:"namespace"`
			VisibilityLevel   int    `json:"visibility_level"`
			PathWithNamespace string `json:"path_with_namespace"`
			DefaultBranch     string `json:"default_branch"`
			CiConfigPath      string `json:"ci_config_path"`
			Homepage          string `json:"homepage"`
			Url               string `json:"url"`
			SshUrl            string `json:"ssh_url"`
			HttpUrl           string `json:"http_url"`
		} `json:"target"`
		LastCommit struct {
			Id        string    `json:"id"`
			Message   string    `json:"message"`
			Title     string    `json:"title"`
			Timestamp time.Time `json:"timestamp"`
			Url       string    `json:"url"`
			Author    struct {
				Name  string `json:"name"`
				Email string `json:"email"`
			} `json:"author"`
		} `json:"last_commit"`
		WorkInProgress      bool   `json:"work_in_progress"`
		TotalTimeSpent      int    `json:"total_time_spent"`
		TimeChange          int    `json:"time_change"`
		HumanTotalTimeSpent any    `json:"human_total_time_spent"`
		HumanTimeChange     any    `json:"human_time_change"`
		HumanTimeEstimate   any    `json:"human_time_estimate"`
		AssigneeIds         []any  `json:"assignee_ids"`
		State               string `json:"state"`
	} `json:"merge_request"`
}
