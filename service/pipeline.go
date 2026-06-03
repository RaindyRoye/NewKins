package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/gokins/core/common"
	"github.com/gokins/core/runtime"
	"github.com/gokins/core/utils"
	"github.com/gokins/gokins/bean"
	"github.com/gokins/gokins/comm"
	"github.com/gokins/gokins/model"
	"gopkg.in/yaml.v3"
)

func Run(ctx context.Context, uid, pipeId, sha, event string) (*model.TPipelineVersion, *runtime.Build, error) {
	tpipe := &model.TPipelineConf{}
	ok, _ := comm.Db.Context(ctx).Where("pipeline_id=?", pipeId).Get(tpipe)
	if !ok {
		return nil, nil, ErrPipelineNotFound
	}
	if tpipe.YmlContent == "" {
		return nil, nil, ErrPipelineYmlEmpty
	}
	pipe := &bean.Pipeline{}
	err := yaml.Unmarshal([]byte(tpipe.YmlContent), pipe)
	if err != nil {
		return nil, nil, fmt.Errorf("parse pipeline yaml: %w", err)
	}
	return preBuild(ctx, uid, pipe, tpipe, sha, event)
}

func ReBuild(ctx context.Context, uid string, tvp *model.TPipelineVersion) (*model.TPipelineVersion, *runtime.Build, error) {
	tpipe := &model.TPipelineConf{}
	ok, _ := comm.Db.Context(ctx).Where("pipeline_id=?", tvp.PipelineId).Get(tpipe)
	if !ok {
		return nil, nil, ErrPipelineNotFound
	}
	if tvp.Content == "" {
		return nil, nil, ErrPipelineYmlEmpty
	}
	pipe := &bean.Pipeline{}
	err := yaml.Unmarshal([]byte(tvp.Content), pipe)
	if err != nil {
		return nil, nil, fmt.Errorf("parse pipeline version yaml: %w", err)
	}
	return preBuild(ctx, uid, pipe, tpipe, tvp.Sha, "rebuild", tvp)
}

