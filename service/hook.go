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

// triggerRunRecorder records the outcome of a trigger execution to the database.
// It is used by all trigger types (webhook, web, timer) to persist execution results.
type triggerRunRecorder struct {
	ctx           context.Context
	triggerId     string
	pipeVersionId string
	infos         string
	err           error
}

// record saves the trigger run to the database with the recorded outcome.
func (r *triggerRunRecorder) record() {
	ttr := &model.TTriggerRun{
		Id:            utils.NewXid(),
		Tid:           r.triggerId,
		PipeVersionId: r.pipeVersionId,
		Created:       time.Now(),
	}
	if r.err != nil {
		ttr.Error = r.err.Error()
	}
	if r.infos != "" {
		ttr.Infos = r.infos
	}
	if _, err := comm.Db.Context(r.ctx).InsertOne(ttr); err != nil {
		logrus.Errorf("trigger run record (trigger=%s): %v", r.triggerId, err)
	}
}

// parseTriggerParams validates and unmarshals trigger parameters from JSON.
// Returns an error if params are empty or malformed.
func parseTriggerParams(triggerId, params string) (map[string]string, error) {
	if params == "" {
		return nil, ErrTriggerNoParams
	}
	m := make(map[string]string)
	if err := json.Unmarshal([]byte(params), &m); err != nil {
		return nil, fmt.Errorf("unmarshal trigger params (trigger=%s): %w", triggerId, err)
	}
	return m, nil
}

// executeTrigger runs a pipeline for a trigger and records the execution.
// It handles permission checking, parameter parsing, and pipeline execution.
func executeTrigger(ctx context.Context, tt *model.TTrigger, sha, event, infos string) (*runtime.Build, error) {
	rec := &triggerRunRecorder{
		ctx:       ctx,
		triggerId: tt.Id,
		infos:     infos,
	}
	defer func() {
		if rec.err != nil {
			rec.record()
		}
	}()

	if err := TriggerPermCtx(ctx, tt); err != nil {
		rec.err = fmt.Errorf("check trigger permissions (trigger=%s): %w", tt.Id, err)
		return nil, rec.err
	}

	tvp, rb, err := Run(ctx, tt.Uid, tt.PipelineId, sha, event)
	if err != nil {
		rec.err = fmt.Errorf("run pipeline (trigger=%s, pipeline=%s): %w", tt.Id, tt.PipelineId, err)
		return nil, rec.err
	}
	rec.pipeVersionId = tvp.Id
	rec.record()
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
	m, err := parseTriggerParams(tt.Id, tt.Params)
	if err != nil {
		return nil, err
	}

	hookType, ok := m["hookType"]
	if !ok {
		return nil, ErrHookTypeEmpty
	}

	secret := m["secret"]
	event := m["event"]
	branch := m["branch"]

	h, err := parseHook(hookType, req, secret)
	if err != nil {
		return nil, fmt.Errorf("parse webhook (type=%s): %w", hookType, err)
	}

	var sha string
	var events string
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

	branchs := h.Repository().Branch
	bts, _ := json.Marshal(h)
	infos := string(bts)

	if event != "" && event != events {
		return nil, ErrWebhookEventMismatch
	}
	if branch != "" && branch != branchs {
		return nil, ErrBranchMismatch
	}

	rb, err = executeTrigger(ctx, tt, sha, "webHook", infos)
	return rb, err
}

// TriggerWeb processes a web trigger with secret validation.
func TriggerWeb(ctx context.Context, tt *model.TTrigger, secret string) (rb *runtime.Build, err error) {
	m, err := parseTriggerParams(tt.Id, tt.Params)
	if err != nil {
		return nil, err
	}

	pSecret, ok := m["secret"]
	if !ok {
		return nil, ErrTriggerNoSecret
	}
	if secret != pSecret {
		return nil, ErrTriggerSecretMismatch
	}

	branch := m["branch"]
	rb, err = executeTrigger(ctx, tt, branch, "web", "{}")
	return rb, err
}

// TriggerTimer processes a timer-triggered pipeline execution.
func TriggerTimer(ctx context.Context, tt *model.TTrigger) (rb *runtime.Build, err error) {
	rb, err = executeTrigger(ctx, tt, "", "timer", "{}")
	return rb, err
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
