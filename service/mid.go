package service

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/gokins/gokins/model"
)

const LgUserKey = "lguser"

// MidUserCheck validates that the current user is authenticated and active.
// Uses the request's context for database/cache lookups, enabling cancellation.
func MidUserCheck(c *gin.Context) {
	if c == nil || c.Request == nil {
		c.String(http.StatusForbidden, "Not Auth")
		c.Abort()
		return
	}
	usr, ok := CurrUserCacheCtx(c.Request.Context(), c)
	if !ok || (!IsAdmin(usr) && usr.Active != 1) {
		c.String(http.StatusForbidden, "Not Auth")
		c.Abort()
		return
	}
	c.Set(LgUserKey, usr)
	c.Next()
}
func GetMidLgUser(c *gin.Context) *model.TUser {
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