func preBuild(ctx context.Context, uid string, pipe *bean.Pipeline, tpipe *model.TPipelineConf, sha, event string,
	tvp ...*model.TPipelineVersion) (*model.TPipelineVersion, *runtime.Build, error) {
	db := comm.Db.Context(ctx)
	tp := &model.TPipeline{}
	ok, _ := db.Where("id=? and deleted != 1", tpipe.PipelineId).Get(tp)
	if !ok {
		return nil, nil, ErrPipelineNotFound
	}
	err := pipe.Check()
	if err != nil {
		return nil, nil, fmt.Errorf("pipeline config check: %w", err)
	}
	pipe.ConvertCmd()

	m, err := convertVar(ctx, tpipe.PipelineId, pipe.Vars)
	if err != nil {
		return nil, nil, fmt.Errorf("convertVar: %w", err)
	}
	m["GOKINSV_GIT_SHA"] = &runtime.Variables{
		Name:   "GOKINSV_GIT_SHA",
		Value:  sha,
		Secret: true,
	}
	m["GOKINSV_RUNNER_REPOPATH"] = &runtime.Variables{
		Name:   "GOKINSV_RUNNER_REPOPATH",
		Value:  "{{RUNNER_REPOPATH}}",
		Secret: false,
	}
	replaceStages(pipe.Stages, m)

	number := int64(0)
	_, err = db.
		SQL("SELECT max(number) FROM t_pipeline_version WHERE pipeline_id = ?", tpipe.PipelineId).
		Get(&number)
	if err != nil {
		return nil, nil, fmt.Errorf("query max pipeline version number: %w", err)
	}
	tpv := &model.TPipelineVersion{
		Id:                  utils.NewXid(),
		Uid:                 uid,
		Number:              number + 1,
		Events:              event,
		Sha:                 sha,
		PipelineName:        tp.Name,
		PipelineDisplayName: tp.DisplayName,
		PipelineId:          tpipe.PipelineId,
		Version:             "",
		Content:             tpipe.YmlContent,
		Created:             time.Now(),
		Deleted:             0,
		RepoCloneUrl:        tpipe.Url,
	}
	if len(tvp) > 0 && tvp[0] != nil {
		tpv.Sha = tvp[0].Sha
		tpv.Content = tvp[0].Content
	}
	_, err = db.InsertOne(tpv)
	if err != nil {
		return nil, nil, fmt.Errorf("insert pipeline version: %w", err)
	}

	tb := &model.TBuild{
		Id:                utils.NewXid(),
		PipelineId:        tpipe.PipelineId,
		PipelineVersionId: tpv.Id,
		Status:            common.BuildStatusPending,
		Created:           time.Now(),
		Version:           "",
	}
	_, err = db.InsertOne(tb)
	if err != nil {
		return nil, nil, fmt.Errorf("insert build: %w", err)
	}

	rb := &runtime.Build{
		Id:                tb.Id,
		PipelineId:        tb.PipelineId,
		PipelineVersionId: tb.PipelineVersionId,
		Status:            common.BuildStatusPending,
		Created:           time.Now(),
		Repo: &runtime.Repository{
			Name:     tpipe.Username,
			Token:    tpipe.AccessToken,
			Sha:      tpv.Sha,
			CloneURL: tpipe.Url,
		},
		Vars: m,
	}

	for i, stage := range pipe.Stages {
		ts := &model.TStage{
			Id:                utils.NewXid(),
			PipelineVersionId: tpv.Id,
			BuildId:           tb.Id,
			Status:            common.BuildStatusPending,
			Name:              stage.Name,
			DisplayName:       stage.DisplayName,
			Created:           time.Now(),
			Stage:             stage.Stage,
			Sort:              i,
		}
		rt := &runtime.Stage{
			Id:          ts.Id,
			BuildId:     tb.Id,
			Status:      common.BuildStatusPending,
			Name:        stage.Name,
			DisplayName: stage.DisplayName,
			Created:     time.Now(),
			Stage:       stage.Stage,
		}
		_, err = db.InsertOne(ts)
		if err != nil {
			return nil, nil, fmt.Errorf("insert stage %q: %w", stage.Name, err)
		}
		for j, step := range stage.Steps {
			if step.Disable {
				continue
			}
			cmds, err := json.Marshal(step.Commands)
			if err != nil {
				return nil, nil, fmt.Errorf("marshal step %q commands: %w", step.Name, err)
			}
			djs, err := json.Marshal(step.Waits)
			if err != nil {
				return nil, nil, fmt.Errorf("marshal step %q waits: %w", step.Name, err)
			}
			tsp := &model.TStep{
				Id:                utils.NewXid(),
				BuildId:           tb.Id,
				StageId:           ts.Id,
				DisplayName:       step.DisplayName,
				PipelineVersionId: tpv.Id,
				Step:              step.Step,
				Status:            common.BuildStatusPending,
				Name:              step.Name,
				Created:           time.Now(),
				Commands:          string(cmds),
				Waits:             string(djs),
				Sort:              j,
			}
			rtp := &runtime.Step{
				Id:          tsp.Id,
				BuildId:     tb.Id,
				StageId:     ts.Id,
				DisplayName: step.DisplayName,
				Step:        step.Step,
				Status:      common.BuildStatusPending,
				Name:        step.Name,
				Commands:    step.Commands,
				Waits:       step.Waits,
				Env:         step.Env,
				Input:       step.Input,
				MustCopy:    step.MustCopy,
				RepoPath:    step.Repo,
			}
			if rtp.RepoPath == "" && stage.Repo != "" {
				rtp.RepoPath = stage.Repo
			}
			for _, v := range step.Artifacts {
				rtp.Artifacts = append(rtp.Artifacts, &runtime.Artifact{
					Scope:      v.Scope,
					Repository: v.Repository,
					Name:       v.Name,
					Path:       v.Path,
				})
			}
			for _, v := range step.UseArtifacts {
				rtp.UseArtifacts = append(rtp.UseArtifacts, &runtime.UseArtifact{
					Scope:      v.Scope,
					Repository: v.Repository,
					Name:       v.Name,
					Path:       v.Path,
					// IsForce:     v.IsForce,
					IsUrl:       v.IsUrl,
					Alias:       v.Alias,
					SourceStage: v.FromStage,
					SourceStep:  v.FromStep,
				})
			}
			_, err = db.InsertOne(tsp)
			if err != nil {
				return nil, nil, fmt.Errorf("insert step %q: %w", step.Name, err)
			}
			rt.Steps = append(rt.Steps, rtp)
		}
		rb.Stages = append(rb.Stages, rt)
	}
	return tpv, rb, nil
}

