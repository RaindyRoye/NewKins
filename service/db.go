package service

import (
	"context"
	"fmt"

	"github.com/gokins/gokins/comm"
	"github.com/sirupsen/logrus"
)

// GetIdOrAid looks up a record by id first, then falls back to aid.
// Returns true if the record was found.
// DB errors are logged but do not change the return value (false = not found or error).
// Uses the global comm.Ctx; prefer GetIdOrAidCtx when a request context is available.
func GetIdOrAid(id interface{}, e interface{}) bool {
	return GetIdOrAidCtx(comm.Ctx, id, e)
}

// GetIdOrAidCtx is the context-aware version of GetIdOrAid.
func GetIdOrAidCtx(ctx context.Context, id interface{}, e interface{}) bool {
	if id == nil || e == nil {
		return false
	}
	if v, ok := id.(string); ok && v == "" {
		return false
	}
	ok, err := comm.Db.Context(ctx).Where("id=?", id).Get(e)
	if err != nil {
		logrus.Errorf("GetIdOrAid(id=%v) query error: %v", id, err)
		return false
	}
	if !ok {
		ok, err = comm.Db.Context(ctx).Where("aid=?", id).Get(e)
		if err != nil {
			logrus.Errorf("GetIdOrAid(aid=%v) fallback query error: %v", id, err)
			return false
		}
	}
	return ok
}

// GetIdOrAidE is like GetIdOrAid but returns errors instead of logging them.
// Use this in contexts where callers need to distinguish "not found" from "DB error".
// Uses the global comm.Ctx; prefer GetIdOrAidECtx when a request context is available.
func GetIdOrAidE(id interface{}, e interface{}) (bool, error) {
	return GetIdOrAidECtx(comm.Ctx, id, e)
}

// GetIdOrAidECtx is the context-aware version of GetIdOrAidE.
func GetIdOrAidECtx(ctx context.Context, id interface{}, e interface{}) (bool, error) {
	if id == nil || e == nil {
		return false, nil
	}
	if v, ok := id.(string); ok && v == "" {
		return false, nil
	}
	ok, err := comm.Db.Context(ctx).Where("id=?", id).Get(e)
	if err != nil {
		return false, fmt.Errorf("GetIdOrAidE(id=%v): %w", id, err)
	}
	if !ok {
		ok, err = comm.Db.Context(ctx).Where("aid=?", id).Get(e)
		if err != nil {
			return false, fmt.Errorf("GetIdOrAidE(aid=%v): %w", id, err)
		}
	}
	return ok, nil
}
