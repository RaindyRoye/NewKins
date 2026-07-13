package util

import (
	"fmt"
	"net/http"
	"reflect"
	"runtime/debug"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

type GinController interface {
	GetPath() string // 必须"/"开头
	Routes(g gin.IRoutes)
}

func GinRegController(g *gin.Engine, gc GinController) {
	var gp gin.IRoutes
	if g == nil || gc == nil {
		return
	}
	gp = g
	if len(gc.GetPath()) > 1 {
		gp = g.Group(gc.GetPath())
		/*if gc.GetMid()!=nil{
			gp.Use(gc.GetMid())
		}*/
	}
	gc.Routes(gp)
}
func GinReqParseJson(fn any) gin.HandlerFunc {
	fnv := reflect.ValueOf(fn)
	if fnv.Kind() != reflect.Func {
		return nil
	}
	fnt := fnv.Type()
	return func(c *gin.Context) {
		nmIn := fnt.NumIn()
		inls := make([]reflect.Value, nmIn)
		inls[0] = reflect.ValueOf(c)
		for i := 1; i < nmIn; i++ {
			argt := fnt.In(i)
			argtr := argt
			if argt.Kind() == reflect.Pointer {
				argtr = argt.Elem()
			}
			inls[i] = reflect.Zero(argt)
			if strings.Contains(c.ContentType(), "application/json") {
				if argtr.Kind() == reflect.Struct || argtr.Kind() == reflect.Map {
					argv := reflect.New(argtr)
					if err := c.BindJSON(argv.Interface()); err != nil {
						logrus.Warnf("params bind error at arg %d: %v", i, err)
						c.String(http.StatusBadRequest, fmt.Sprintf("invalid request body for parameter %d", i))
						return
					}
					if argt.Kind() == reflect.Pointer {
						inls[i] = argv
					} else {
						inls[i] = argv.Elem()
					}
				}
			}
		}
		defer func() {
			if err := recover(); err != nil {
				logrus.Errorf("router panic:%+v", err)
				logrus.Errorf("router stack:%s", string(debug.Stack()))
				c.String(http.StatusInternalServerError, "internal server error")
			}
		}()
		fnv.Call(inls)
	}
}

// RespInternalErr logs the detailed error server-side and returns a generic
// "internal server error" message to the client. An optional context message
// can be provided to give more information in the server logs.
// This prevents leaking internal details (DB schemas, SQL queries, etc.) to clients.
func RespInternalErr(c *gin.Context, msg string, err error) {
	logrus.Errorf("[route] %s: %v", msg, err)
	c.String(http.StatusInternalServerError, "internal server error")
}

// RespErr sends a custom error message to the client while also logging the
// full error details server-side. Use this when the error message is safe to
// show to the client (e.g., validation errors, business logic errors).
func RespErr(c *gin.Context, statusCode int, msg string, err error) {
	logrus.Errorf("[route] %s: %v", msg, err)
	c.String(statusCode, msg)
}

// RecoverResult is intended for use in deferred recover() blocks inside
// functions that return a named error. It converts the recovered value to an
// error, logs the stack trace at the given context label, and assigns the
// result back into the caller's named error return.
//
// Typical usage inside a hook parser:
//
//	func Parse(...) (wb hook.WebHook, err error) {
//	    defer util.RecoverResult(&err, "github.Parse")
//	    ...
//	}
func RecoverResult(errp *error, label string) {
	r := recover()
	if r == nil {
		return
	}
	logrus.Warnf("%s panic: %+v", label, r)
	logrus.Warnf("%s stack:\n%s", label, string(debug.Stack()))
	if e, ok := r.(error); ok {
		*errp = fmt.Errorf("%s: panic: %w", label, e)
	} else {
		*errp = fmt.Errorf("%s: panic: %v", label, r)
	}
}

// RecoverLog is intended for use in deferred recover() blocks inside
// goroutines or methods that do not return a named error. It logs the
// recovered value and stack trace at the given context label, but does
// not attempt to propagate the error to the caller.
//
// Typical usage inside a goroutine or long-running loop:
//
//	go func() {
//	    defer util.RecoverLog("BuildEngine.run")
//	    ...
//	}()
func RecoverLog(label string) {
	r := recover()
	if r == nil {
		return
	}
	logrus.Warnf("%s panic: %+v", label, r)
	logrus.Warnf("%s stack:\n%s", label, string(debug.Stack()))
}

func MidAccessAllowFun(c *gin.Context) {
	method := strings.ToUpper(c.Request.Method)
	if method == "OPTIONS" || method == "POST" {
		c.Header("Access-Control-Allow-Origin", c.Request.Header.Get("Origin"))
		c.Header("Access-Control-Allow-Headers", "*,Content-Type")
		c.Header("Access-Control-Allow-Methods", "*")
		c.Header("Access-Control-Allow-Credentials", "true")
	}
	// 放行所有OPTIONS方法
	if method == "OPTIONS" {
		c.AbortWithStatus(http.StatusNoContent)
		return
	}
	// 处理请求
	c.Next()
}

// MidRequestLog returns a Gin middleware that logs structured request information
// including method, path, status code, latency, client IP, and request ID.
// The request ID is extracted from the context (set by MidRequestID middleware).
// This enables production observability without verbose debug logging.
func MidRequestLog() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		query := c.Request.URL.RawQuery

		// Process request
		c.Next()

		// Calculate latency
		latency := time.Since(start)

		// Extract request ID from context (set by MidRequestID)
		requestID := ""
		if v, ok := c.Get(RequestIDKey); ok {
			if s, ok := v.(string); ok {
				requestID = s
			}
		}

		// Get client IP
		clientIP := c.ClientIP()

		// Get response status
		status := c.Writer.Status()

		// Get method
		method := c.Request.Method

		// Log structured request info
		// Skip logging for health check endpoints to reduce noise
		if path == "/healthz" || path == "/readyz" {
			return
		}

		fields := logrus.Fields{
			"status":     status,
			"latency":    latency.String(),
			"client_ip":  clientIP,
			"method":     method,
			"path":       path,
			"request_id": requestID,
		}

		if query != "" {
			fields["query"] = query
		}

		// Use different log levels based on status code
		if status >= 500 {
			logrus.WithFields(fields).Error("request completed with server error")
		} else if status >= 400 {
			logrus.WithFields(fields).Warn("request completed with client error")
		} else {
			logrus.WithFields(fields).Info("request completed")
		}
	}
}
