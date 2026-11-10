package service

import (
	"github.com/gin-gonic/gin"
	"github.com/gokins/gokins/model"
	"github.com/gokins/gokins/pkg/middleware"
)

// LgUserKey is the gin.Context key used to store the authenticated user.
// Deprecated: Use middleware.LgUserKey instead.
const LgUserKey = middleware.LgUserKey

// MidUserCheck validates the current user and sets it in the gin.Context.
// It checks if the user is authenticated and either active or an admin.
// This is a thin wrapper around the generic middleware.MidAuth for backward compatibility.
func MidUserCheck(c *gin.Context) {
	middleware.MidAuth(CurrUserCache, IsAdmin)(c)
}

// GetMidLgUser extracts the authenticated user from the gin.Context.
// Deprecated: Use middleware.GetAuthUser instead.
func GetMidLgUser(c *gin.Context) *model.TUser {
	return middleware.GetAuthUser(c)
}
