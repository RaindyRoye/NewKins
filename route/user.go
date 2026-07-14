package route

import (
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gokins/core/utils"
	"github.com/gokins/gokins/comm"
	"github.com/gokins/gokins/model"
	"github.com/gokins/gokins/pkg/middleware"
	"github.com/gokins/gokins/service"
	"github.com/gokins/gokins/util"
	hbtp "github.com/mgr9525/HyperByte-Transfer-Protocol"
)

type UserController struct{}

// passRateLimiter limits password change attempts to 5 per minute per IP
// to mitigate brute-force attacks on the password change endpoint.
var passRateLimiter = middleware.NewRateLimiter(5, time.Minute)

func (UserController) GetPath() string {
	return "/api/user"
}
func (c *UserController) Routes(g gin.IRoutes) {
	g.Use(service.MidUserCheck)
	g.POST("/page", util.GinReqParseJson(c.page))
	g.POST("/new", util.GinReqParseJson(c.new))
	g.POST("/info", util.GinReqParseJson(c.info))
	g.POST("/upinfo", util.GinReqParseJson(c.upinfo))
	g.POST("/upass", middleware.MidRateLimit(passRateLimiter), util.GinReqParseJson(c.upass))
	g.POST("/active", util.GinReqParseJson(c.active))
	g.POST("/perm", util.GinReqParseJson(c.perm))
}
func (UserController) page(c *gin.Context, m *hbtp.Map) {
	var ls []*model.TUser
	q := m.GetString("q")
	pg, _ := m.GetInt("page")

	ctx := c.Request.Context()
	ses := comm.Db.Context(ctx).OrderBy("aid ASC")
	if q != "" {
		ses.And("name like ? or nick like ?", "%"+q+"%", "%"+q+"%")
	}

	page, err := comm.FindPage(ses, &ls, pg, 20)
	if err != nil {
		util.RespInternalErr(c, "user page query", err)
		return
	}
	c.JSON(http.StatusOK, page)
}
func (UserController) new(c *gin.Context, m *hbtp.Map) {
	name := strings.TrimSpace(m.GetString("name"))
	nick := strings.TrimSpace(m.GetString("nick"))
	pass := m.GetString("pass")
	// pmUser:=m.GetBool("pmUser")
	// pmOrg:=m.GetBool("pmOrg")
	// pmPipe:=m.GetBool("pmPipe")
	if name == "" || nick == "" || pass == "" {
		c.String(http.StatusBadRequest, "param err")
		return
	}
	ctx := c.Request.Context()
	lgusr := service.GetMidLgUser(c)
	if !service.IsAdmin(lgusr) {
		uf, ok := service.GetUserInfoCtx(ctx, lgusr.Id)
		if !ok || uf.PermUser != 1 {
			c.String(http.StatusMethodNotAllowed, "no permission")
			return
		}
	}
	_, ok := service.FindUserNameCtx(ctx, name)
	if ok {
		c.String(http.StatusConflict, "reged")
		return
	}
	ne := &model.TUser{
		Id:        utils.NewXid(),
		Name:      name,
		Pass:      utils.Md5String(pass),
		Nick:      nick,
		Created:   time.Now(),
		LoginTime: time.Now(),
		Active:    1,
	}
	/*if pmUser{
		ne.NewUser=1
	}
	if pmOrg{
		ne.NewOrg=1
	}
	if pmPipe{
		ne.NewPipe=1
	}*/
	_, err := comm.Db.Context(ctx).InsertOne(ne)
	if err != nil {
		util.RespInternalErr(c, "create user", err)
		return
	}
	c.String(http.StatusOK, ne.Id)
}

