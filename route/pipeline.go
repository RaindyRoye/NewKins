package route

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/gokins/gokins/engine"
	"github.com/gokins/gokins/model"

	"github.com/gin-gonic/gin"
	"github.com/gokins/core/utils"
	"github.com/gokins/gokins/bean"
	"github.com/gokins/gokins/comm"
	"github.com/gokins/gokins/service"
	"github.com/gokins/gokins/util"
	hbtp "github.com/mgr9525/HyperByte-Transfer-Protocol"
	"github.com/sirupsen/logrus"
	"gopkg.in/yaml.v3"
)

type PipelineController struct{}

func (PipelineController) GetPath() string {
	return "/api/pipeline"
}
func (c *PipelineController) Routes(g gin.IRoutes) {
	g.Use(service.MidUserCheck)
	g.POST("/org/pipelines", util.GinReqParseJson(c.orgPipelines))
	g.POST("/pipelines", util.GinReqParseJson(c.getPipelines))
	g.POST("/new", util.GinReqParseJson(c.new))
	g.POST("/delete", util.GinReqParseJson(c.delete))
	g.POST("/info", util.GinReqParseJson(c.info))
	g.POST("/save", util.GinReqParseJson(c.save))
	g.POST("/run", util.GinReqParseJson(c.run))
	g.POST("/copy", util.GinReqParseJson(c.copy))
	g.POST("/rebuild", util.GinReqParseJson(c.rebuild))
	g.POST("/pipelineVersions", util.GinReqParseJson(c.pipelineVersions))
	g.POST("/pipelineVersion", util.GinReqParseJson(c.pipelineVersion))
	g.POST("/search/sha", util.GinReqParseJson(c.searchSha))
	g.POST("/vars", util.GinReqParseJson(c.vars))
	g.POST("/var/save", util.GinReqParseJson(c.varSave))
	g.POST("/var/del", util.GinReqParseJson(c.varDel))
}
func (PipelineController) orgPipelines(c *gin.Context, m *hbtp.Map) {
	orgId := m.GetString("orgId")
	q := m.GetString("q")
	pg, _ := m.GetInt("page")
	if orgId == "" {
		c.String(http.StatusBadRequest, "param err")
		return
	}
	lgusr := service.GetMidLgUser(c)
	perm := service.NewOrgPermCtx(c.Request.Context(), lgusr, orgId)
	if perm.Org() == nil || perm.Org().Deleted == 1 {
		c.String(http.StatusNotFound, "not found org")
		return
	}
	if !perm.CanRead() {
		c.String(http.StatusMethodNotAllowed, "No Auth")
		return
	}
	ls := make([]*model.TPipeline, 0)
	var err error
	var page *bean.Page
	// if comm.IsMySQL {
	gen := &bean.PageGen{
		CountCols: "DISTINCT(pipe.id)",
		FindCols:  "DISTINCT(pipe.id),pipe.*",
	}
	gen.SQL = `
			select {{select}} from t_pipeline pipe
			LEFT JOIN t_org_pipe top on pipe.id = top.pipe_id
			where top.org_id = ? and pipe.deleted != 1
		    `
	gen.Args = append(gen.Args, perm.Org().Id)
	if q != "" {
		gen.SQL += "\nAND pipe.name like ? "
		gen.Args = append(gen.Args, "%"+q+"%")
	}
	gen.SQL += "\nORDER BY pipe.id DESC"
	page, err = comm.FindPagesCtx(c.Request.Context(), gen, &ls, pg, 10)
	if err != nil {
		util.RespInternalErr(c, "db operation", err)
		return
	}
	//}
	if err := fillPipelineListBuildInfo(c.Request.Context(), ls); err != nil {
		util.RespInternalErr(c, "fill pipeline info", err)
		return
	}
	c.JSON(http.StatusOK, page)
}
func (PipelineController) getPipelines(c *gin.Context, m *hbtp.Map) {
	q := m.GetString("q")
	pg, _ := m.GetInt("page")
	lgusr := service.GetMidLgUser(c)
	ls := make([]*model.TPipeline, 0)
	var err error
	var page *bean.Page
	// if comm.IsMySQL {
	gen := &bean.PageGen{
		CountCols: "pipe.id",
		FindCols:  "pipe.*",
	}
	gen.SQL = `
			select {{select}} from t_pipeline pipe where pipe.deleted != 1 `
	if !service.IsAdmin(lgusr) {
		gen.SQL += ` and pipe.uid = ? `
		gen.Args = append(gen.Args, lgusr.Id)
	}
	if q != "" {
		gen.SQL += "\nAND pipe.name like ? "
		gen.Args = append(gen.Args, "%"+q+"%")
	}
	gen.SQL += "\nORDER BY pipe.id DESC"
	page, err = comm.FindPagesCtx(c.Request.Context(), gen, &ls, pg, 10)
	if err != nil {
		util.RespInternalErr(c, "db operation", err)
		return
	}
	//}
	if err := fillPipelineListBuildInfo(c.Request.Context(), ls); err != nil {
		util.RespInternalErr(c, "fill pipeline info", err)
		return
	}
	c.JSON(http.StatusOK, page)
}

