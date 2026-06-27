package route

import (
	"errors"
	"fmt"
	"io"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/gokins/core/runtime"
	"github.com/gokins/core/utils"
	"github.com/gokins/gokins/bean"
	"github.com/gokins/gokins/comm"
	"github.com/gokins/gokins/engine"
	"github.com/gokins/gokins/util"
	"gopkg.in/yaml.v3"
)

// ErrBuildEmpty is returned when a build request has a nil body.
var ErrBuildEmpty = errors.New("build is empty")

// ErrStagesEmpty is returned when a build has no stages defined.
var ErrStagesEmpty = errors.New("stages is empty")

type ApiController struct{}

func (ApiController) GetPath() string {
	return "/api"
}
func (c *ApiController) Routes(g gin.IRoutes) {
	g.Any("/", c.hello)
	g.Any("/version", c.version)
	g.POST("/builds", util.GinReqParseJson(c.test))
}
func (ApiController) hello(c *gin.Context) {
	c.String(200, "hello world")
}
func (ApiController) version(c *gin.Context) {
	c.JSON(200, gin.H{
		"version":    comm.Version,
		"build_time": comm.BuildTime,
		"git_commit": comm.GitCommit,
	})
}
func (ApiController) test(c *gin.Context) {
	all, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"err": fmt.Errorf("read request body: %w", err),
		})
		return
	}
	y := &bean.Pipeline{}
	err = yaml.Unmarshal(all, y)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"err": fmt.Errorf("parse pipeline yaml: %w", err),
		})
		return
	}
	marshal, err := yaml.Marshal(y)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"err": fmt.Errorf("marshal pipeline yaml: %w", err),
		})
		return
	}
	b := &runtime.Build{}
	err = yaml.Unmarshal(marshal, b)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"err": fmt.Errorf("convert pipeline to build: %w", err),
		})
		return
	}
	err = prebuild(b)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"err": fmt.Errorf("prebuild: %w", err),
		})
		return
	}
	engine.Mgr.BuildEgn().Put(b)
	c.JSON(http.StatusOK, gin.H{
		"msg": b,
	})
}

func prebuild(b *runtime.Build) error {
	if b == nil {
		return ErrBuildEmpty
	}
	if len(b.Stages) == 0 {
		return ErrStagesEmpty
	}
	pipelineId := utils.NewXid()
	buildId := utils.NewXid()
	b.Id = buildId
	b.PipelineId = pipelineId
	b.Repo = &runtime.Repository{
		Name:     "",
		Token:    "",
		Sha:      "",
		CloneURL: "https://gitee.com/SuperHeroJim/gokins-test.git",
	}
	for _, stage := range b.Stages {
		stage.Id = utils.NewXid()
		stage.BuildId = buildId
		for _, step := range stage.Steps {
			step.Id = utils.NewXid()
			step.StageId = stage.Id
			step.BuildId = buildId
		}
	}
	return nil
}
