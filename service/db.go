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
func GetIdOrAid(id any, e any) bool {
	return GetIdOrAidCtx(comm.Ctx, id, e)
}

// GetIdOrAidCtx is the context-aware version of GetIdOrAid.
func GetIdOrAidCtx(ctx context.Context, id any, e any) bool {
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
func GetIdOrAidE(id any, e any) (bool, error) {
	return GetIdOrAidECtx(comm.Ctx, id, e)
}

// GetIdOrAidECtx is the context-aware version of GetIdOrAidE.
func GetIdOrAidECtx(ctx context.Context, id any, e any) (bool, error) {
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

// BatchOrgPipeCounts returns the number of pipelines for each org ID in a single query,
// eliminating N+1 queries when listing organizations.
func BatchOrgPipeCounts(ctx context.Context, orgIds []string) (map[string]int64, error) {
	if len(orgIds) == 0 {
		return map[string]int64{}, nil
	}
	type orgCount struct {
		OrgId string `xorm:"org_id"`
		Cnt   int64  `xorm:"cnt"`
	}
	var counts []orgCount
	err := comm.Db.Context(ctx).Table("t_org_pipe").
		Select("org_id, COUNT(*) as cnt").
		In("org_id", orgIds).
		GroupBy("org_id").
		Find(&counts)
	if err != nil {
		return nil, fmt.Errorf("batch org pipe counts: %w", err)
	}
	result := make(map[string]int64, len(counts))
	for _, c := range counts {
		result[c.OrgId] = c.Cnt
	}
	return result, nil
}

// BatchOrgUserCounts returns the number of users for each org ID in a single query,
// eliminating N+1 queries when listing organizations.
func BatchOrgUserCounts(ctx context.Context, orgIds []string) (map[string]int64, error) {
	if len(orgIds) == 0 {
		return map[string]int64{}, nil
	}
	type orgCount struct {
		OrgId string `xorm:"org_id"`
		Cnt   int64  `xorm:"cnt"`
	}
	var counts []orgCount
	err := comm.Db.Context(ctx).Table("t_user_org").
		Select("org_id, COUNT(*) as cnt").
		In("org_id", orgIds).
		GroupBy("org_id").
		Find(&counts)
	if err != nil {
		return nil, fmt.Errorf("batch org user counts: %w", err)
	}
	result := make(map[string]int64, len(counts))
	for _, c := range counts {
		result[c.OrgId] = c.Cnt
	}
	return result, nil
}
