package service

import (
	"encoding/json"
	"errors"
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

func TriggerHook(tt *model.TTrigger, req *http.Request) (rb *runtime.Build, err error) {
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
		if _, err := comm.Db.InsertOne(ttr); err != nil {
			logrus.Errorf("TriggerHook: failed to save trigger run: %v", err)
		}
	}()
	if tt.Params == "" {
		return nil, errors.New("触发器没有配置参数")
	}
	err = TriggerPerm(tt)
	if err != nil {
		return nil, err
	}
	m := map[string]string{}
	err = json.Unmarshal([]byte(tt.Params), &m)
	if err != nil {
		err = fmt.Errorf("触发器配置参数错误: %w", err)
		return nil, err
	}
	hookType, ok := m["hookType"]
	if !ok {
		err = errors.New("hookType为空")
		return nil, err
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
		return nil, err
	}
	sha := ""
	events := ""
	branchs := ""
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
		return nil, errors.New("webhook解析失败")
	}
	branchs = h.Repository().Branch
	bts, _ := json.Marshal(h)
	infos = string(bts)
	if event != "" && event != events {
		return nil, errors.New("webhook事件不匹配")
	}
	if branch != "" && branch != branchs {
		return nil, errors.New("分支不匹配")
	}

	tvp, rb, err := Run(tt.Uid, tt.PipelineId, sha, "webHook")
	if err != nil {
		return nil, err
	}
	tvpId = tvp.Id
	return rb, nil
}

func TriggerWeb(tt *model.TTrigger, secret string) (rb *runtime.Build, err error) {
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
		if _, err := comm.Db.InsertOne(ttr); err != nil {
			logrus.Errorf("TriggerWeb: failed to save trigger run: %v", err)
		}
	}()
	if tt.Params == "" {
		return nil, errors.New("触发器没有配置参数")
	}
	err = TriggerPerm(tt)
	if err != nil {
		return nil, err
	}
	m := map[string]string{}
	err = json.Unmarshal([]byte(tt.Params), &m)
	if err != nil {
		err = fmt.Errorf("触发器配置参数错误: %w", err)
		return nil, err
	}
	pSecret, ok := m["secret"]
	if !ok {
		err = errors.New("触发器没有配置密钥")
		return nil, err
	}
	if secret != pSecret {
		err = errors.New("密钥不正确")
		return nil, err
	}
	branch := ""
	if s, ok := m["branch"]; ok {
		branch = s
	}
	tvp, rb, err := Run(tt.Uid, tt.PipelineId, branch, "web")
	if err != nil {
		return nil, err
	}
	tvpId = tvp.Id
	return rb, nil
}
func TriggerTimer(tt *model.TTrigger) (rb *runtime.Build, err error) {
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
		if _, err := comm.Db.InsertOne(ttr); err != nil {
			logrus.Errorf("TriggerTimer: failed to save trigger run: %v", err)
		}
	}()
	err = TriggerPerm(tt)
	if err != nil {
		return nil, err
	}
	tvp, rb, err := Run(tt.Uid, tt.PipelineId, "", "timer")
	if err != nil {
		return nil, err
	}
	ttr.PipeVersionId = tvp.Id
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
		return nil, fmt.Errorf("unknown webhook type: %q", hookType)
	}
}
