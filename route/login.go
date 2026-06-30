package route

import (
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gokins/core/common"
	"github.com/gokins/core/utils"
	"github.com/gokins/gokins/bean"
	"github.com/gokins/gokins/comm"
	"github.com/gokins/gokins/model"
	"github.com/gokins/gokins/service"
	"github.com/gokins/gokins/util"
	"github.com/golang-jwt/jwt/v5"
	hbtp "github.com/mgr9525/HyperByte-Transfer-Protocol"
	"github.com/sirupsen/logrus"
)

type LoginController struct{}

// loginRateLimiter limits login attempts to 10 per minute per IP.
var loginRateLimiter = util.NewRateLimiter(10, time.Minute)

func (LoginController) GetPath() string {
	return "/api/lg"
}
func (c *LoginController) Routes(g gin.IRoutes) {
	g.POST("/info", c.info)
	g.POST("/login", util.MidRateLimit(loginRateLimiter), util.GinReqParseJson(c.login))
}
func (LoginController) info(c *gin.Context) {
	rt := hbtp.Map{}
	usr, ok := service.CurrUserCache(c)
	if ok {
		usrs := &model.TUser{}
		if err := utils.Struct2Struct(usrs, usr); err != nil {
			logrus.Warnf("login info: struct2struct user: %v", err)
		}
		rt["user"] = usrs
		info, _ := service.GetUserInfoCtx(c.Request.Context(), usrs.Id)
		rt["info"] = info
		if service.IsAdmin(usr) {
			info.PermUser = 1
			info.PermOrg = 1
			info.PermPipe = 1
		}
	}
	rt["login"] = ok
	c.JSON(200, rt)
}
func (LoginController) login(c *gin.Context, m *bean.LoginReq) {
	m.Name = strings.TrimSpace(m.Name)
	if m.Name == "" || m.Pass == "" {
		c.String(400, "param err")
		return
	}
	usr, ok := service.FindUserNameCtx(c.Request.Context(), m.Name)
	if !ok {
		c.String(404, "not found user")
		return
	}
	if !service.IsAdmin(usr) && usr.Active != 1 {
		c.String(http.StatusForbidden, "user account is not active")
		return
	}
	if usr.Pass != utils.Md5String(m.Pass) {
		c.String(http.StatusUnauthorized, "invalid username or password")
		return
	}
	key := comm.Cfg.Server.LoginKey
	if key == "" {
		c.String(http.StatusInternalServerError, "server configuration error: login key not set")
		return
	}
	token, err := util.CreateToken(jwt.MapClaims{
		"uid": usr.Id,
	}, key, time.Hour*24*5)
	if err != nil {
		util.RespInternalErr(c, "create auth token", err)
		return
	}
	rt := &bean.LoginRes{
		Token:         token,
		Id:            usr.Id,
		Name:          usr.Name,
		Nick:          usr.Nick,
		Avatar:        usr.Avatar,
		LastLoginTime: usr.LoginTime.Format(common.TimeFmt),
	}
	c.JSON(200, rt)

	usr.LoginTime = time.Now()
	if _, err := comm.Db.Context(c.Request.Context()).Cols("login_time").Where("id=?", usr.Id).Update(usr); err != nil {
		logrus.Errorf("login: failed to update login time for user %s: %v", usr.Id, err)
	}
}
