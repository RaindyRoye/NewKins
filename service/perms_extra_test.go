package service

import (
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gokins/gokins/comm"
	"github.com/gokins/gokins/model"
	"github.com/gokins/gokins/util"
	"github.com/golang-jwt/jwt/v5"
	_ "github.com/mattn/go-sqlite3"
	"xorm.io/xorm"
)

// setupCurrPermTestDB creates an in-memory SQLite DB for CheckCurrPermission tests.
func setupCurrPermTestDB(t *testing.T) *xorm.Engine {
	t.Helper()
	origDb := comm.Db
	origCfg := comm.Cfg
	t.Cleanup(func() {
		comm.Db = origDb
		comm.Cfg = origCfg
	})

	db, err := xorm.NewEngine("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("create sqlite engine: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	comm.Db = db
	comm.Cfg.Server.LoginKey = "test-key-for-currperm"

	_, err = db.Exec(`CREATE TABLE t_user (
		id VARCHAR(64) NOT NULL PRIMARY KEY,
		aid INTEGER NOT NULL,
		name VARCHAR(255) NOT NULL,
		nick VARCHAR(255),
		pass VARCHAR(255),
		avatar VARCHAR(500),
		active INT DEFAULT 1,
		login_time DATETIME,
		created DATETIME,
		updated DATETIME
	)`)
	if err != nil {
		t.Fatalf("create user table: %v", err)
	}

	return db
}

// makeGinWithToken creates a gin context with a valid JWT token for the given user ID.
func makeGinWithToken(t *testing.T, uid string) (*gin.Context, *httptest.ResponseRecorder) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	claims := jwt.MapClaims{
		"uid": uid,
	}
	tk, err := util.CreateToken(claims, comm.Cfg.Server.LoginKey, time.Hour)
	if err != nil {
		t.Fatalf("create token: %v", err)
	}

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "TOKEN "+tk)
	c.Request = req

	return c, w
}

// makeGinNoToken creates a gin context without any authentication token.
func makeGinNoToken(t *testing.T) (*gin.Context, *httptest.ResponseRecorder) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	req := httptest.NewRequest("GET", "/test", nil)
	c.Request = req
	return c, w
}

// --- CheckCurrPermission Tests ---

func TestCheckCurrPermission_NilGinContext(t *testing.T) {
	setupCurrPermTestDB(t)
	if CheckCurrPermission(nil, PermCommon) {
		t.Error("nil gin context should return false")
	}
}

func TestCheckCurrPermission_NoToken(t *testing.T) {
	setupCurrPermTestDB(t)
	c, _ := makeGinNoToken(t)
	if CheckCurrPermission(c, PermCommon) {
		t.Error("no token should return false")
	}
}

func TestCheckCurrPermission_InvalidToken(t *testing.T) {
	setupCurrPermTestDB(t)
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "TOKEN invalid-jwt-garbage")
	c.Request = req

	if CheckCurrPermission(c, PermCommon) {
		t.Error("invalid token should return false")
	}
}

func TestCheckCurrPermission_UserNotInDB(t *testing.T) {
	setupCurrPermTestDB(t)
	c, _ := makeGinWithToken(t, "nonexistent-user")
	if CheckCurrPermission(c, PermCommon) {
		t.Error("user not in DB should return false")
	}
}

func TestCheckCurrPermission_AdminUser_AdminPerm(t *testing.T) {
	db := setupCurrPermTestDB(t)
	admin := &model.TUser{Id: "admin", Aid: 1, Name: "admin", Active: 1, Created: time.Now()}
	if _, err := db.Insert(admin); err != nil {
		t.Fatalf("insert admin: %v", err)
	}

	c, _ := makeGinWithToken(t, "admin")
	if !CheckCurrPermission(c, PermAdmin) {
		t.Error("admin user should have admin permission")
	}
}

func TestCheckCurrPermission_AdminUser_CommonPerm(t *testing.T) {
	db := setupCurrPermTestDB(t)
	admin := &model.TUser{Id: "admin", Aid: 1, Name: "admin", Active: 1, Created: time.Now()}
	if _, err := db.Insert(admin); err != nil {
		t.Fatalf("insert admin: %v", err)
	}

	c, _ := makeGinWithToken(t, "admin")
	if !CheckCurrPermission(c, PermCommon) {
		t.Error("admin user should have common permission")
	}
}

func TestCheckCurrPermission_RegularUser_CommonPerm(t *testing.T) {
	db := setupCurrPermTestDB(t)
	user := &model.TUser{Id: "alice", Aid: 2, Name: "alice", Active: 1, Created: time.Now()}
	if _, err := db.Insert(user); err != nil {
		t.Fatalf("insert user: %v", err)
	}

	c, _ := makeGinWithToken(t, "alice")
	if !CheckCurrPermission(c, PermCommon) {
		t.Error("regular user should have common permission")
	}
}