func (PipelineController) save(c *gin.Context, m *hbtp.Map) {
	name := m.GetString("name")
	content := m.GetString("content")
	pipelineId := m.GetString("pipelineId")
	accessToken := m.GetString("accessToken")
	ul := m.GetString("url")
	username := m.GetString("username")
	displayName := m.GetString("displayName")
	if pipelineId == "" {
		c.String(http.StatusBadRequest, "param err")
		return
	}
	ctx := c.Request.Context()
	usr := service.GetMidLgUser(c)
	perm := service.NewPipePermCtx(ctx, usr, pipelineId)
	if !perm.CanWrite() {
		c.String(http.StatusMethodNotAllowed, "No Auth")
		return
	}
	y := &bean.Pipeline{}
	if err := yaml.Unmarshal([]byte(content), y); err != nil {
		util.RespErr(c, http.StatusBadRequest, "yaml parse error", err)
		return
	}
	err := y.Check()
	if err != nil {
		util.RespErr(c, http.StatusBadRequest, "yaml validation error", err)
		return
	}
	pipeline := &model.TPipeline{
		Name:        name,
		DisplayName: displayName,
	}
	_, err = comm.Db.Context(ctx).Cols("name,display_name").Where("id = ?", pipelineId).Update(pipeline)
	if err != nil {
		util.RespInternalErr(c, "db operation", err)
		return
	}
	tpc := &model.TPipelineConf{
		YmlContent:  content,
		Url:         ul,
		Username:    username,
		AccessToken: accessToken,
	}
	_, err = comm.Db.Context(ctx).Cols("yml_content,url,username,access_token").Where("pipeline_id = ?", pipelineId).Update(tpc)
	if err != nil {
		util.RespInternalErr(c, "db operation", err)
		return
	}
	c.String(http.StatusOK, "ok")
}
func (PipelineController) delete(c *gin.Context, m *hbtp.Map) {
	id := m.GetString("id")
	if id == "" {
		c.String(http.StatusBadRequest, "param err")
		return
	}
	ctx := c.Request.Context()
	usr := service.GetMidLgUser(c)
	perm := service.NewPipePermCtx(ctx, usr, id)
	if perm.Pipeline() == nil || perm.Pipeline().Deleted == 1 {
		c.String(http.StatusNotFound, "未找到流水线信息")
		return
	}
	if !perm.CanWrite() {
		c.String(http.StatusMethodNotAllowed, "No Auth")
		return
	}
	tp := &model.TPipeline{
		Deleted:     1,
		DeletedTime: time.Now(),
	}
	_, err := comm.Db.Context(ctx).Cols("deleted").Where("id = ?", id).Update(tp)
	if err != nil {
		util.RespInternalErr(c, "pipeline delete", err)
		return
	}
	version := &model.TPipelineVersion{
		Deleted: 1,
	}
	_, err = comm.Db.Context(ctx).Cols("deleted").Where("pipeline_id = ?", id).Update(version)
	if err != nil {
		util.RespInternalErr(c, "pipeline delete", err)
		return
	}
	c.String(http.StatusOK, "ok")
}
func (PipelineController) new(c *gin.Context, npipe *bean.NewPipeline) {
	if !npipe.Check() {
		c.String(http.StatusBadRequest, "param err")
		return
	}
	y := &bean.Pipeline{}
	err := yaml.Unmarshal([]byte(npipe.Content), y)
	if err != nil {
		util.RespErr(c, http.StatusBadRequest, "yaml parse error", err)
		return
	}
	err = y.Check()
	if err != nil {
		util.RespErr(c, http.StatusBadRequest, "yaml validation error", err)
		return
	}
	ctx := c.Request.Context()
	lgusr := service.GetMidLgUser(c)
	perm := service.NewOrgPermCtx(ctx, lgusr, npipe.OrgId)
	if npipe.OrgId != "" && perm.Org() == nil {
		c.String(http.StatusNotFound, "组织不存在")
		return
	}
	if !perm.IsAdmin() {
		uf, ok := service.GetUserInfoCtx(ctx, lgusr.Id)
		if !ok || uf.PermPipe != 1 {
			c.String(http.StatusMethodNotAllowed, "no permission")
			return
		}
		if perm.Org() != nil && !perm.CanWrite() {
			c.String(http.StatusMethodNotAllowed, "No Auth")
			return
		}
	}
	pipeline := &model.TPipeline{
		Id:           utils.NewXid(),
		Uid:          lgusr.Id,
		Name:         npipe.Name,
		DisplayName:  npipe.DisplayName,
		PipelineType: "",
	}
	_, err = comm.Db.Context(ctx).InsertOne(pipeline)
	if err != nil {
		util.RespInternalErr(c, "db operation", err)
		return
	}
	tpc := &model.TPipelineConf{
		PipelineId:  pipeline.Id,
		YmlContent:  npipe.Content,
		Url:         npipe.Url,
		Username:    npipe.Username,
		AccessToken: npipe.AccessToken,
	}
	_, err = comm.Db.Context(ctx).InsertOne(tpc)
	if err != nil {
		util.RespInternalErr(c, "db operation", err)
		return
	}
	if len(npipe.Vars) > 0 {
		for _, v := range npipe.Vars {
			pipelineVar := &model.TPipelineVar{}
			err = utils.Struct2Struct(pipelineVar, v)
			if err != nil {
				util.RespInternalErr(c, "struct conversion", err)
				return
			}
			pipelineVar.Uid = lgusr.Id
			pipelineVar.PipelineId = pipeline.Id
			if v.Public {
				pipelineVar.Public = 1
			}
			_, err = comm.Db.Context(ctx).InsertOne(pipelineVar)
			if err != nil {
				util.RespInternalErr(c, "db operation", err)
				return
			}
		}
	}
	if perm.Org() != nil {
		top := &model.TOrgPipe{
			OrgId:   perm.Org().Id,
			PipeId:  pipeline.Id,
			Created: time.Now(),
			Public:  0,
		}
		_, err = comm.Db.Context(ctx).InsertOne(top)
		if err != nil {
			util.RespInternalErr(c, "db operation", err)
			return
		}
	}
	c.JSON(http.StatusOK, pipeline)
}

