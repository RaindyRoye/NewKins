package route

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/gokins/gokins/comm"
	"github.com/gokins/gokins/model"
	"github.com/gokins/gokins/service"
	"github.com/gokins/gokins/util"
	hbtp "github.com/mgr9525/HyperByte-Transfer-Protocol"
)

type PipelineVersionController struct{}

func (PipelineVersionController) GetPath() string {
	return "/api/pipelineVersion"
}
func (c *PipelineVersionController) Routes(g gin.IRoutes) {
	g.Use(service.MidUserCheck)
	g.POST("/delete", util.GinReqParseJson(c.delete))
}

func (PipelineVersionController) delete(c *gin.Context, m *hbtp.Map) {
	id := m.GetString("id")
	if id == "" {
		c.String(http.StatusBadRequest, "param err")
		return
	}
	ctx := c.Request.Context()
	tpv := &model.TPipelineVersion{}
	ok, err := comm.Db.Context(ctx).Where("id = ? ", id).Get(tpv)
	if err != nil {
		util.RespInternalErr(c, "query pipeline version", err)
		return
	}
	if !ok {
		c.String(http.StatusNotFound, "not found pipe_var")
		return
	}
	perm := service.NewPipePermCtx(c.Request.Context(), service.GetMidLgUser(c), tpv.PipelineId)
	if perm.Pipeline() == nil {
		c.String(http.StatusNotFound, "not found pipe")
		return
	}
	if !perm.CanWrite() {
		c.String(http.StatusForbidden, "no permission")
		return
	}
	tpv.Deleted = 1
	_, err = comm.Db.Context(ctx).Cols("deleted").Where("id = ?", tpv.Id).Update(tpv)
	if err != nil {
		util.RespInternalErr(c, "db operation", err)
		return
	}
	c.String(http.StatusOK, "ok")
}
