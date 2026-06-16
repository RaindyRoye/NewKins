package server

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/base64"
	"errors"
	"net/http"
	"net/http/pprof"
	"path/filepath"
	"runtime/debug"
	"strings"
	"sync"
	"time"

	"github.com/gokins/gokins/util/httpex"

	"github.com/gin-gonic/gin"
	"github.com/gokins/core"
	"github.com/gokins/gokins/comm"
	"github.com/gokins/gokins/route"
	"github.com/gokins/gokins/util"
	hbtp "github.com/mgr9525/HyperByte-Transfer-Protocol"
	"github.com/sirupsen/logrus"
)

// shutdownTimeout is the maximum time to wait for in-flight requests
// to complete during graceful shutdown.
const shutdownTimeout = 10 * time.Second

// apiRateLimiter limits all API requests to 120 per minute per IP.
var apiRateLimiter = util.NewRateLimiter(120, time.Minute)

func runWeb() {
	defer func() {
		if err := recover(); err != nil {
			logrus.Errorf("Web recover:%v", err)
			logrus.Errorf("Web stack:%s", string(debug.Stack()))
		}
	}()
	comm.WebEgn = gin.Default()
	comm.WebEgn.Use(util.MidRequestID())
	comm.WebEgn.Use(util.MidSecurityHeaders())
	comm.WebEgn.Use(midUiHandle)

	srv := &http.Server{
		Addr:              comm.WebHost,
		Handler:           comm.WebEgn,
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      60 * time.Second,
		IdleTimeout:       120 * time.Second,
	}

	// Start server in a goroutine so we can handle graceful shutdown
	errCh := make(chan error, 1)
	go func() {
		logrus.Infof("Web server listening on %s", comm.WebHost)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
		close(errCh)
	}()

	// Wait for context cancellation or server error
	select {
	case <-comm.Ctx.Done():
		logrus.Info("Shutting down web server gracefully...")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
		defer cancel()
		if err := srv.Shutdown(shutdownCtx); err != nil {
			logrus.Errorf("Web server forced shutdown: %v", err)
		}
	case err := <-errCh:
		if err != nil {
			logrus.Errorf("Web err:%v", err)
		}
	}

	comm.Cancel()
	time.Sleep(time.Millisecond * 100)
}