func (PipelineController) info(c *gin.Context, m *hbtp.Map) {
	id := m.GetString("id")
	if id == "" {
		c.String(http.StatusBadRequest, "param err")
		return
	}
	ctx := c.Request.Context()
	lgusr := service.GetMidLgUser(c)
	perm := service.NewPipePermCtx(ctx, lgusr, id)
	if perm.Pipeline() == nil || perm.Pipeline().Deleted == 1 {
		c.String(http.StatusNotFound, "未找到流水线信息")
		return
	}
	if !perm.CanRead() {
		c.String(http.StatusMethodNotAllowed, "No Auth")
		return
	}
	pipe := &model.TPipelineInfo{}
	ok, err := comm.Db.Context(ctx).Where("id=? and deleted != 1", id).Get(pipe)
	if err != nil {
		util.RespInternalErr(c, "query pipeline", err)
		return
	}
	if !ok {
		c.String(http.StatusNotFound, "未找到流水线信息")
		return
	}
	tpc := &model.TPipelineConf{}
	_, err = comm.Db.Context(ctx).Where("pipeline_id=?", pipe.Id).Get(tpc)
	if err != nil {
		util.RespInternalErr(c, "db operation", err)
		return
	}
	pipe.YmlContent = tpc.YmlContent
	pipe.Url = tpc.Url
	s := comm.MaskedValue
	if perm.CanWrite() {
		pipe.Username = tpc.Username
		pipe.AccessToken = tpc.AccessToken
	} else {
		pipe.Username = s
		pipe.AccessToken = s
	}
	c.JSON(http.StatusOK, hbtp.Map{
		"pipe": pipe,
		"perm": hbtp.Map{
			"read":  perm.CanRead(),
			"write": perm.CanWrite(),
			"exec":  perm.CanExec(),
		},
	})
}

