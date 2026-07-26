package route

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
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
	"github.com/sirupsen/logrus"
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
		c.String(http.StatusBadRequest, "param err")
		return
	}
	lgusr := service.GetMidLgUser(c)
	perm := service.NewPipePermCtx(c.Request.Context(), lgusr, pipelineId)
	if perm.Pipeline() == nil {
		c.String(http.StatusNotFound, "流水线不存在")
		return
	}
	if !perm.IsAdmin() {
		if !perm.CanRead() {
			c.String(http.StatusMethodNotAllowed, "No Auth")
			return
		}
	}
	ctx := c.Request.Context()
	ls := make([]*model.TTrigger, 0)
	session := comm.Db.Context(ctx).And("pipeline_id = ?", pipelineId)
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
	// Batch fetch users to eliminate N+1 queries
	uidSet := make(map[string]struct{})
	for _, v := range ls {
		if v.Uid != "" {
			uidSet[v.Uid] = struct{}{}
		}
	}
	uids := make([]string, 0, len(uidSet))
	for uid := range uidSet {
		uids = append(uids, uid)
	}
	userMap, err := service.BatchGetUsers(ctx, uids)
	if err != nil {
		logrus.Warnf("trigger list: batch get users: %v", err)
	}
	for _, v := range ls {
		if usr, ok := userMap[v.Uid]; ok {
			v.Nick = usr.Nick
			v.Avat = usr.Avatar
		}
		if err := json.Unmarshal([]byte(v.Params), &v.Param); err != nil {
			logrus.Warnf("trigger list: unmarshal params (trigger=%s): %v", v.Id, err)
		}
	}
	ms := map[string]any{}
	ms["page"] = page
	ms["host"] = comm.Cfg.Server.Host
	c.JSON(http.StatusOK, ms)
}

func (TriggerController) save(c *gin.Context, tp *bean.TriggerParam) {
	ctx := c.Request.Context()
	if err := tp.Check(); err != nil {
		util.RespErr(c, http.StatusBadRequest, "validation error", err)
		return
	}
	lgusr := service.GetMidLgUser(c)
	perm := service.NewPipePermCtx(ctx, lgusr, tp.PipelineId)
	if perm.Pipeline() == nil {
		c.String(http.StatusNotFound, "流水线不存在")
		return
	}
	if !perm.IsAdmin() && !perm.CanWrite() {
		c.String(http.StatusMethodNotAllowed, "No Auth")
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
		if err := engine.Mgr.TimerEng().Refresh(ctx, tt.Id); err != nil {
			util.RespInternalErr(c, "timer refresh", err)
			return
		}
	}
	c.JSON(http.StatusOK, "ok")
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
		c.String(http.StatusNotFound, "触发器不存在")
		return
	}
	lgusr := service.GetMidLgUser(c)
	perm := service.NewPipePermCtx(ctx, lgusr, tt.PipelineId)
	if perm.Pipeline() == nil {
		c.String(http.StatusNotFound, "流水线不存在")
		return
	}
	if !perm.IsAdmin() && !perm.CanWrite() {
		c.String(http.StatusMethodNotAllowed, "No Auth")
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
	c.JSON(http.StatusOK, "ok")
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
		c.String(http.StatusNotFound, "触发器不存在")
		return
	}
	lgusr := service.GetMidLgUser(c)
	perm := service.NewPipePermCtx(ctx, lgusr, tt.PipelineId)
	if perm.Pipeline() == nil {
		c.String(http.StatusNotFound, "流水线不存在")
		return
	}
	if !perm.IsAdmin() && !perm.CanRead() {
		c.String(http.StatusMethodNotAllowed, "No Auth")
		return
	}
	var ls []*model.TTriggerRun
	session := comm.Db.Context(ctx).Where("tid = ?", tt.Id).Desc("created")
	page, err := comm.FindPage(session, &ls, pg)
	if err != nil {
		util.RespInternalErr(c, "db operation", err)
		return
	}
	// Batch fetch pipeline version info to eliminate N+1 queries
	pvIds := make([]string, 0, len(ls))
	for _, v := range ls {
		if v.Error == "" && v.PipeVersionId != "" {
			pvIds = append(pvIds, v.PipeVersionId)
		}
	}
	if len(pvIds) > 0 {
		pvMap, err := batchRunPipelineVersions(ctx, pvIds)
		if err != nil {
			logrus.Warnf("trigger runs: batch query pipeline versions: %v", err)
		}
		for _, v := range ls {
			if rpv, ok := pvMap[v.PipeVersionId]; ok {
				v.Number = rpv.Number
				v.PipelineName = rpv.PipelineName
				v.PipelineDisplayName = rpv.PipelineDisplayName
				v.BStatus = rpv.Status
			}
		}
	}
	c.JSON(http.StatusOK, page)
}

// batchRunPipelineVersions fetches pipeline version info for multiple IDs in a single query,
// eliminating N+1 queries in the trigger runs endpoint.
func batchRunPipelineVersions(ctx context.Context, versionIds []string) (map[string]*model.RunPipelineVersion, error) {
	if len(versionIds) == 0 {
		return map[string]*model.RunPipelineVersion{}, nil
	}
	var versions []*model.RunPipelineVersion
	err := comm.Db.Context(ctx).Table("t_pipeline_version").
		In("t_pipeline_version.id", versionIds).
		Join("left", "t_build", "t_build.pipeline_version_id = t_pipeline_version.id").
		Find(&versions)
	if err != nil {
		return nil, fmt.Errorf("batch run pipeline versions: %w", err)
	}
	result := make(map[string]*model.RunPipelineVersion, len(versions))
	for _, v := range versions {
		result[v.Id] = v
	}
	return result, nil
}
