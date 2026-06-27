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

// TriggerHook processes a webhook trigger using the request's context.
// Prefer TriggerHookCtx when a request context is available explicitly.
func TriggerHook(tt *model.TTrigger, req *http.Request) (rb *runtime.Build, err error) {
	return TriggerHookCtx(req.Context(), tt, req)
}

// TriggerHookCtx processes a webhook trigger with an explicit context for
// database operations and cancellation. The req parameter is used to parse
// the webhook payload from the HTTP request body and headers.
func TriggerHookCtx(ctx context.Context, tt *model.TTrigger, req *http.Request) (rb *runtime.Build, err error) {
	tvpId := ""
	infos := "{}"
	defer func() {
		ttr := &model.TTriggerRun{
			Id:            utils.NewXid(),
			Tid:           tt.Id,
			PipeVersionId: tvpId,
			Created:       time.Now(),
		}
		if err != nil {
			ttr.Error = err.Error()
		}
		if infos != "" {
			ttr.Infos = infos
		}
		if _, err := comm.Db.Context(ctx).InsertOne(ttr); err != nil {
			logrus.Errorf("TriggerHookCtx: failed to save trigger run: %v", err)
		}
	}()
	if tt.Params == "" {
		return nil, ErrTriggerNoParams
	}
	err = TriggerPermCtx(ctx, tt)
	if err != nil {
		return nil, fmt.Errorf("check trigger permissions (trigger=%s): %w", tt.Id, err)
	}
	m := map[string]string{}
	err = json.Unmarshal([]byte(tt.Params), &m)
	if err != nil {
		return nil, fmt.Errorf("unmarshal trigger params (trigger=%s): %w", tt.Id, err)
	}
	hookType, ok := m["hookType"]
	if !ok {
		return nil, ErrHookTypeEmpty
	}
	secret := ""
	if s, ok := m["secret"]; ok {
		secret = s
	}
	event := ""
	if s, ok := m["event"]; ok {
		event = s
	}
	branch := ""
	if s, ok := m["branch"]; ok {
		branch = s
	}
	h, err := parseHook(hookType, req, secret)
	if err != nil {
		return nil, fmt.Errorf("parse webhook (type=%s): %w", hookType, err)
	}
	var sha string
	var events string
	var branchs string
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
		return nil, ErrWebhookParseFailed
	}
	branchs = h.Repository().Branch
	bts, _ := json.Marshal(h)
	infos = string(bts)
	if event != "" && event != events {
		return nil, ErrWebhookEventMismatch
	}
	if branch != "" && branch != branchs {
		return nil, ErrBranchMismatch
	}

	tvp, rb, err := Run(ctx, tt.Uid, tt.PipelineId, sha, "webHook")
	if err != nil {
		return nil, fmt.Errorf("run pipeline (trigger=%s, pipeline=%s): %w", tt.Id, tt.PipelineId, err)
	}
	tvpId = tvp.Id
	return rb, nil
}

func TriggerWeb(ctx context.Context, tt *model.TTrigger, secret string) (rb *runtime.Build, err error) {
	tvpId := ""
	defer func() {
		ttr := &model.TTriggerRun{
			Id:            utils.NewXid(),
			Tid:           tt.Id,
			PipeVersionId: tvpId,
			Infos:         "{}",
			Created:       time.Now(),
		}
		if err != nil {
			ttr.Error = err.Error()
		}
		if _, err := comm.Db.Context(ctx).InsertOne(ttr); err != nil {
			logrus.Errorf("TriggerWeb: failed to save trigger run: %v", err)
		}
	}()
	if tt.Params == "" {
		return nil, ErrTriggerNoParams
	}
	err = TriggerPermCtx(ctx, tt)
	if err != nil {
		return nil, fmt.Errorf("check trigger permissions (trigger=%s): %w", tt.Id, err)
	}
	m := map[string]string{}
	err = json.Unmarshal([]byte(tt.Params), &m)
	if err != nil {
		return nil, fmt.Errorf("unmarshal trigger params (trigger=%s): %w", tt.Id, err)
	}
	pSecret, ok := m["secret"]
	if !ok {
		return nil, ErrTriggerNoSecret
	}
	if secret != pSecret {
		return nil, ErrTriggerSecretMismatch
	}
	branch := ""
	if s, ok := m["branch"]; ok {
		branch = s
	}
	tvp, rb, err := Run(ctx, tt.Uid, tt.PipelineId, branch, "web")
	if err != nil {
		return nil, fmt.Errorf("run pipeline (trigger=%s, pipeline=%s): %w", tt.Id, tt.PipelineId, err)
	}
	tvpId = tvp.Id
	return rb, nil
}
func TriggerTimer(ctx context.Context, tt *model.TTrigger) (rb *runtime.Build, err error) {
	ttr := &model.TTriggerRun{
		Id:      utils.NewXid(),
		Tid:     tt.Id,
		Created: time.Now(),
		Infos:   "{}",
	}
	defer func() {
		if err != nil {
			ttr.Error = err.Error()
		}
		if _, err := comm.Db.Context(ctx).InsertOne(ttr); err != nil {
			logrus.Errorf("TriggerTimer: failed to save trigger run: %v", err)
		}
	}()
	err = TriggerPermCtx(ctx, tt)
	if err != nil {
		return nil, fmt.Errorf("check trigger permissions (trigger=%s): %w", tt.Id, err)
	}
	tvp, rb, err := Run(ctx, tt.Uid, tt.PipelineId, "", "timer")
	if err != nil {
		return nil, fmt.Errorf("run pipeline (trigger=%s, pipeline=%s): %w", tt.Id, tt.PipelineId, err)
	}
	ttr.PipeVersionId = tvp.Id
	return rb, nil
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
		return nil, fmt.Errorf("unknown webhook type: %q", hookType)
	}
}