func (PipelineController) run(c *gin.Context, m *hbtp.Map) {
	pipelineId := m.GetString("pipelineId")
	sha := m.GetString("sha")
	if pipelineId == "" {
		c.String(http.StatusBadRequest, "param err")
		return
	}
	lgusr := service.GetMidLgUser(c)
	perm := service.NewPipePermCtx(c.Request.Context(), lgusr, pipelineId)
	if perm.Pipeline() == nil || perm.Pipeline().Deleted == 1 {
		c.String(http.StatusNotFound, "未找到流水线信息")
		return
	}
	if !perm.CanExec() {
		c.String(http.StatusMethodNotAllowed, "No Auth")
		return
	}
	tvp, rb, err := service.Run(c.Request.Context(), lgusr.Id, pipelineId, sha, "run")
	if err != nil {
		util.RespInternalErr(c, "pipeline run", err)
		return
	}
	engine.Mgr.BuildEgn().Put(rb)
	c.JSON(http.StatusOK, tvp)
}

func (PipelineController) copy(c *gin.Context, m *hbtp.Map) {
	pipelineId := m.GetString("pipelineId")
	if pipelineId == "" {
		c.String(http.StatusBadRequest, "param err")
		return
	}
	ctx := c.Request.Context()
	lgusr := service.GetMidLgUser(c)
	perm := service.NewPipePermCtx(ctx, lgusr, pipelineId)
	if perm.Pipeline() == nil || perm.Pipeline().Deleted == 1 {
		c.String(http.StatusNotFound, "未找到流水线信息")
		return
	}
	if !perm.CanRead() {
		c.String(http.StatusMethodNotAllowed, "No Auth")
		return
	}
	if !perm.IsAdmin() {
		uf, ok := service.GetUserInfoCtx(ctx, lgusr.Id)
		if !ok || uf.PermPipe != 1 {
			c.String(http.StatusMethodNotAllowed, "no permission")
			return
		}
	}
	pipe := &model.TPipeline{
		Id:           utils.NewXid(),
		Uid:          lgusr.Id,
		Name:         fmt.Sprintf("%s_copy", perm.Pipeline().Name),
		DisplayName:  perm.Pipeline().DisplayName,
		PipelineType: perm.Pipeline().PipelineType,
	}
	_, err := comm.Db.Context(ctx).InsertOne(pipe)
	if err != nil {
		util.RespInternalErr(c, "db operation", err)
		return
	}

	tpc := &model.TPipelineConf{}
	_, err = comm.Db.Context(ctx).Where("pipeline_id=?", perm.Pipeline().Id).Get(tpc)
	if err != nil {
		util.RespInternalErr(c, "db operation", err)
		return
	}
	ne := &model.TPipelineConf{
		PipelineId:  pipe.Id,
		Url:         tpc.Url,
		AccessToken: tpc.AccessToken,
		YmlContent:  tpc.YmlContent,
		Username:    tpc.Username,
	}
	_, err = comm.Db.Context(ctx).InsertOne(ne)
	if err != nil {
		util.RespInternalErr(c, "db operation", err)
		return
	}
	c.JSON(http.StatusOK, pipe)
}
func (PipelineController) rebuild(c *gin.Context, m *hbtp.Map) {
	pipelineVersionId := m.GetString("pipelineVersionId")
	if pipelineVersionId == "" {
		c.String(http.StatusBadRequest, "param err")
		return
	}
	tvp := &model.TPipelineVersion{}
	ok, err := comm.Db.Context(c.Request.Context()).Where("id=? and deleted != 1", pipelineVersionId).Get(tvp)
	if err != nil {
		util.RespInternalErr(c, "query pipeline version", err)
		return
	}
	if !ok {
		c.String(http.StatusNotFound, "构建记录不存在")
		return
	}
	lgusr := service.GetMidLgUser(c)
	perm := service.NewPipePermCtx(c.Request.Context(), lgusr, tvp.PipelineId)
	if perm.Pipeline() == nil || perm.Pipeline().Deleted == 1 {
		c.String(http.StatusNotFound, "未找到流水线信息")
		return
	}
	if !perm.CanExec() {
		c.String(http.StatusMethodNotAllowed, "No Permission")
		return
	}
	tvp, rb, err := service.ReBuild(c.Request.Context(), lgusr.Id, tvp)
	if err != nil {
		util.RespInternalErr(c, "pipeline rebuild", err)
		return
	}
	engine.Mgr.BuildEgn().Put(rb)
	c.JSON(http.StatusOK, tvp)
}