func (UserController) info(c *gin.Context, m *hbtp.Map) {
	id := m.GetString("id")
	if id == "" {
		c.String(http.StatusBadRequest, "param err")
		return
	}
	ctx := c.Request.Context()
	usr := &model.TUser{}
	ok := service.GetIdOrAidCtx(ctx, id, usr)
	if !ok {
		c.String(http.StatusNotFound, "not found user")
		return
	}
	uinfo, _ := service.GetUserInfoCtx(ctx, usr.Id)
	c.JSON(http.StatusOK, hbtp.Map{
		"user": usr,
		"info": uinfo,
	})
}
func (UserController) upinfo(c *gin.Context, m *hbtp.Map) {
	id := m.GetString("id")
	nick := strings.TrimSpace(m.GetString("nick"))
	phone := m.GetString("phone")
	email := m.GetString("email")
	remark := m.GetString("remark")
	if id == "" || nick == "" {
		c.String(http.StatusBadRequest, "param err")
		return
	}
	ctx := c.Request.Context()
	usr := &model.TUser{}
	ok := service.GetIdOrAidCtx(ctx, id, usr)
	if !ok {
		c.String(http.StatusNotFound, "not found user")
		return
	}
	lgusr := service.GetMidLgUser(c)
	if !service.IsAdmin(lgusr) && usr.Id != lgusr.Id {
		c.String(http.StatusMethodNotAllowed, "is not you")
		return
	}
	uinfo, isup := service.GetUserInfoCtx(ctx, usr.Id)
	usr.Nick = nick
	_, err := comm.Db.Context(ctx).Cols("nick").Where("id=?", usr.Id).Update(usr)
	if err != nil {
		util.RespInternalErr(c, "update user nick", err)
		return
	}
	uinfo.Phone = phone
	uinfo.Email = email
	uinfo.Remark = remark
	if isup {
		_, err = comm.Db.Context(ctx).Cols("phone", "email", "remark").
			Where("id=?", usr.Id).Update(uinfo)
	} else {
		uinfo.Id = usr.Id
		_, err = comm.Db.Context(ctx).InsertOne(uinfo)
	}
	if err != nil {
		util.RespInternalErr(c, "update user info", err)
		return
	}
	service.ClearUserCache(usr.Id)
	c.String(http.StatusOK, usr.Id)
}
func (UserController) upass(c *gin.Context, m *hbtp.Map) {
	id := m.GetString("id")
	olds := m.GetString("olds")
	pass := m.GetString("pass")
	if id == "" || pass == "" {
		c.String(http.StatusBadRequest, "param err")
		return
	}
	ctx := c.Request.Context()
	lgusr := service.GetMidLgUser(c)
	usr := &model.TUser{}
	if id == lgusr.Id {
		usr = lgusr
	} else {
		ok := service.GetIdOrAidCtx(ctx, id, usr)
		if !ok {
			c.String(http.StatusNotFound, "not found user")
			return
		}
	}

	if comm.NotUpPass && !service.IsAdmin(lgusr) {
		c.String(http.StatusForbidden, "can't update")
		return
	}
	if usr.Id == lgusr.Id {
		if olds == "" {
			c.String(http.StatusBadRequest, "param err1")
			return
		}
		if usr.Pass != utils.Md5String(olds) {
			c.String(http.StatusUnauthorized, "old pass err")
			return
		}
	} else if !service.IsAdmin(lgusr) {
		c.String(http.StatusMethodNotAllowed, "is not admin")
		return
	}

	usr.Pass = utils.Md5String(pass)
	_, err := comm.Db.Context(ctx).Cols("pass").Where("id=?", usr.Id).Update(usr)
	if err != nil {
		util.RespInternalErr(c, "update user password", err)
		return
	}
	service.ClearUserCache(usr.Id)
	c.String(http.StatusOK, usr.Id)
}
func (UserController) active(c *gin.Context, m *hbtp.Map) {
	id := m.GetString("id")
	act := m.GetString("act")
	if id == "" || act == "" {
		c.String(http.StatusBadRequest, "param err")
		return
	}
	ctx := c.Request.Context()
	lgusr := service.GetMidLgUser(c)
	if !service.IsAdmin(lgusr) {
		c.String(http.StatusMethodNotAllowed, "is not admin")
		return
	}
	usr := &model.TUser{}
	ok := service.GetIdOrAidCtx(ctx, id, usr)
	if !ok {
		c.String(http.StatusNotFound, "not found user")
		return
	}
	if act == "1" {
		usr.Active = 1
	} else {
		usr.Active = 0
	}
	_, err := comm.Db.Context(ctx).Cols("active").Where("id=?", usr.Id).Update(usr)
	if err != nil {
		util.RespInternalErr(c, "update user active status", err)
		return
	}
	service.ClearUserCache(usr.Id)
	c.String(http.StatusOK, usr.Id)
}
func (UserController) perm(c *gin.Context, m *hbtp.Map) {
	id := m.GetString("id")
	permUser := m.GetBool("permUser")
	permOrg := m.GetBool("permOrg")
	permPipe := m.GetBool("permPipe")
	if id == "" {
		c.String(http.StatusBadRequest, "param err")
		return
	}
	ctx := c.Request.Context()
	lgusr := service.GetMidLgUser(c)
	if !service.IsAdmin(lgusr) {
		uf, ok := service.GetUserInfoCtx(ctx, lgusr.Id)
		if !ok || uf.PermUser != 1 {
			c.String(http.StatusMethodNotAllowed, "no permission")
			return
		}
	}
	usr := &model.TUser{}
	ok := service.GetIdOrAidCtx(ctx, id, usr)
	if !ok {
		c.String(http.StatusNotFound, "not found user")
		return
	}
	uinfo, isup := service.GetUserInfoCtx(ctx, usr.Id)
	if permUser {
		uinfo.PermUser = 1
	} else {
		uinfo.PermUser = 0
	}
	if permOrg {
		uinfo.PermOrg = 1
	} else {
		uinfo.PermOrg = 0
	}
	if permPipe {
		uinfo.PermPipe = 1
	} else {
		uinfo.PermPipe = 0
	}
	var err error
	if isup {
		_, err = comm.Db.Context(ctx).Cols("perm_user", "perm_org", "perm_pipe").
			Where("id=?", usr.Id).Update(uinfo)
	} else {
		uinfo.Id = usr.Id
		_, err = comm.Db.Context(ctx).InsertOne(uinfo)
	}
	if err != nil {
		util.RespInternalErr(c, "update user permissions", err)
		return
	}
	service.ClearUserCache(usr.Id)
	c.String(http.StatusOK, usr.Id)
}
