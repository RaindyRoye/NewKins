package server

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/pprof"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/gokins/gokins/pkg/middleware"
	"github.com/gokins/gokins/util/httpex"

	"github.com/gin-gonic/gin"
	"github.com/gokins/core"
	"github.com/gokins/gokins/comm"
	"github.com/gokins/gokins/route"
	"github.com/gokins/gokins/util"
	"github.com/sirupsen/logrus"
)

// shutdownTimeout is the maximum time to wait for in-flight requests
// to complete during graceful shutdown.
const shutdownTimeout = 10 * time.Second

// uiBufferSize is the buffer size used when streaming static UI files to clients.
// 32 KiB is a good balance: large enough to amortize syscall overhead, small
// enough to keep memory usage modest for concurrent clients.
const uiBufferSize = 32 * 1024

// uiMaxSize is the maximum allowed uncompressed size for a single static
// UI asset. Files larger than this are rejected to prevent decompression
// bombs (gosec G110) from consuming server memory.
const uiMaxSize = 50 << 20 // 50 MiB

// uiCacheMaxAge is the Cache-Control max-age value for immutable static assets.
// One year in seconds (RFC 7234 recommends at most 1 year).
const uiCacheMaxAge = 365 * 24 * 60 * 60

// webDoneCh is closed when the runWeb goroutine has fully exited, allowing
// the main Run loop to wait for graceful web server shutdown without a
// fixed-duration sleep.
var webDoneCh = make(chan struct{})

// apiRateLimiter limits all API requests to 120 per minute per IP.
var apiRateLimiter = middleware.NewRateLimiter(120, time.Minute)

func runWeb() {
	defer close(webDoneCh)
	defer util.RecoverLog("Web")

	comm.WebEgn = gin.Default()
	comm.WebEgn.Use(middleware.MidRequestID())
	comm.WebEgn.Use(middleware.MidSecurityHeaders())
	comm.WebEgn.Use(midUiHandle)

	// Signal that the gin engine is initialized and ready for route registration.
	// This replaces the fragile time.Sleep(10ms) in Run() with proper synchronization.
	close(webReadyCh)

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
		defer util.RecoverLog("Web.ListenAndServe")
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
}