func (PipelineController) pipelineVersions(c *gin.Context, m *hbtp.Map) {
	pipelineId := m.GetString("pipelineId")
	pg, _ := m.GetInt("page")
	usr := service.GetMidLgUser(c)
	ls := make([]*model.TPipelineVersion, 0)
	var page *bean.Page
	var err error
	if pipelineId != "" {
		perm := service.NewPipePermCtx(c.Request.Context(), usr, pipelineId)
		if perm.Pipeline() == nil || perm.Pipeline().Deleted == 1 {
			c.String(http.StatusNotFound, "未找到流水线信息")
			return
		}
		if !perm.CanRead() {
			c.String(http.StatusMethodNotAllowed, "No Auth")
			return
		}
		where := comm.Db.Context(c.Request.Context()).Where("pipeline_id = ? and deleted != 1", pipelineId).Desc("id")
		page, err = comm.FindPage(where, &ls, pg)
		if err != nil {
			util.RespInternalErr(c, "db operation", err)
			return
		}
	} else {
		if service.IsAdmin(usr) {
			where := comm.Db.Context(c.Request.Context()).Where(" deleted != 1").Desc("id")
			page, err = comm.FindPage(where, &ls, pg)
			if err != nil {
				util.RespInternalErr(c, "db operation", err)
				return
			}
		} else {
			tpipeIds := []string{}
			err = comm.Db.Context(c.Request.Context()).Table(&model.TPipeline{}).Cols("id").Where("uid = ? and deleted != 1", usr.Id).Find(&tpipeIds)
			if err != nil {
				util.RespInternalErr(c, "db operation", err)
				return
			}
			if len(tpipeIds) == 0 {
				c.JSON(http.StatusOK, page)
				return
			}
			where := comm.Db.Context(c.Request.Context()).In("pipeline_id", tpipeIds).Where("deleted != 1").Desc("id")
			page, err = comm.FindPage(where, &ls, pg, 20)
			if err != nil {
				util.RespInternalErr(c, "db operation", err)
				return
			}
		}
	}

	// Batch load latest builds for all pipeline versions (eliminates N+1 queries)
	versionIds := make([]string, len(ls))
	for i, v := range ls {
		versionIds[i] = v.Id
	}
	latestBuilds, err := service.BatchLatestBuildsForVersions(c.Request.Context(), versionIds)
	if err != nil {
		util.RespInternalErr(c, "batch load builds", err)
		return
	}
	for _, v := range ls {
		if b, ok := latestBuilds[v.Id]; ok {
			v.Build = b
		}
	}

	c.JSON(http.StatusOK, page)
}
func (PipelineController) pipelineVersion(c *gin.Context, m *hbtp.Map) {
	id := m.GetString("id")
	if id == "" {
		c.String(http.StatusBadRequest, "param err")
		return
	}
	ctx := c.Request.Context()
	pv := &model.TPipelineVersion{}
	ok, err := comm.Db.Context(ctx).Where("id=?", id).Get(pv)
	if err != nil {
		util.RespInternalErr(c, "query pipeline version", err)
		return
	}
	if !ok {
		c.String(http.StatusNotFound, "not found pv")
		return
	}
	usr := &model.TUser{}
	service.GetIdOrAidCtx(ctx, pv.Uid, usr)
	build := &model.RunBuild{}
	ok, err = comm.Db.Context(ctx).Where("pipeline_version_id=?", pv.Id).Get(build)
	if err != nil {
		logrus.Warnf("pipeline buildInfo: query build (pv=%s): %v", pv.Id, err)
	}
	if !ok {
		c.String(http.StatusNotFound, "not found build")
		return
	}
	perm := service.NewPipePermCtx(ctx, service.GetMidLgUser(c), pv.PipelineId)
	if perm.Pipeline() == nil {
		c.String(http.StatusNotFound, "not found pipe")
		return
	}
	if !perm.CanRead() {
		c.String(http.StatusMethodNotAllowed, "no permission")
		return
	}

	pipeShow := &bean.PipelineShow{}
	err = utils.Struct2Struct(pipeShow, perm.Pipeline())
	if err != nil {
		util.RespInternalErr(c, "struct conversion pipeline show", err)
		return
	}
	pinfo := &model.TPipelineConf{}
	ok, err = comm.Db.Context(ctx).Where("pipeline_id=?", perm.Pipeline().Id).Get(pinfo)
	if err != nil {
		logrus.Warnf("pipeline buildInfo: query pipeline conf (pipe=%s): %v", perm.Pipeline().Id, err)
	}
	if ok {
		pipeShow.Url = pinfo.Url
	}
	c.JSON(http.StatusOK, hbtp.Map{
		"build": build,
		"pv":    pv,
		"usr":   usr,
		"pipe":  pipeShow,
		"perm": hbtp.Map{
			"read":  perm.CanRead(),
			"write": perm.CanWrite(),
			"exec":  perm.CanExec(),
		},
	})
}
func (PipelineController) searchSha(c *gin.Context, m *hbtp.Map) {
	id := m.GetString("id")
	q := m.GetString("q")
	if id == "" {
		c.String(http.StatusBadRequest, "param err")
		return
	}
	perm := service.NewPipePermCtx(c.Request.Context(), service.GetMidLgUser(c), id)
	if perm.Pipeline() == nil {
		c.String(http.StatusNotFound, "not found pipe")
		return
	}
	if !perm.CanRead() {
		c.String(http.StatusMethodNotAllowed, "no permission")
		return
	}
	shas := []string{}
	session := comm.Db.Context(c.Request.Context()).Table("t_pipeline_version").
		Distinct("sha").Cols("sha").
		Where("pipeline_id = ?", id).Desc("sha")
	if q != "" {
		session.And("sha like ?", "%"+q+"%")
	}
	err := session.Find(&shas)
	if err != nil {
		util.RespInternalErr(c, "db operation", err)
		return
	}
	res := make([]map[string]string, 0)
	for _, sha := range shas {
		if sha == "" {
			continue
		}
		m2 := map[string]string{}
		m2["name"] = sha
		res = append(res, m2)
	}
	c.JSON(http.StatusOK, res)
}
func (PipelineController) vars(c *gin.Context, m *hbtp.Map) {
	pipelineId := m.GetString("pipelineId")
	q := m.GetString("q")
	pg, _ := m.GetInt("page")
	if pipelineId == "" {
		c.String(http.StatusBadRequest, "param err")
		return
	}
	perm := service.NewPipePermCtx(c.Request.Context(), service.GetMidLgUser(c), pipelineId)
	if perm.Pipeline() == nil {
		c.String(http.StatusNotFound, "not found pipe")
		return
	}
	if !perm.CanRead() {
		c.String(http.StatusMethodNotAllowed, "no permission")
		return
	}
	var ls []*model.TPipelineVar
	var page *bean.Page
	var err error
	session := comm.Db.Context(c.Request.Context()).Where("pipeline_id = ?", pipelineId)
	if q != "" {
		session.And("(name like ? or value like ?)", "%"+q+"%", "%"+q+"%")
	}
	page, err = comm.FindPage(session, &ls, pg)
	if err != nil {
		util.RespInternalErr(c, "db operation", err)
		return
	}
	if !perm.CanWrite() {
		for _, v := range ls {
			if v.Public != 0 {
				v.Value = comm.MaskedValue
			}
		}
	}
	c.JSON(http.StatusOK, page)
}
func (PipelineController) varSave(c *gin.Context, pv *bean.PipelineVar) {
	if pv.Value == "" || pv.Name == "" || pv.PipelineId == "" {
		c.String(http.StatusBadRequest, "param err")
		return
	}
	perm := service.NewPipePermCtx(c.Request.Context(), service.GetMidLgUser(c), pv.PipelineId)
	if perm.Pipeline() == nil {
		c.String(http.StatusNotFound, "not found pipe")
		return
	}
	if !perm.CanWrite() {
		c.String(http.StatusMethodNotAllowed, "no permission")
		return
	}
	pipelineVar := &model.TPipelineVar{}
	err := utils.Struct2Struct(pipelineVar, pv)
	if err != nil {
		util.RespInternalErr(c, "db operation", err)
		return
	}
	if pv.Public {
		pipelineVar.Public = 1
	}
	tpv := &model.TPipelineVar{}
	ok, err := comm.Db.Context(c.Request.Context()).Where("pipeline_id = ? and name = ?", pv.PipelineId, pv.Name).Get(tpv)
	if err != nil {
		util.RespInternalErr(c, "db operation", err)
		return
	}
	if pv.Aid > 0 {
		if ok && tpv.Aid != pv.Aid {
			c.String(http.StatusConflict, "duplicate variable name")
			return
		}
		_, err = comm.Db.Context(c.Request.Context()).Cols("name,value,remarks,public").Where("aid = ?", pv.Aid).Update(pipelineVar)
		if err != nil {
			util.RespInternalErr(c, "db operation", err)
			return
		}
		c.String(http.StatusOK, "ok")
		return
	}
	if ok {
		c.String(http.StatusConflict, "duplicate variable name")
		return
	}
	_, err = comm.Db.Context(c.Request.Context()).InsertOne(pipelineVar)
	if err != nil {
		util.RespInternalErr(c, "db operation", err)
		return
	}
	c.String(http.StatusOK, "ok")
}
func (PipelineController) varDel(c *gin.Context, m *hbtp.Map) {
	aId, err := m.GetInt("aid")
	if err != nil || aId <= 0 {
		c.String(http.StatusBadRequest, "param err")
		return
	}
	pipelineVar := &model.TPipelineVar{}
	ok, err := comm.Db.Context(c.Request.Context()).Where("aid = ? ", aId).Get(pipelineVar)
	if err != nil {
		util.RespInternalErr(c, "query pipeline var", err)
		return
	}
	if !ok {
		c.String(http.StatusNotFound, "not found pipe_var")
		return
	}
	perm := service.NewPipePermCtx(c.Request.Context(), service.GetMidLgUser(c), pipelineVar.PipelineId)
	if perm.Pipeline() == nil {
		c.String(http.StatusNotFound, "not found pipe")
		return
	}
	if !perm.CanWrite() {
		c.String(http.StatusMethodNotAllowed, "no permission")
		return
	}
	_, err = comm.Db.Context(c.Request.Context()).Where("aid = ?", aId).Delete(pipelineVar)
	if err != nil {
		util.RespInternalErr(c, "db operation", err)
		return
	}
	c.String(http.StatusOK, "ok")
}

