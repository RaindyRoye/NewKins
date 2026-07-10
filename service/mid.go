package service

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/gokins/gokins/model"
)

const LgUserKey = "lguser"

// MidUserCheck is a Gin middleware that verifies the request is authenticated
// and the user account is active. It returns 403 Forbidden when the token is
// missing, invalid, or the account is deactivated. On success the user struct
// is stored in the Gin context under LgUserKey so downstream handlers can
// retrieve it with GetMidLgUser.
func MidUserCheck(c *gin.Context) {
	usr, ok := CurrUserCache(c)
	if !ok || (!IsAdmin(usr) && usr.Active != 1) {
		c.String(http.StatusForbidden, "forbidden")
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
