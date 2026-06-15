package service

import (
	"context"

	"github.com/gokins/gokins/comm"
	"github.com/gokins/gokins/model"
)

// TriggerPerm checks whether the trigger creator has permission to run the pipeline.
// Uses the global comm.Ctx; prefer TriggerPermCtx when a request context is available.
func TriggerPerm(tt *model.TTrigger) error {
	return TriggerPermCtx(comm.Ctx, tt)
}

// TriggerPermCtx checks whether the trigger creator has permission to run the pipeline,
// using the provided context for database queries.
func TriggerPermCtx(ctx context.Context, tt *model.TTrigger) error {
	if tt == nil {
		return ErrPipelineNotFound
	}
	lgus := &model.TUser{
		Id: tt.Uid,
	}
	perm := NewPipePermCtx(ctx, lgus, tt.PipelineId)
	if perm.Pipeline() == nil {
		return ErrPipelineNotFound
	}
	if !IsAdmin(lgus) && !perm.CanWrite() {
		return ErrPermissionDenied
	}
	return nil
}
