package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/gokins/gokins/model"
)

func makeTestUser(id string, active int) *model.TUser {
	return &model.TUser{
		Id:     id,
		Active: active,
	}
}

func testIsAdmin(usr *model.TUser) bool {
	return usr.Id == "admin"
}

func TestMidAuth_Authenticated(t *testing.T) {
	gin.SetMode(gin.TestMode)

	resolver := func(c *gin.Context) (*model.TUser, bool) {
		return makeTestUser("user1", 1), true
	}

	router := gin.New()
	router.Use(MidAuth(resolver, testIsAdmin))
	router.GET("/test", func(c *gin.Context) {
		usr := GetAuthUser(c)
		if usr == nil {
			c.String(http.StatusInternalServerError, "no user")
			return
		}
		c.String(http.StatusOK, "ok")
	})

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}
}

func TestMidAuth_Unauthenticated(t *testing.T) {
	gin.SetMode(gin.TestMode)

	resolver := func(c *gin.Context) (*model.TUser, bool) {
		return nil, false
	}

	router := gin.New()
	router.Use(MidAuth(resolver, testIsAdmin))
	router.GET("/test", func(c *gin.Context) {
		c.String(http.StatusOK, "should not reach")
	})

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("expected status 403, got %d", w.Code)
	}
}

func TestMidAuth_InactiveNonAdmin(t *testing.T) {
	gin.SetMode(gin.TestMode)

	resolver := func(c *gin.Context) (*model.TUser, bool) {
		return makeTestUser("user1", 0), true // inactive, not admin
	}

	router := gin.New()
	router.Use(MidAuth(resolver, testIsAdmin))
	router.GET("/test", func(c *gin.Context) {
		c.String(http.StatusOK, "should not reach")
	})

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("expected status 403, got %d", w.Code)
	}
}

func TestMidAuth_AdminBypass(t *testing.T) {
	gin.SetMode(gin.TestMode)

	resolver := func(c *gin.Context) (*model.TUser, bool) {
		return makeTestUser("admin", 0), true // inactive but admin
	}

	router := gin.New()
	router.Use(MidAuth(resolver, testIsAdmin))
	router.GET("/test", func(c *gin.Context) {
		c.String(http.StatusOK, "ok")
	})

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}
}

func TestGetAuthUser_Missing(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.GET("/test", func(c *gin.Context) {
		usr := GetAuthUser(c)
		if usr != nil {
			c.String(http.StatusOK, "has user")
			return
		}
		c.String(http.StatusOK, "no user")
	})

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Body.String() != "no user" {
		t.Errorf("expected 'no user', got %q", w.Body.String())
	}
}
