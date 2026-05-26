package util

import (
	"fmt"
	"net/http"
	"reflect"
	"strings"

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
func GinReqParseJson(fn interface{}) gin.HandlerFunc {
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
						c.String(500, fmt.Sprintf("params err[%d]:%+v", i, err))
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
				c.String(500, fmt.Sprintf("router err:%+v", err))
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

func MidAccessAllowFun(c *gin.Context) {
	method := strings.ToUpper(c.Request.Method)
	if method == "OPTIONS" || method == "POST" {
		c.Header("Access-Control-Allow-Origin", c.Request.Header.Get("Origin"))
		c.Header("Access-Control-Allow-Headers", "*,Content-Type")
		c.Header("Access-Control-Allow-Methods", "*")
		c.Header("Access-Control-Allow-Credentials", "true")
	}
	//放行所有OPTIONS方法
	if method == "OPTIONS" {
		c.AbortWithStatus(http.StatusNoContent)
		return
	}
	// 处理请求
	c.Next()
}
