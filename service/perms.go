package service

import (
	"context"

	"github.com/gin-gonic/gin"
	"github.com/gokins/gokins/comm"
	"github.com/gokins/gokins/model"
)

// Permission level constants used across the application for access control checks.
const (
	PermCommon = "common"
	PermAdmin  = "admin"
)

// AdminUserName is the reserved name for the super-admin account.
const AdminUserName = "admin"

// CheckPermission checks whether a user identified by uid has the given
// permission level, using the global comm.Ctx.
// Prefer CheckPermissionCtx when a request context is available.
func CheckPermission(uid string, perms string) bool {
	return CheckPermissionCtx(comm.Ctx, uid, perms)
}

// CheckPermissionCtx is the context-aware version of CheckPermission.
// It fetches the user with the provided context (enabling cancellation/timeout)
// and then delegates to CheckUPermission.
func CheckPermissionCtx(ctx context.Context, uid string, perms string) bool {
	usr, ok := GetUserCtx(ctx, uid)
	if !ok {
		return false
	}
	return CheckUPermission(usr, perms)
}
func CheckUPermission(usr *model.TUser, perms string) bool {
	if usr == nil {
		return false
	}
	if perms == PermCommon {
		return true
	} else if perms == PermAdmin && usr.Name == AdminUserName {
		return true
	}
	return false
}

// CheckCurrPermission checks the current user's permission from a Gin context,
// using the request's context for database operations.
func CheckCurrPermission(c *gin.Context, perms string) bool {
	if c == nil || c.Request == nil {
		return false
	}
	usr, ok := CurrUserCacheCtx(c.Request.Context(), c)
	if !ok {
		return false
	}
	return CheckUPermission(usr, perms)
}
