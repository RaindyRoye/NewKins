package service

import (
	"github.com/gin-gonic/gin"
	"github.com/gokins/gokins/model"
)

// Permission level constants used across the application for access control checks.
const (
	PermCommon = "common"
	PermAdmin  = "admin"
)

// AdminUserName is the reserved name for the super-admin account.
const AdminUserName = "admin"

func CheckPermission(uid string, perms string) bool {
	usr, ok := GetUser(uid)
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
func CheckCurrPermission(c *gin.Context, perms string) bool {
	usr, ok := CurrUserCache(c)
	if !ok {
		return false
	}
	return CheckUPermission(usr, perms)
}
