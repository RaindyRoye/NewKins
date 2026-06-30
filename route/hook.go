package route

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gokins/gokins/comm"
	"github.com/gokins/gokins/engine"
	"github.com/gokins/gokins/model"
	"github.com/gokins/gokins/service"
	"github.com/gokins/gokins/util"
	hbtp "github.com/mgr9525/HyperByte-Transfer-Protocol"
)

type HookController struct {
}

// hookRateLimiter limits webhook requests to 30 per minute per IP to prevent abuse.
var hookRateLimiter = util.NewRateLimiter(30, time.Minute)

func (HookController) GetPath() string {
	return "/trigger"
}
func (c *HookController) Routes(g gin.IRoutes) {
	g.POST("/hook/:triggerId", util.MidRateLimit(hookRateLimiter), c.hooks)
	g.POST("/web/:triggerId", util.MidRateLimit(hookRateLimiter), util.GinReqParseJson(c.web))
}
func (HookController) hooks(c *gin.Context) {
	triggerId := c.Param("triggerId")
	if triggerId == "" {
		c.String(http.StatusBadRequest, "param err")
		return
	}
	tt := &model.TTrigger{}
	ok, err := comm.Db.Context(c.Request.Context()).Where("id = ? and enabled != 0", triggerId).Get(tt)
	if err != nil {
		util.RespInternalErr(c, "query trigger", err)
		return
	}
	if !ok {
		c.String(http.StatusNotFound, "触发器不存在或者未激活")
		return
	}
	rb, err := service.TriggerHook(tt, c.Request)
	if err != nil {
		util.RespInternalErr(c, "trigger hook", err)
		return
	}
	engine.Mgr.BuildEgn().Put(rb)
	c.JSON(http.StatusOK, gin.H{
		"msg": "ok",
	})
}

func (HookController) web(c *gin.Context, m *hbtp.Map) {
	triggerId := c.Param("triggerId")
	secret := m.GetString("secret")
	if triggerId == "" || secret == "" {
		c.String(http.StatusBadRequest, "param err")
		return
	}
	tt := &model.TTrigger{}
	ok, err := comm.Db.Context(c.Request.Context()).Where("id = ? and enabled != 0", triggerId).Get(tt)
	if err != nil {
		util.RespInternalErr(c, "query trigger", err)
		return
	}
	if !ok {
		c.String(http.StatusNotFound, "触发器不存在或者未激活")
		return
	}
	rb, err := service.TriggerWeb(c.Request.Context(), tt, secret)
	if err != nil {
		util.RespInternalErr(c, "trigger web", err)
		return
	}
	engine.Mgr.BuildEgn().Put(rb)
	c.JSON(http.StatusOK, gin.H{
		"msg": "ok",
	})
}
