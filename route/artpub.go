package route

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gokins/core/common"
	"github.com/gokins/core/utils"
	"github.com/gokins/gokins/comm"
	"github.com/gokins/gokins/model"
	"github.com/gokins/gokins/service"
	"github.com/gokins/gokins/util"
	"github.com/sirupsen/logrus"
)

// Sentinel errors for artifact download path validation.
// These can be used with errors.Is() for programmatic error checking.
var (
	// ErrPathTraversal is returned when a path parameter contains directory
	// traversal sequences (e.g., "../") that could escape the base directory.
	ErrPathTraversal = errors.New("path traversal detected")

	// ErrAbsolutePath is returned when a path parameter contains an absolute
	// path, which is not allowed for security reasons.
	ErrAbsolutePath = errors.New("absolute path not allowed")
)

type ArtPublicController struct{}

func (ArtPublicController) GetPath() string {
	return "/api/art/pub"
}
func (c *ArtPublicController) Routes(g gin.IRoutes) {
	g.GET("/down/:id/*pth", c.down)
	g.GET("/downs/:id/:name/*pth", c.downs)
}
func (cs *ArtPublicController) down(c *gin.Context) {
	id := c.Param("id")
	pth := c.Param("pth")
	tms := c.Query("times")
	random := c.Query("random")
	sign := c.Query("sign")
	if tms == "" || random == "" || sign == "" {
		c.String(http.StatusBadRequest, "param err")
		return
	}

	tm, err := time.Parse(time.RFC3339Nano, tms)
	if err != nil {
		c.String(http.StatusBadRequest, "param err:times")
		return
	}
	if time.Since(tm).Hours() > 20 {
		c.String(http.StatusRequestTimeout, "request timeout")
		return
	}

	signs := utils.Md5String(id + tms + random + comm.Cfg.Server.DownToken)
	if sign != signs {
		c.String(http.StatusForbidden, "No Permission")
		return
	}

	artv := &model.TArtifactVersion{}
	ctx := c.Request.Context()
	ok := service.GetIdOrAidCtx(ctx, id, artv)
	if !ok {
		c.String(http.StatusNotFound, "Not Found")
		return
	}
	arty := &model.TArtifactory{}
	ok = service.GetIdOrAidCtx(ctx, artv.RepoId, arty)
	if !ok || arty.Deleted == 1 {
		c.String(http.StatusNotFound, "Not Found repo")
		return
	}
	baseDir := filepath.Join(comm.WorkPath, common.PathArtifacts, artv.Id)
	fls, err := safeJoin(baseDir, pth)
	if err != nil {
		logrus.Warnf("artpub down: path traversal attempt for artifact %s: %v", id, err)
		c.String(http.StatusForbidden, "invalid path")
		return
	}
	cs.downFile(c, fls)
}
func (cs *ArtPublicController) downs(c *gin.Context) {
	id := c.Param("id")
	name := c.Param("name")
	pth := c.Param("pth")
	tms := c.Query("times")
	random := c.Query("random")
	sign := c.Query("sign")
	if tms == "" || random == "" || sign == "" {
		c.String(http.StatusBadRequest, "param err")
		return
	}

	tm, err := time.Parse(time.RFC3339Nano, tms)
	if err != nil {
		c.String(http.StatusBadRequest, "param err:times")
		return
	}
	if time.Since(tm).Hours() > 20 {
		c.String(http.StatusRequestTimeout, "request timeout")
		return
	}

	signs := utils.Md5String(id + name + tms + random + comm.Cfg.Server.DownToken)
	if sign != signs {
		c.String(http.StatusForbidden, "No Permission")
		return
	}

	job := &model.TStep{}
	ok := service.GetIdOrAidCtx(c.Request.Context(), id, job)
	if !ok {
		c.String(http.StatusNotFound, "Not Found")
		return
	}
	bdpth := filepath.Join(comm.WorkPath, common.PathBuild, job.BuildId)
	baseDir := filepath.Join(bdpth, common.PathJobs, job.Id, common.PathArts)
	fls, err := safeJoin(baseDir, name, pth)
	if err != nil {
		logrus.Warnf("artpub downs: path traversal attempt for step %s: %v", id, err)
		c.String(http.StatusForbidden, "invalid path")
		return
	}
	cs.downFile(c, fls)
}

// sanitizePath validates that a user-supplied path segment does not escape
// its intended base directory via directory traversal ("..") or absolute
// paths. It cleans the path and rejects any dangerous components.
//
// This protects against attackers who craft URLs like
// /api/art/pub/down/ID/../../../etc/passwd to read arbitrary server files.
func sanitizePath(pth string) error {
	if filepath.IsAbs(pth) {
		return fmt.Errorf("sanitizePath: %w: %s", ErrAbsolutePath, pth)
	}
	// Check raw path segments before cleaning, as Clean() normalizes away ".."
	for _, seg := range strings.Split(pth, string(filepath.Separator)) {
		if seg == ".." {
			return fmt.Errorf("sanitizePath: %w: %s", ErrPathTraversal, pth)
		}
	}
	// Also check the cleaned result in case of encoded traversals
	cleaned := filepath.Clean(pth)
	for _, seg := range strings.Split(cleaned, string(filepath.Separator)) {
		if seg == ".." {
			return fmt.Errorf("sanitizePath: %w: %s", ErrPathTraversal, pth)
		}
	}
	return nil
}

// safeJoin joins a base directory with one or more user-supplied path segments,
// validating each segment against traversal attacks before joining.
// Returns the joined path or an error if any segment is unsafe.
func safeJoin(base string, parts ...string) (string, error) {
	for _, p := range parts {
		if err := sanitizePath(p); err != nil {
			return "", err
		}
	}
	all := append([]string{base}, parts...)
	return filepath.Join(all...), nil
}

func (ArtPublicController) downFile(c *gin.Context, fls string) {
	stat, err := os.Stat(fls)
	if err != nil {
		c.String(http.StatusNotFound, "Not Found File")
		return
	}
	var nms string
	var contsz int64
	var rdr io.Reader
	if stat.IsDir() {
		nms = stat.Name() + ".zip"
		zipth := filepath.Join(comm.WorkPath, common.PathTmp, utils.NewXid())
		err = utils.Zip(fls, zipth)
		if err != nil {
			util.RespInternalErr(c, "zip artifacts", err)
			return
		}
		defer func() { _ = os.RemoveAll(zipth) }()
		stat, err = os.Stat(zipth)
		if err != nil {
			util.RespInternalErr(c, "stat zip file", err)
			return
		}
		fls = zipth
	} else {
		nms = stat.Name()
	}

	fl, err := os.Open(filepath.Clean(fls))
	if err != nil {
		c.String(http.StatusNotFound, "Not Found File")
		return
	}
	defer func() { _ = fl.Close() }()
	rdr = fl
	contsz = stat.Size()

	c.Header("Connection", "Keep-Alive")
	c.Header("Content-Type", "application/octet-stream")
	c.Header("Cache-Control", "max-age=360000000")
	c.Header("Content-Length", fmt.Sprintf("%d", contsz))
	c.Header("Content-Disposition", fmt.Sprintf(`attachment;filename="%s"`, url.QueryEscape(nms)))
	c.Status(http.StatusOK)
	bts := make([]byte, 10240)
	for c.Request.Context().Err() == nil {
		n, err := rdr.Read(bts)
		if n <= 0 {
			break
		}
		if _, werr := c.Writer.Write(bts[:n]); werr != nil {
			break
		}
		if err != nil {
			break
		}
	}
}