// fillPipelineListBuildInfo enriches a list of pipelines with user info, build counts,
// and latest build data using batch queries instead of N+1 individual queries.
// This reduces database round-trips from O(N) to O(1) for each page of results.
func fillPipelineListBuildInfo(ctx context.Context, ls []*model.TPipeline) error {
	if len(ls) == 0 {
		return nil
	}

	// Collect unique pipeline IDs and user IDs
	pipelineIds := make([]string, len(ls))
	uidSet := make(map[string]struct{})
	for i, v := range ls {
		pipelineIds[i] = v.Id
		if v.Uid != "" {
			uidSet[v.Uid] = struct{}{}
		}
	}

	// Batch fetch users
	uids := make([]string, 0, len(uidSet))
	for uid := range uidSet {
		uids = append(uids, uid)
	}
	userMap, err := service.BatchGetUsers(ctx, uids)
	if err != nil {
		return fmt.Errorf("batch get users: %w", err)
	}

	// Batch fetch build counts
	countMap, err := service.BatchBuildCounts(ctx, pipelineIds)
	if err != nil {
		return fmt.Errorf("batch build counts: %w", err)
	}

	// Batch fetch latest builds
	buildMap, err := service.BatchLatestBuilds(ctx, pipelineIds)
	if err != nil {
		return fmt.Errorf("batch latest builds: %w", err)
	}

	// Enrich pipelines with batched data
	for _, v := range ls {
		if usr, ok := userMap[v.Uid]; ok {
			v.Nick = usr.Nick
			v.Avat = usr.Avatar
		}
		v.Buildln = countMap[v.Id]
		if b, ok := buildMap[v.Id]; ok {
			v.Build = b
		}
	}

	return nil
}
