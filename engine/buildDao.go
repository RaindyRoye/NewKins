package engine

import (
	"context"
	"errors"
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

// errDbNotAvailable is returned by database update helpers when the global
// database engine (comm.Db) has not been initialized. Previously such calls
// would panic with a nil-pointer dereference and rely on recover() to swallow
// the crash; returning a sentinel error lets callers log or react explicitly.
var errDbNotAvailable = errors.New("database engine is not available")

// safeUpdateBuild persists build and related entity state to the database,
// returning any database error to the caller instead of swallowing it via a
// panic-recovered deferred log. The function still logs errors internally for
// backward compatibility with existing callers that ignore the return value.
func (c *BuildTask) safeUpdateBuild(build *runtime.Build) error {
	ctx := c.taskCtx()

	if comm.Db == nil {
		logrus.Errorf("BuildTask.updateBuild: %v", errDbNotAvailable)
		return errDbNotAvailable
	}

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
		return err
	}

	if !common.BuildStatusEnded(e.Status) {
		return nil
	}

	if err := c.cancelRelatedEntities(ctx, build.Id, "build_id"); err != nil {
		return err
	}
	return nil
}

// cancelRelatedEntities marks child rows (stages, steps, cmd lines) of a
// finished build as canceled when they did not reach a terminal state. Any
// database error is logged and returned so callers can decide how to react
// instead of silently swallowing the failure via recover().
func (c *BuildTask) cancelRelatedEntities(ctx context.Context, id, fk string) error {
	now := time.Now()
	statusFilter := "build_id=? and `status`!=? and `status`!=? and `status`!=?"
	switch fk {
	case "stage_id":
		statusFilter = "stage_id=? and `status`!=? and `status`!=? and `status`!=?"
	case "step_id":
		statusFilter = "step_id=? and `status`!=? and `status`!=? and `status`!=?"
	}

	stge := &model.TStage{
		Status:   common.BuildStatusCancel,
		Finished: now,
		Updated:  now,
	}
	if fk == "build_id" {
		if _, err := comm.Db.Context(ctx).Cols("status", "finished", "updated").
			Where(statusFilter, id, common.BuildStatusOk, common.BuildStatusError, common.BuildStatusCancel).
			Update(stge); err != nil {
			logrus.Errorf("BuildTask.cancelRelatedEntities stage err:%v", err)
			return err
		}
	}

	stpe := &model.TStep{
		Status:   common.BuildStatusCancel,
		Finished: now,
		Updated:  now,
	}
	if fk == "build_id" || fk == "stage_id" {
		if _, err := comm.Db.Context(ctx).Cols("status", "finished", "updated").
			Where(statusFilter, id, common.BuildStatusOk, common.BuildStatusError, common.BuildStatusCancel).
			Update(stpe); err != nil {
			logrus.Errorf("BuildTask.cancelRelatedEntities step err:%v", err)
			return err
		}
	}

	cmde := &model.TCmdLine{
		Status:   common.BuildStatusCancel,
		Finished: now,
	}
	if _, err := comm.Db.Context(ctx).Cols("status", "finished").
		Where(statusFilter, id, common.BuildStatusOk, common.BuildStatusError, common.BuildStatusCancel).
		Update(cmde); err != nil {
		logrus.Errorf("BuildTask.cancelRelatedEntities cmd err:%v", err)
		return err
	}
	return nil
}

func (c *BuildTask) updateBuild(build *runtime.Build) {
	_ = c.safeUpdateBuild(build)
}
func (c *BuildTask) updateStage(stage *runtime.Stage) {
	defer util.RecoverLog("BuildTask.updateStage")

	ctx := c.taskCtx()

	if comm.Db == nil {
		logrus.Errorf("BuildTask.updateStage: %v", errDbNotAvailable)
		return
	}

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
		return
	}

	if !common.BuildStatusEnded(e.Status) {
		return
	}
	stpe := &model.TStep{
		Status:   common.BuildStatusCancel,
		Finished: time.Now(),
		Updated:  time.Now(),
	}
	_, err = comm.Db.Context(ctx).Cols("status", "finished", "updated").
		Where("stage_id=? and `status`!=? and `status`!=? and `status`!=?",
			stage.Id, common.BuildStatusOk, common.BuildStatusError, common.BuildStatusCancel).Update(stpe)
	if err != nil {
		logrus.Errorf("BuildTask.updateStage step err:%v", err)
	}
}
func (c *BuildTask) updateStep(job *jobSync) {
	defer util.RecoverLog("BuildTask.updateStep")

	ctx := c.taskCtx()

	if comm.Db == nil {
		logrus.Errorf("BuildTask.updateStep: %v", errDbNotAvailable)
		return
	}

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
		return
	}

	if !common.BuildStatusEnded(e.Status) {
		return
	}
	cmde := &model.TCmdLine{
		Status:   common.BuildStatusCancel,
		Finished: time.Now(),
	}
	_, err = comm.Db.Context(ctx).Cols("status", "finished").
		Where("step_id=? and `status`!=? and `status`!=? and `status`!=?",
			job.step.Id, common.BuildStatusOk, common.BuildStatusError, common.BuildStatusCancel).Update(cmde)
	if err != nil {
		logrus.Errorf("BuildTask.updateStep cmd err:%v", err)
	}
}
func (c *BuildTask) updateStepCmd(cmd *cmdSync) {
	defer util.RecoverLog("BuildTask.updateStepCmd")

	ctx := c.taskCtx()

	if comm.Db == nil {
		logrus.Errorf("BuildTask.updateStepCmd: %v", errDbNotAvailable)
		return
	}

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