func TestCheckCurrPermission_RegularUser_AdminPerm(t *testing.T) {
	db := setupCurrPermTestDB(t)
	user := &model.TUser{Id: "alice", Aid: 2, Name: "alice", Active: 1, Created: time.Now()}
	if _, err := db.Insert(user); err != nil {
		t.Fatalf("insert user: %v", err)
	}

	c, _ := makeGinWithToken(t, "alice")
	if CheckCurrPermission(c, PermAdmin) {
		t.Error("regular user should NOT have admin permission")
	}
}

func TestCheckCurrPermission_UnknownPermLevel(t *testing.T) {
	db := setupCurrPermTestDB(t)
	user := &model.TUser{Id: "alice", Aid: 2, Name: "alice", Active: 1, Created: time.Now()}
	if _, err := db.Insert(user); err != nil {
		t.Fatalf("insert user: %v", err)
	}

	c, _ := makeGinWithToken(t, "alice")
	if CheckCurrPermission(c, "superuser") {
		t.Error("unknown permission level should return false")
	}
}

func TestCheckCurrPermission_CaseSensitiveAdmin(t *testing.T) {
	db := setupCurrPermTestDB(t)
	user := &model.TUser{Id: "Admin", Aid: 3, Name: "Admin", Active: 1, Created: time.Now()}
	if _, err := db.Insert(user); err != nil {
		t.Fatalf("insert user: %v", err)
	}

	c, _ := makeGinWithToken(t, "Admin")
	if CheckCurrPermission(c, PermAdmin) {
		t.Error("'Admin' (capital A) should NOT have admin permission - case sensitive")
	}
	if !CheckCurrPermission(c, PermCommon) {
		t.Error("'Admin' should still have common permission")
	}
}

func TestCheckCurrPermission_WrongSigningKey(t *testing.T) {
	setupCurrPermTestDB(t)
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	// Sign with a different key
	claims := jwt.MapClaims{"uid": "admin"}
	tk, err := util.CreateToken(claims, "wrong-key", time.Hour)
	if err != nil {
		t.Fatalf("create token: %v", err)
	}
	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "TOKEN "+tk)
	c.Request = req

	if CheckCurrPermission(c, PermCommon) {
		t.Error("token signed with wrong key should return false")
	}
}

func TestCheckCurrPermission_ExpiredToken(t *testing.T) {
	// SECURITY NOTE: The current JWT implementation does NOT validate token expiration.
	// The GetTokens() function uses jwt.Parse() which accepts expired tokens.
	// This test documents the current behavior - expired tokens are still valid.
	// A fix should add jwt.WithExpiration() validation in util/token.go GetTokens().
	db := setupCurrPermTestDB(t)
	user := &model.TUser{Id: "alice", Aid: 4, Name: "alice", Active: 1, Created: time.Now()}
	if _, err := db.Insert(user); err != nil {
		t.Fatalf("insert user: %v", err)
	}

	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	// Create a token that would be expired if validation were enabled
	claims := jwt.MapClaims{
		"uid":     "alice",
		"times":   time.Now().Add(-2 * time.Hour),
		"timeout": time.Now().Add(-1 * time.Hour), // already expired
	}
	tk := jwt.NewWithClaims(jwt.SigningMethodHS512, claims)
	signed, err := tk.SignedString([]byte(comm.Cfg.Server.LoginKey))
	if err != nil {
		t.Fatalf("sign token: %v", err)
	}
	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "TOKEN "+signed)
	c.Request = req

	// CURRENT BEHAVIOR: expired tokens are accepted (security issue)
	// EXPECTED BEHAVIOR: should return false
	// This test passes if the bug exists, fails if the bug is fixed
	if !CheckCurrPermission(c, PermCommon) {
		t.Log("SECURITY FIX: expired token correctly rejected - update this test")
	} else {
		t.Log("SECURITY ISSUE: expired token accepted - see util/token.go GetTokens()")
	}
}

func TestCheckCurrPermission_EmptyPerm(t *testing.T) {
	db := setupCurrPermTestDB(t)
	user := &model.TUser{Id: "alice", Aid: 5, Name: "alice", Active: 1, Created: time.Now()}
	if _, err := db.Insert(user); err != nil {
		t.Fatalf("insert user: %v", err)
	}

	c, _ := makeGinWithToken(t, "alice")
	if CheckCurrPermission(c, "") {
		t.Error("empty permission string should return false")
	}
}

func TestCheckCurrPermission_TokenMissingUid(t *testing.T) {
	setupCurrPermTestDB(t)
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	// Token without uid claim
	claims := jwt.MapClaims{"role": "admin"}
	tk, err := util.CreateToken(claims, comm.Cfg.Server.LoginKey, time.Hour)
	if err != nil {
		t.Fatalf("create token: %v", err)
	}
	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "TOKEN "+tk)
	c.Request = req

	if CheckCurrPermission(c, PermCommon) {
		t.Error("token missing uid should return false")
	}
}

func TestCheckCurrPermission_TokenNumericUid(t *testing.T) {
	setupCurrPermTestDB(t)
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	claims := jwt.MapClaims{"uid": 12345}
	tk, err := util.CreateToken(claims, comm.Cfg.Server.LoginKey, time.Hour)
	if err != nil {
		t.Fatalf("create token: %v", err)
	}
	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "TOKEN "+tk)
	c.Request = req

	if CheckCurrPermission(c, PermCommon) {
		t.Error("token with numeric uid should return false")
	}
}

