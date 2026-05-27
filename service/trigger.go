package service

import (
	"github.com/gokins/gokins/model"
)

func TriggerPerm(tt *model.TTrigger) error {
	lgus := &model.TUser{
		Id: tt.Uid,
	}
	perm := NewPipePerm(lgus, tt.PipelineId)
	if perm.Pipeline() == nil {
		return ErrPipelineNotFound
	}
	if !IsAdmin(lgus) && !perm.CanWrite() {
		return ErrPermissionDenied
	}
	return nil
}
