package engine

import (
	"context"
	"time"

	"github.com/gokins/core/common"
	"github.com/gokins/core/runtime"
	"github.com/gokins/gokins/comm"
	"github.com/gokins/gokins/model"
	"github.com/gokins/gokins/util"
	"github.com/sirupsen/logrus"
)

// taskCtx returns the BuildTask's own context if available (which has a timeout
// bounded to the build deadline), otherwise falls back to the global comm.Ctx.
// This ensures database operations within a build task respect the build's
// deadline rather than using an unbounded global context.
func (c *BuildTask) taskCtx() context.Context {
	if c.ctx != nil {
		return c.ctx
	}
	return comm.Ctx
}

// cascadeCancel cancels child entities when a parent build/stage/step ends.
// This helper reduces code duplication in update* methods.
func (c *BuildTask) cascadeCancel(ctx context.Context, table, foreignKey, id string, cols []string) {
	now := time.Now()
	statuses := []string{
		common.BuildStatusOk,
		common.BuildStatusError,
		common.BuildStatusCancel,
	}

	switch table {
	case "t_stage":
		entity := &model.TStage{
			Status:   common.BuildStatusCancel,
			Finished: now,
			Updated:  now,
		}
		_, err := comm.Db.Context(ctx).Cols(cols...).
			Where(foreignKey+"=? and `status` NOT IN ?", id, statuses).
			Update(entity)
		if err != nil {
			logrus.Errorf("cascadeCancel %s err:%v", table, err)
		}
	case "t_step":
		entity := &model.TStep{
			Status:   common.BuildStatusCancel,
			Finished: now,
			Updated:  now,
		}
		_, err := comm.Db.Context(ctx).Cols(cols...).
			Where(foreignKey+"=? and `status` NOT IN ?", id, statuses).
			Update(entity)
		if err != nil {
			logrus.Errorf("cascadeCancel %s err:%v", table, err)
		}
	case "t_cmd_line":
		entity := &model.TCmdLine{
			Status:   common.BuildStatusCancel,
			Finished: now,
		}
		_, err := comm.Db.Context(ctx).Cols(cols...).
			Where(foreignKey+"=? and `status` NOT IN ?", id, statuses).
			Update(entity)
		if err != nil {
			logrus.Errorf("cascadeCancel %s err:%v", table, err)
		}
	}
}

func (c *BuildTask) updateBuild(build *runtime.Build) {
	defer util.RecoverLog("BuildTask.updateBuild")

	ctx := c.taskCtx()

	e := &model.TBuild{
		Status:   build.Status,
		Error:    build.Error,
		Event:    build.Event,
		Started:  build.Started,
		Finished: build.Finished,
		Updated:  time.Now(),
	}
	_, err := comm.Db.Context(ctx).Cols("status", "event", "error", "started", "finished", "updated").
		Where("id=?", build.Id).Update(e)
	if err != nil {
		logrus.Errorf("BuildTask.updateBuild db err:%v", err)
	}

	if !common.BuildStatusEnded(e.Status) {
		return
	}

	// Cascade cancel child entities
	c.cascadeCancel(ctx, "t_stage", "build_id", build.Id, []string{"status", "finished", "updated"})
	c.cascadeCancel(ctx, "t_step", "build_id", build.Id, []string{"status", "finished", "updated"})
	c.cascadeCancel(ctx, "t_cmd_line", "build_id", build.Id, []string{"status", "finished"})
}
func (c *BuildTask) updateStage(stage *runtime.Stage) {
	defer util.RecoverLog("BuildTask.updateStage")

	ctx := c.taskCtx()

	e := &model.TStage{
		Status:   stage.Status,
		Error:    stage.Error,
		Started:  stage.Started,
		Finished: stage.Finished,
		Updated:  time.Now(),
	}
	_, err := comm.Db.Context(ctx).Cols("status", "error", "started", "finished", "updated").
		Where("id=?", stage.Id).Update(e)
	if err != nil {
		logrus.Errorf("BuildTask.updateStage db err:%v", err)
	}

	if !common.BuildStatusEnded(e.Status) {
		return
	}

	// Cascade cancel child steps
	c.cascadeCancel(ctx, "t_step", "stage_id", stage.Id, []string{"status", "finished", "updated"})
}
func (c *BuildTask) updateStep(job *jobSync) {
	defer util.RecoverLog("BuildTask.updateStep")

	ctx := c.taskCtx()

	job.RLock()
	defer job.RUnlock()
	e := &model.TStep{
		Status:   job.step.Status,
		Event:    job.step.Event,
		Error:    job.step.Error,
		ExitCode: job.step.ExitCode,
		Started:  job.step.Started,
		Finished: job.step.Finished,
		Updated:  time.Now(),
	}
	_, err := comm.Db.Context(ctx).Cols("status", "event", "error", "exit_code", "started", "finished", "updated").
		Where("id=?", job.step.Id).Update(e)
	if err != nil {
		logrus.Errorf("BuildTask.updateStep db err:%v", err)
	}

	if !common.BuildStatusEnded(e.Status) {
		return
	}

	// Cascade cancel child command lines
	c.cascadeCancel(ctx, "t_cmd_line", "step_id", job.step.Id, []string{"status", "finished"})
}
func (c *BuildTask) updateStepCmd(cmd *cmdSync) {
	defer util.RecoverLog("BuildTask.updateStepCmd")

	ctx := c.taskCtx()

	cmd.RLock()
	defer cmd.RUnlock()
	cmde := &model.TCmdLine{
		Status: cmd.status,
		Code:   cmd.code,
	}
	cols := []string{"status"}
	switch cmd.status {
	case common.BuildStatusRunning:
		cmde.Started = cmd.started
		cols = append(cols, "started")
	default:
		cmde.Finished = cmd.finished
		cols = append(cols, "finished")
	}
	_, err := comm.Db.Context(ctx).Cols(cols...).Where("id=?", cmd.cmd.Id).Update(cmde)
	if err != nil {
		logrus.Errorf("BuildTask.updateStepCmd db err:%v", err)
	}
}