func regApi() {
	// Apply rate limiting to all /api/* routes.
	comm.WebEgn.Use(func(c *gin.Context) {
		if strings.HasPrefix(c.Request.URL.Path, "/api/") || c.Request.URL.Path == "/api" {
			util.MidRateLimit(apiRateLimiter)(c)
			return
		}
		c.Next()
	})
	if core.Debug {
		comm.WebEgn.Use(util.MidAccessAllowFun)
		// pprof profiling endpoints (debug mode only)
		pprofGroup := comm.WebEgn.Group("/debug/pprof")
		{
			pprofGroup.GET("/", gin.WrapF(pprof.Index))
			pprofGroup.GET("/cmdline", gin.WrapF(pprof.Cmdline))
			pprofGroup.GET("/profile", gin.WrapF(pprof.Profile))
			pprofGroup.GET("/symbol", gin.WrapF(pprof.Symbol))
			pprofGroup.GET("/trace", gin.WrapF(pprof.Trace))
			pprofGroup.GET("/allocs", gin.WrapH(pprof.Handler("allocs")))
			pprofGroup.GET("/block", gin.WrapH(pprof.Handler("block")))
			pprofGroup.GET("/goroutine", gin.WrapH(pprof.Handler("goroutine")))
			pprofGroup.GET("/heap", gin.WrapH(pprof.Handler("heap")))
			pprofGroup.GET("/mutex", gin.WrapH(pprof.Handler("mutex")))
			pprofGroup.GET("/threadcreate", gin.WrapH(pprof.Handler("threadcreate")))
		}
		logrus.Info("pprof profiling endpoints enabled at /debug/pprof")
	}
	// Health check endpoints
	comm.WebEgn.GET("/healthz", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok", "version": comm.Version})
	})
	comm.WebEgn.GET("/readyz", func(c *gin.Context) {
		if comm.Db != nil && comm.BCache != nil {
			c.JSON(200, gin.H{"status": "ready", "db": "connected", "cache": "connected"})
		} else {
			c.JSON(503, gin.H{"status": "not_ready", "db": comm.Db != nil, "cache": comm.BCache != nil})
		}
	})

	util.GinRegController(comm.WebEgn, &route.ApiController{})
	util.GinRegController(comm.WebEgn, &route.ArtifactController{})
	util.GinRegController(comm.WebEgn, &route.ArtPublicController{})
	util.GinRegController(comm.WebEgn, &route.LoginController{})
	util.GinRegController(comm.WebEgn, &route.UserController{})
	util.GinRegController(comm.WebEgn, &route.OrgController{})
	util.GinRegController(comm.WebEgn, &route.PipelineController{})
	util.GinRegController(comm.WebEgn, &route.PipelineVersionController{})
	util.GinRegController(comm.WebEgn, &route.RuntimeController{})
	util.GinRegController(comm.WebEgn, &route.YmlController{})
	util.GinRegController(comm.WebEgn, &route.TriggerController{})
	util.GinRegController(comm.WebEgn, &route.HookController{})
}
func midUiHandle(c *gin.Context) {
	c.Next()
	if c.Writer.Status() != http.StatusNotFound || c.Writer.Size() > 0 {
		return
	}
	pth := c.Request.URL.Path
	if !comm.Installed && !strings.HasPrefix(pth, "/gokinsui/") && pth != "/install" {
		httpex.ResMsgUrl(c, "未安装,跳转中...", "/install")
		return
	}
	r, err := getFile(pth[1:])
	if err != nil {
		r, err = getFile("index.html")
	}
	if err != nil {
		// c.String(404, "rdr err:"+err.Error())
		httpex.ResMsgUrl(c, "未找到内容,跳转中...", "/")
		return
	}
	rd, err := r.Open()
	if err != nil {
		// c.String(500, "open err:"+err.Error())
		httpex.ResMsgUrl(c, "内容有误,跳转中...", "/")
		return
	}
	defer func() { _ = rd.Close() }()
	c.Writer.Header().Set("Cache-Control", "max-age=360000000")

	ext := filepath.Ext(r.Name)
	switch ext {
	case ".html":
		c.Writer.Header().Set("Cache-Control", "no-cache")
		c.Writer.Header().Set("Pragma", "no-cache")
		c.Writer.Header().Set("Expires", "0")
		c.Writer.Header().Set("Content-Type", "text/html")
	case ".css":
		c.Writer.Header().Set("Content-Type", "text/css")
	case ".js":
		c.Writer.Header().Set("Content-Type", "application/javascript")
	case ".svg":
		c.Writer.Header().Set("Content-Type", "image/svg+xml")
	case ".woff2":
		// c.Writer.Header().Set("Content-Type", "image/svg+xml")
	case ".ttf", ".ttc":
		c.Writer.Header().Set("Content-Type", "application/x-font-ttf")
	}
	c.Status(200)
	bts := make([]byte, 1024)
	for !hbtp.EndContext(c) {
		n, err := rd.Read(bts)
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

var (
	rder     *zip.Reader
	rderOnce sync.Once
	rderErr  error
)

func getRdr() (*zip.Reader, error) {
	rderOnce.Do(func() {
		bts, err := base64.StdEncoding.DecodeString(comm.StaticPkg)
		if err != nil {
			rderErr = err
			return
		}
		buf := bytes.NewReader(bts)
		r, err := zip.NewReader(buf, buf.Size())
		if err != nil {
			rderErr = err
			return
		}
		rder = r
	})
	if rderErr != nil {
		return nil, rderErr
	}
	return rder, nil
}
func getFile(pth string) (*zip.File, error) {
	if pth == "" {
		return nil, errors.New("getFile: path parameter is empty")
	}
	// Prevent path traversal attacks
	cleaned := filepath.Clean(pth)
	if strings.Contains(cleaned, "..") || filepath.IsAbs(cleaned) {
		return nil, errors.New("getFile: invalid path")
	}
	// println("getFile:" + pth)
	r, err := getRdr()
	if err != nil {
		return nil, err
	}
	for _, f := range r.File {
		nm := strings.ReplaceAll(f.Name, "\\", "/")
		// println(fmt.Sprintf("find zip file:%s, %s",pth, nm))
		if cleaned == nm {
			return f, nil
		}
	}
	return nil, errors.New("file not found")
}
