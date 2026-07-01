// Package middleware provides common HTTP/Gin middleware for the NewKins
// application. This package extracts reusable middleware from util/ to
// provide a clean, focused API for route configuration.
package middleware

import (
	"fmt"
	"net/http"
	"reflect"
	"runtime/debug"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// GinController defines the interface for registering route controllers.
type GinController interface {
	GetPath() string // Must start with "/"
	Routes(g gin.IRoutes)
}

// RegisterController registers a GinController with the given Gin engine.
// If the controller's path is longer than 1 character, it creates a route group.
func RegisterController(g *gin.Engine, gc GinController) {
	var gp gin.IRoutes
	if g == nil || gc == nil {
		return
	}
	gp = g
	if len(gc.GetPath()) > 1 {
		gp = g.Group(gc.GetPath())
	}
	gc.Routes(gp)
}

// JSONBinder returns a Gin middleware that binds JSON request bodies to handler
// function parameters using reflection. The handler function must accept a
// *gin.Context as its first parameter, followed by optional struct/map parameters
// that will be populated from the JSON body.
func JSONBinder(fn any) gin.HandlerFunc {
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

// InternalError logs the detailed error server-side and returns a generic
// "internal server error" message to the client. An optional context message
// can be provided to give more information in the server logs.
// This prevents leaking internal details (DB schemas, SQL queries, etc.) to clients.
func InternalError(c *gin.Context, msg string, err error) {
	logrus.Errorf("[route] %s: %v", msg, err)
	c.String(http.StatusInternalServerError, "internal server error")
}

// RespondError sends a custom error message to the client while also logging the
// full error details server-side. Use this when the error message is safe to
// show to the client (e.g., validation errors, business logic errors).
func RespondError(c *gin.Context, statusCode int, msg string, err error) {
	logrus.Errorf("[route] %s: %v", msg, err)
	c.String(statusCode, msg)
}

// CORSAllowAll returns a Gin middleware that handles CORS preflight and
// allows all origins, headers, and methods. OPTIONS requests are terminated
// with a 204 No Content response.
func CORSAllowAll() gin.HandlerFunc {
	return func(c *gin.Context) {
		method := strings.ToUpper(c.Request.Method)
		if method == "OPTIONS" || method == "POST" {
			c.Header("Access-Control-Allow-Origin", c.Request.Header.Get("Origin"))
			c.Header("Access-Control-Allow-Headers", "*,Content-Type")
			c.Header("Access-Control-Allow-Methods", "*")
			c.Header("Access-Control-Allow-Credentials", "true")
		}
		// Terminate OPTIONS requests
		if method == "OPTIONS" {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		c.Next()
	}
}