func regApi() {
	// Apply rate limiting to all /api/* routes.
	comm.WebEgn.Use(func(c *gin.Context) {
		if strings.HasPrefix(c.Request.URL.Path, "/api/") || c.Request.URL.Path == "/api" {
			middleware.MidRateLimit(apiRateLimiter)(c)
			return
		}
		c.Next()
	})
	if core.Debug {
		comm.WebEgn.Use(middleware.MidCORS())
	}

	// Enable pprof in debug mode or when explicitly enabled via config
	enablePprof := core.Debug || comm.Cfg.Server.EnablePprof
	if enablePprof {
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
		logrus.Infof("pprof profiling endpoints enabled at /debug/pprof (debug=%v, config=%v)", core.Debug, comm.Cfg.Server.EnablePprof)
	}
	// Health check endpoints
	comm.WebEgn.GET("/healthz", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":     "ok",
			"version":    comm.Version,
			"build_time": comm.BuildTime,
			"git_commit": comm.GitCommit,
		})
	})
	comm.WebEgn.GET("/readyz", func(c *gin.Context) {
		status := gin.H{
			"status": "ready",
			"db":     "connected",
			"cache":  "connected",
		}
		httpStatus := http.StatusOK

		if comm.Db == nil {
			status["db"] = "disconnected"
			status["status"] = "not_ready"
			httpStatus = http.StatusServiceUnavailable
		} else {
			// Verify actual database connectivity with a ping.
			db := comm.Db.DB()
			if db != nil {
				if err := db.Ping(); err != nil {
					status["db"] = fmt.Sprintf("error: %v", err)
					status["status"] = "not_ready"
					httpStatus = http.StatusServiceUnavailable
				}
			}
		}
		if comm.BCache == nil {
			status["cache"] = "disconnected"
			status["status"] = "not_ready"
			httpStatus = http.StatusServiceUnavailable
		}
		c.JSON(httpStatus, status)
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
		httpex.ResMsgUrl(c, "未找到内容,跳转中...", "/")
		return
	}
	// Reject oversized assets before opening to prevent decompression bombs (gosec G110).
	if r.UncompressedSize64 > uiMaxSize {
		httpex.ResMsgUrl(c, "内容过大,拒绝加载", "/")
		return
	}
	rd, err := r.Open()
	if err != nil {
		httpex.ResMsgUrl(c, "内容有误,跳转中...", "/")
		return
	}
	defer func() { _ = rd.Close() }()

	// Wrap in LimitReader as defense-in-depth against decompression bombs.
	// The size check above rejects known-large files, but LimitReader ensures
	// we never read beyond uiMaxSize even if metadata is corrupted.
	limitedRd := io.LimitReader(rd, uiMaxSize+1)

	// Set Content-Length from the uncompressed file size so clients can
	// show download progress and allocate buffers efficiently.
	c.Writer.Header().Set("Content-Length", fmt.Sprintf("%d", r.UncompressedSize64))

	// Determine content type from file extension and set appropriate cache headers.
	ext := filepath.Ext(r.Name)
	switch ext {
	case ".html":
		// HTML must not be cached aggressively so deployments pick up new versions.
		c.Writer.Header().Set("Cache-Control", "no-cache")
		c.Writer.Header().Set("Pragma", "no-cache")
		c.Writer.Header().Set("Expires", "0")
		c.Writer.Header().Set("Content-Type", "text/html")
	case ".css":
		c.Writer.Header().Set("Content-Type", "text/css")
		c.Writer.Header().Set("Cache-Control", fmt.Sprintf("max-age=%d", uiCacheMaxAge))
	case ".js":
		c.Writer.Header().Set("Content-Type", "application/javascript")
		c.Writer.Header().Set("Cache-Control", fmt.Sprintf("max-age=%d", uiCacheMaxAge))
	case ".svg":
		c.Writer.Header().Set("Content-Type", "image/svg+xml")
		c.Writer.Header().Set("Cache-Control", fmt.Sprintf("max-age=%d", uiCacheMaxAge))
	case ".woff2":
		c.Writer.Header().Set("Cache-Control", fmt.Sprintf("max-age=%d", uiCacheMaxAge))
	case ".ttf", ".ttc":
		c.Writer.Header().Set("Content-Type", "application/x-font-ttf")
		c.Writer.Header().Set("Cache-Control", fmt.Sprintf("max-age=%d", uiCacheMaxAge))
	}

	c.Status(http.StatusOK)

	// Use io.Copy with a reusable buffer instead of manual read/write loops.
	// This reduces GC pressure (single allocation per request vs. one per chunk)
	// and respects context cancellation through gin's response writer.
	buf := make([]byte, uiBufferSize)
	_, _ = io.CopyBuffer(c.Writer, limitedRd, buf) //nolint:gosec // G110: limitedRd is bounded by io.LimitReader(uiMaxSize+1) above, and UncompressedSize64 is pre-checked
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
		return nil, fmt.Errorf("getFile: %w", ErrPathEmpty)
	}
	// Prevent path traversal attacks
	cleaned := filepath.Clean(pth)
	if strings.Contains(cleaned, "..") || filepath.IsAbs(cleaned) {
		return nil, fmt.Errorf("getFile: %w: %s", ErrInvalidPath, pth)
	}
	// println("getFile:" + pth)
	r, err := getRdr()
	if err != nil {
		return nil, fmt.Errorf("getFile: %w", err)
	}
	for _, f := range r.File {
		nm := strings.ReplaceAll(f.Name, "\\", "/")
		// println(fmt.Sprintf("find zip file:%s, %s",pth, nm))
		if cleaned == nm {
			return f, nil
		}
	}
	return nil, fmt.Errorf("getFile: %w: %s", ErrFileNotFound, pth)
}