func getOrgVars(ctx context.Context, pipelineId string) ([]*model.TOrgVar, error) {
	var rts []*model.TOrgVar
	// Use a single query with a subquery to avoid N+1 problem
	err := comm.Db.Context(ctx).SQL(
		"SELECT * FROM t_org_var WHERE org_id IN (SELECT org_id FROM t_org_pipe WHERE pipe_id = ?)",
		pipelineId,
	).Find(&rts)
	if err != nil {
		return nil, fmt.Errorf("query org vars for pipeline %q: %w", pipelineId, err)
	}
	return rts, nil
}

func convertVar(ctx context.Context, pipelineId string, vm map[string]string) (map[string]*runtime.Variables, error) {
	vms := make(map[string]*runtime.Variables, 0)

	oVars, err := getOrgVars(ctx, pipelineId)
	if err != nil {
		return nil, fmt.Errorf("getOrgVars: %w", err)
	}
	for _, v := range oVars {
		vms[v.Name] = &runtime.Variables{
			Name:   v.Name,
			Value:  v.Value,
			Secret: v.Public == 1,
		}
	}

	var tVars []*model.TPipelineVar
	if err := comm.Db.Context(ctx).Where("pipeline_id = ? ", pipelineId).Find(&tVars); err != nil {
		return nil, fmt.Errorf("query pipeline vars: %w", err)
	}
	for _, v := range tVars {
		vms[v.Name] = &runtime.Variables{
			Name:   v.Name,
			Value:  v.Value,
			Secret: v.Public == 1,
		}
	}

	for k, v := range vm {
		s, b := replace(v, vms, true)
		vms[k] = &runtime.Variables{
			Name:   k,
			Value:  s,
			Secret: b,
		}
	}

	for k, v := range vms {
		s, _ := replace(v.Value, vms, true)
		vms[k].Value = s
	}
	return vms, nil
}

func replaceStages(stages []*bean.Stage, mVars map[string]*runtime.Variables) {
	for _, stage := range stages {
		replaceStage(stage, mVars)
	}
}
func replaceStage(stage *bean.Stage, mVars map[string]*runtime.Variables) {
	s, _ := replace(stage.Stage, mVars)
	stage.Stage = s
	s, _ = replace(stage.Name, mVars)
	stage.Name = s
	s, _ = replace(stage.DisplayName, mVars)
	stage.DisplayName = s
	s, _ = replace(stage.Repo, mVars)
	stage.Repo = s
	if len(stage.Steps) > 0 {
		replaceSteps(stage.Steps, mVars)
	}
}
func replaceSteps(steps []*bean.Step, mVars map[string]*runtime.Variables) {
	for _, step := range steps {
		replaceStep(step, mVars)
	}
}
func replaceStep(step *bean.Step, mVars map[string]*runtime.Variables) {
	s, _ := replace(step.Step, mVars)
	step.Step = s
	s, _ = replace(step.Name, mVars)
	step.Name = s
	s, _ = replace(step.DisplayName, mVars)
	step.DisplayName = s
	s, _ = replace(step.Image, mVars)
	step.Image = s
	s, _ = replace(step.Repo, mVars)
	step.Repo = s
	if len(step.Env) > 0 {
		step.Env = replaceMaps(step.Env, mVars)
	}
	if len(step.Input) > 0 {
		step.Input = replaceMaps(step.Input, mVars)
	}
}

func replaceMaps(envs map[string]string, mVars map[string]*runtime.Variables) map[string]string {
	m := map[string]string{}
	for k, v := range envs {
		s, _ := replace(v, mVars, true)
		m[k] = s
	}
	return m
}

func replace(s string, mVars map[string]*runtime.Variables, mustShow ...bool) (string, bool) {
	if s == "" {
		return "", false
	}
	conts := s
	ms := len(mustShow) == 1 && mustShow[0]
	secret := false
	if common.RegVar.MatchString(s) {
		all := common.RegVar.FindAllStringSubmatch(s, -1)
		for _, v2 := range all {
			rVar, ok := mVars[v2[1]]
			va := ""
			st := false
			if ok {
				st = rVar.Secret
				va = rVar.Value
				if !secret && rVar.Secret {
					secret = true
				}
			}
			if !ms && st {
				conts = strings.ReplaceAll(conts, v2[0], comm.MaskedValue)
			} else {
				conts = strings.ReplaceAll(conts, v2[0], va)
			}
		}
	}
	return conts, secret
}
