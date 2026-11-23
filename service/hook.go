package service

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gokins/core/runtime"
	"github.com/gokins/core/utils"
	"github.com/gokins/gokins/comm"
	"github.com/gokins/gokins/hook"
	"github.com/gokins/gokins/hook/gitea"
	"github.com/gokins/gokins/hook/gitee"
	"github.com/gokins/gokins/hook/github"
	"github.com/gokins/gokins/hook/gitlab"
	"github.com/gokins/gokins/model"
	"github.com/sirupsen/logrus"
)

// triggerResult captures the outcome of a trigger execution for logging purposes.
type triggerResult struct {
	versionId string
	infos     string
}

// executeTrigger is the common helper that:
// 1. Checks trigger permissions
// 2. Runs the pipeline
// 3. Records the trigger run in the database
//
// The caller provides a fn that performs the actual trigger-specific logic
// (parsing webhook, extracting sha/event/branch, etc.) and returns the
// build result along with version ID and info string for logging.
func executeTrigger(ctx context.Context, tt *model.TTrigger, fn func() (versionId, infos string, rb *runtime.Build, err error)) (rb *runtime.Build, err error) {
	res := triggerResult{infos: "{}"}
	defer func() {
		if comm.Db == nil {
			return
		}
		ttr := &model.TTriggerRun{
			Id:            utils.NewXid(),
			Tid:           tt.Id,
			PipeVersionId: res.versionId,
			Infos:         res.infos,
			Created:       time.Now(),
		}
		if err != nil {
			ttr.Error = err.Error()
		}
		if _, dbErr := comm.Db.Context(ctx).InsertOne(ttr); dbErr != nil {
			logrus.Errorf("executeTrigger: failed to save trigger run (trigger=%s): %v", tt.Id, dbErr)
		}
	}()

	if tt.Params == "" {
		return nil, ErrTriggerNoParams
	}
	if permErr := TriggerPermCtx(ctx, tt); permErr != nil {
		return nil, fmt.Errorf("check trigger permissions (trigger=%s): %w", tt.Id, permErr)
	}

	var versionId, infos string
	versionId, infos, rb, err = fn()
	if err != nil {
		return nil, err
	}
	res.versionId = versionId
	if infos != "" {
		res.infos = infos
	}
	return rb, nil
}

// TriggerHook processes a webhook trigger using the request's context.
// Prefer TriggerHookCtx when a request context is available explicitly.
func TriggerHook(tt *model.TTrigger, req *http.Request) (rb *runtime.Build, err error) {
	return TriggerHookCtx(req.Context(), tt, req)
}

// TriggerHookCtx processes a webhook trigger with an explicit context for
// database operations and cancellation. The req parameter is used to parse
// the webhook payload from the HTTP request body and headers.
func TriggerHookCtx(ctx context.Context, tt *model.TTrigger, req *http.Request) (rb *runtime.Build, err error) {
	return executeTrigger(ctx, tt, func() (string, string, *runtime.Build, error) {
		m := map[string]string{}
		if err := json.Unmarshal([]byte(tt.Params), &m); err != nil {
			return "", "{}", nil, fmt.Errorf("unmarshal trigger params (trigger=%s): %w", tt.Id, err)
		}
		hookType, ok := m["hookType"]
		if !ok {
			return "", "{}", nil, ErrHookTypeEmpty
		}
		secret := m["secret"]
		event := m["event"]
		branch := m["branch"]

		h, err := parseHook(hookType, req, secret)
		if err != nil {
			return "", "{}", nil, fmt.Errorf("parse webhook (type=%s): %w", hookType, err)
		}

		var sha, events string
		switch c := h.(type) {
		case *hook.PullRequestHook:
			events = "pr"
			sha = c.PullRequest.Base.Sha
		case *hook.PullRequestCommentHook:
			events = "comment"
			sha = c.PullRequest.Base.Sha
		case *hook.PushHook:
			events = "push"
			sha = c.After
		default:
			return "", "{}", nil, ErrWebhookParseFailed
		}

		branchs := h.Repository().Branch
		bts, _ := json.Marshal(h)
		infos := string(bts)

		if event != "" && event != events {
			return "", infos, nil, ErrWebhookEventMismatch
		}
		if branch != "" && branch != branchs {
			return "", infos, nil, ErrBranchMismatch
		}

		tvp, rb, err := Run(ctx, tt.Uid, tt.PipelineId, sha, "webHook")
		if err != nil {
			return "", infos, nil, fmt.Errorf("run pipeline (trigger=%s, pipeline=%s): %w", tt.Id, tt.PipelineId, err)
		}
		return tvp.Id, infos, rb, nil
	})
}

// TriggerWeb handles a web-triggered pipeline execution.
func TriggerWeb(ctx context.Context, tt *model.TTrigger, secret string) (rb *runtime.Build, err error) {
	return executeTrigger(ctx, tt, func() (string, string, *runtime.Build, error) {
		m := map[string]string{}
		if err := json.Unmarshal([]byte(tt.Params), &m); err != nil {
			return "", "{}", nil, fmt.Errorf("unmarshal trigger params (trigger=%s): %w", tt.Id, err)
		}
		pSecret, ok := m["secret"]
		if !ok {
			return "", "{}", nil, ErrTriggerNoSecret
		}
		if secret != pSecret {
			return "", "{}", nil, ErrTriggerSecretMismatch
		}
		branch := m["branch"]

		tvp, rb, err := Run(ctx, tt.Uid, tt.PipelineId, branch, "web")
		if err != nil {
			return "", "{}", nil, fmt.Errorf("run pipeline (trigger=%s, pipeline=%s): %w", tt.Id, tt.PipelineId, err)
		}
		return tvp.Id, "{}", rb, nil
	})
}

// TriggerTimer handles a timer-triggered pipeline execution.
func TriggerTimer(ctx context.Context, tt *model.TTrigger) (rb *runtime.Build, err error) {
	return executeTrigger(ctx, tt, func() (string, string, *runtime.Build, error) {
		tvp, rb, err := Run(ctx, tt.Uid, tt.PipelineId, "", "timer")
		if err != nil {
			return "", "{}", nil, fmt.Errorf("run pipeline (trigger=%s, pipeline=%s): %w", tt.Id, tt.PipelineId, err)
		}
		return tvp.Id, "{}", rb, nil
	})
}

func parseHook(hookType string, req *http.Request, secret string) (hook.WebHook, error) {
	switch strings.ToLower(hookType) {
	case "gitee", "giteepremium":
		return gitee.Parse(req, secret)
	case "github":
		return github.Parse(req, secret)
	case "gitlab":
		return gitlab.Parse(req, secret)
	case "gitea":
		return gitea.Parse(req, secret)
	default:
		return nil, fmt.Errorf("%w: %q", ErrUnknownWebhookType, hookType)
	}
}