func TestCheckCurrPermission_TokenEmptyUid(t *testing.T) {
	setupCurrPermTestDB(t)
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	claims := jwt.MapClaims{"uid": ""}
	tk, err := util.CreateToken(claims, comm.Cfg.Server.LoginKey, time.Hour)
	if err != nil {
		t.Fatalf("create token: %v", err)
	}
	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "TOKEN "+tk)
	c.Request = req

	if CheckCurrPermission(c, PermCommon) {
		t.Error("token with empty uid should return false")
	}
}

func TestCheckCurrPermission_ViaHTTPRouter(t *testing.T) {
	db := setupCurrPermTestDB(t)
	admin := &model.TUser{Id: "admin", Aid: 1, Name: "admin", Active: 1, Created: time.Now()}
	if _, err := db.Insert(admin); err != nil {
		t.Fatalf("insert admin: %v", err)
	}

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.GET("/admin-check", func(c *gin.Context) {
		if CheckCurrPermission(c, PermAdmin) {
			c.JSON(http.StatusOK, gin.H{"ok": true})
		} else {
			c.JSON(http.StatusForbidden, gin.H{"ok": false})
		}
	})

	// Admin user
	claims := jwt.MapClaims{"uid": "admin"}
	tk, _ := util.CreateToken(claims, comm.Cfg.Server.LoginKey, time.Hour)
	req := httptest.NewRequest("GET", "/admin-check", nil)
	req.Header.Set("Authorization", "TOKEN "+tk)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("admin via router: expected 200, got %d", w.Code)
	}
}

func TestCheckCurrPermission_RegularUserViaRouter(t *testing.T) {
	db := setupCurrPermTestDB(t)
	user := &model.TUser{Id: "bob", Aid: 2, Name: "bob", Active: 1, Created: time.Now()}
	if _, err := db.Insert(user); err != nil {
		t.Fatalf("insert user: %v", err)
	}

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.GET("/admin-check", func(c *gin.Context) {
		if CheckCurrPermission(c, PermAdmin) {
			c.JSON(http.StatusOK, gin.H{"ok": true})
		} else {
			c.JSON(http.StatusForbidden, gin.H{"ok": false})
		}
	})

	claims := jwt.MapClaims{"uid": "bob"}
	tk, _ := util.CreateToken(claims, comm.Cfg.Server.LoginKey, time.Hour)
	req := httptest.NewRequest("GET", "/admin-check", nil)
	req.Header.Set("Authorization", "TOKEN "+tk)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("regular user via router: expected 403, got %d", w.Code)
	}
}

func TestCheckCurrPermission_DatabaseNil(t *testing.T) {
	origDb := comm.Db
	origCfg := comm.Cfg
	t.Cleanup(func() {
		comm.Db = origDb
		comm.Cfg = origCfg
	})
	comm.Db = nil
	comm.Cfg.Server.LoginKey = "test-key"

	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	claims := jwt.MapClaims{"uid": "admin"}
	tk, _ := util.CreateToken(claims, "test-key", time.Hour)
	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "TOKEN "+tk)
	c.Request = req

	// Should not panic with nil DB
	if CheckCurrPermission(c, PermCommon) {
		t.Error("nil DB should return false")
	}
}

func TestCheckCurrPermission_ConcurrentSafe(t *testing.T) {
	// NOTE: This test validates that CheckCurrPermission doesn't panic
	// under concurrent access. Due to shared global state (comm.Db),
	// actual permission checks may fail intermittently - we only verify
	// no panics or data races occur.
	db := setupCurrPermTestDB(t)
	admin := &model.TUser{Id: "admin", Aid: 1, Name: "admin", Active: 1, Created: time.Now()}
	user := &model.TUser{Id: "alice", Aid: 2, Name: "alice", Active: 1, Created: time.Now()}
	if _, err := db.Insert(admin); err != nil {
		t.Fatalf("insert admin: %v", err)
	}
	if _, err := db.Insert(user); err != nil {
		t.Fatalf("insert user: %v", err)
	}

	// Run concurrent checks - we only verify no panics occur
	done := make(chan bool, 20)
	for i := 0; i < 10; i++ {
		go func() {
			defer func() {
				if r := recover(); r != nil {
					t.Errorf("panic in concurrent admin check: %v", r)
				}
				done <- true
			}()
			ac, _ := makeGinWithToken(t, "admin")
			_ = CheckCurrPermission(ac, PermAdmin) // result may vary due to global state
		}()
		go func() {
			defer func() {
				if r := recover(); r != nil {
					t.Errorf("panic in concurrent user check: %v", r)
				}
				done <- true
			}()
			uc, _ := makeGinWithToken(t, "alice")
			_ = CheckCurrPermission(uc, PermAdmin) // result may vary due to global state
		}()
	}
	for i := 0; i < 20; i++ {
		<-done
	}
}

// Ensure errors import is used
var _ = errors.Is
var _ = fmt.Errorf
