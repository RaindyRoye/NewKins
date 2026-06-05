package route

import (
	"encoding/json"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gokins/core/utils"
	"github.com/gokins/gokins/bean"
	"github.com/gokins/gokins/comm"
	"github.com/gokins/gokins/engine"
	"github.com/gokins/gokins/model"
	"github.com/gokins/gokins/service"
	"github.com/gokins/gokins/util"
	hbtp "github.com/mgr9525/HyperByte-Transfer-Protocol"
)

type TriggerController struct{}

func (TriggerController) GetPath() string {
	return "/api/trigger"
}
func (c *TriggerController) Routes(g gin.IRoutes) {
	g.Use(service.MidUserCheck)
	g.POST("/triggers", util.GinReqParseJson(c.triggers))
	g.POST("/save", util.GinReqParseJson(c.save))
	g.POST("/delete", util.GinReqParseJson(c.delete))
	g.POST("/runs", util.GinReqParseJson(c.runs))
}

func (TriggerController) triggers(c *gin.Context, m *hbtp.Map) {
	pipelineId := m.GetString("pipelineId")
	types := m.GetString("types")
	q := m.GetString("q")
	pg, _ := m.GetInt("page")
	if pipelineId == "" {
		c.String(500, "param err")
		return
	}
	lgusr := service.GetMidLgUser(c)
	perm := service.NewPipePerm(lgusr, pipelineId)
	if perm.Pipeline() == nil {
		c.String(404, "流水线不存在")
		return
	}
	if !perm.IsAdmin() {
		if !perm.CanRead() {
			c.String(405, "No Auth")
			return
		}
	}
	ctx := c.Request.Context()
	ls := make([]*model.TTrigger, 0)
	session := comm.Db.Context(ctx)
	if pipelineId != "" {
		session.And("pipeline_id = ?", pipelineId)
	}
	if types != "" {
		session.And("types = ?", types)
	}
	if q != "" {
		session.And("name like ?", "%"+q+"%")
	}
	page, err := comm.FindPage(session, &ls, pg)
	if err != nil {
		util.RespInternalErr(c, "db operation", err)
		return
	}
	for _, v := range ls {
		usr, ok := service.GetUserCtx(ctx, v.Uid)
		if ok {
			v.Nick = usr.Nick
			v.Avat = usr.Avatar
		}
		_ = json.Unmarshal([]byte(v.Params), &v.Param)
	}
	ms := map[string]interface{}{}
	ms["page"] = page
	ms["host"] = comm.Cfg.Server.Host
	c.JSON(200, ms)
}

func (TriggerController) save(c *gin.Context, tp *bean.TriggerParam) {
	ctx := c.Request.Context()
	if err := tp.Check(); err != nil {
		util.RespErr(c, 400, "validation error", err)
		return
	}
	lgusr := service.GetMidLgUser(c)
	perm := service.NewPipePerm(lgusr, tp.PipelineId)
	if perm.Pipeline() == nil {
		c.String(404, "流水线不存在")
		return
	}
	if !perm.IsAdmin() && !perm.CanWrite() {
		c.String(405, "No Auth")
		return
	}
	tt := &model.TTrigger{}
	err := utils.Struct2Struct(tt, tp)
	if err != nil {
		util.RespInternalErr(c, "struct conversion", err)
		return
	}
	if tp.Enabled {
		tt.Enabled = 1
	}
	if tp.Id == "" {
		tt.Id = utils.NewXid()
		tt.Created = time.Now()
		tt.Uid = lgusr.Id
		_, err = comm.Db.Context(ctx).InsertOne(tt)
		if err != nil {
			util.RespInternalErr(c, "db operation", err)
			return
		}
	} else {
		tt.Updated = time.Now()
		_, err = comm.Db.Context(ctx).Cols("name,desc,params,types,enabled,updated").Where("id =?", tt.Id).Update(tt)
		if err != nil {
			util.RespInternalErr(c, "db operation", err)
			return
		}
	}
	if tt.Types == "timer" {
		if err := engine.Mgr.TimerEng().Refresh(tt.Id); err != nil {
			util.RespInternalErr(c, "timer refresh", err)
			return
		}
	}
	c.JSON(200, "ok")
}

func (TriggerController) delete(c *gin.Context, m *hbtp.Map) {
	ctx := c.Request.Context()
	id := m.GetString("id")
	tt := &model.TTrigger{}
	ok, err := comm.Db.Context(ctx).Where("id = ?", id).Get(tt)
	if err != nil {
		util.RespInternalErr(c, "query trigger", err)
		return
	}
	if !ok {
		c.String(404, "触发器不存在")
		return
	}
	lgusr := service.GetMidLgUser(c)
	perm := service.NewPipePerm(lgusr, tt.PipelineId)
	if perm.Pipeline() == nil {
		c.String(404, "流水线不存在")
		return
	}
	if !perm.IsAdmin() && !perm.CanWrite() {
		c.String(405, "No Auth")
		return
	}
	_, err = comm.Db.Context(ctx).Where("id = ?", tt.Id).Delete(tt)
	if err != nil {
		util.RespInternalErr(c, "db operation", err)
		return
	}
	tr := model.TTriggerRun{}
	_, err = comm.Db.Context(ctx).Where("tid = ?", tt.Id).Delete(tr)
	if err != nil {
		util.RespInternalErr(c, "db operation", err)
		return
	}

	if tt.Types == "timer" {
		engine.Mgr.TimerEng().Delete(tt.Id)
	}
	c.JSON(200, "ok")
}

func (TriggerController) runs(c *gin.Context, m *hbtp.Map) {
	ctx := c.Request.Context()
	id := m.GetString("id")
	pg, _ := m.GetInt("page")
	tt := &model.TTrigger{}
	ok, err := comm.Db.Context(ctx).Where("id = ?", id).Get(tt)
	if err != nil {
		util.RespInternalErr(c, "query trigger", err)
		return
	}
	if !ok {
		c.String(404, "触发器不存在")
		return
	}
	lgusr := service.GetMidLgUser(c)
	perm := service.NewPipePerm(lgusr, tt.PipelineId)
	if perm.Pipeline() == nil {
		c.String(404, "流水线不存在")
		return
	}
	if !perm.IsAdmin() && !perm.CanRead() {
		c.String(405, "No Auth")
		return
	}
	var ls []*model.TTriggerRun
	session := comm.Db.Context(ctx).Where("tid = ?", tt.Id).Desc("created")
	page, err := comm.FindPage(session, &ls, pg)
	if err != nil {
		util.RespInternalErr(c, "db operation", err)
		return
	}
	for _, v := range ls {
		if v.Error != "" || v.PipeVersionId == "" {
			continue
		}
		rpv := &model.RunPipelineVersion{}
		ok, _ = comm.Db.Context(ctx).Table("t_pipeline_version").
			Where("t_pipeline_version.id = ?", v.PipeVersionId).
			Join("left", "t_build", "t_build.pipeline_version_id = ?", v.PipeVersionId).
			Get(rpv)
		if ok {
			v.Number = rpv.Number
			v.PipelineName = rpv.PipelineName
			v.PipelineDisplayName = rpv.PipelineDisplayName
			v.BStatus = rpv.Status
		}
	}
	c.JSON(200, page)
}
