package route

import (
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
		c.String(500, "param err")
		return
	}
	tt := &model.TTrigger{}
	ok, _ := comm.Db.Where("id = ? and enabled != 0", triggerId).Get(tt)
	if !ok {
		c.String(404, "触发器不存在或者未激活")
		return
	}
	rb, err := service.TriggerHook(tt, c.Request)
	if err != nil {
		util.RespInternalErr(c, "trigger hook", err)
		return
	}
	engine.Mgr.BuildEgn().Put(rb)
	c.JSON(200, gin.H{
		"msg": "ok",
	})
}

func (HookController) web(c *gin.Context, m *hbtp.Map) {
	triggerId := c.Param("triggerId")
	secret := m.GetString("secret")
	if triggerId == "" || secret == "" {
		c.String(500, "param err")
		return
	}
	tt := &model.TTrigger{}
	ok, _ := comm.Db.Where("id = ? and enabled != 0", triggerId).Get(tt)
	if !ok {
		c.String(404, "触发器不存在或者未激活")
		return
	}
	rb, err := service.TriggerWeb(tt, secret)
	if err != nil {
		util.RespInternalErr(c, "trigger web", err)
		return
	}
	engine.Mgr.BuildEgn().Put(rb)
	c.JSON(200, gin.H{
		"msg": "ok",
	})
}
