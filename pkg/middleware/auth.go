package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/gokins/gokins/model"
)

// LgUserKey is the gin.Context key used to store the authenticated user.
const LgUserKey = "lguser"

// UserResolver is a function that retrieves the current user from the request context.
// It returns the user and true if found, or nil and false if not authenticated.
type UserResolver func(c *gin.Context) (*model.TUser, bool)

// AdminChecker is a function that returns true if the user has admin privileges.
type AdminChecker func(usr *model.TUser) bool

// MidAuth creates an authentication middleware that uses the provided resolver
// and admin checker to validate and authorize users.
//
// This middleware:
//  1. Resolves the current user using the resolver function
//  2. Checks if the user is active or is an admin
//  3. Stores the user in gin.Context under LgUserKey
//  4. Calls c.Abort() with 403 Forbidden if authentication fails
func MidAuth(resolver UserResolver, isAdmin AdminChecker) gin.HandlerFunc {
	return func(c *gin.Context) {
		usr, ok := resolver(c)
		if !ok || (!isAdmin(usr) && usr.Active != 1) {
			c.String(http.StatusForbidden, "Not Auth")
			c.Abort()
			return
		}
		c.Set(LgUserKey, usr)
		c.Next()
	}
}

// GetAuthUser extracts the authenticated user from the gin.Context.
// Returns nil if no user is set or if the stored value is not a *model.TUser.
func GetAuthUser(c *gin.Context) *model.TUser {
	usr, ok := c.Get(LgUserKey)
	if !ok {
		return nil
	}
	lguser, ok := usr.(*model.TUser)
	if !ok {
		return nil
	}
	return lguser
}
