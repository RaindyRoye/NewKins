package service

import (
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gokins/gokins/comm"
	"github.com/gokins/gokins/model"
	"github.com/gokins/gokins/util"
	_ "github.com/mattn/go-sqlite3"
	bolt "go.etcd.io/bbolt"
)

// setupCurrPermTest creates an isolated in-memory SQLite DB and bbolt cache
// for CheckCurrPermission tests. Cleans up via t.Cleanup.
func setupCurrPermTest(t *testing.T) {
	t.Helper()
	eng := setupUserTestDB(t)
	_ = eng // comm.Db is already set by setupUserTestDB

	tmpFile, err := os.CreateTemp("", "currperm_cache_test_*.db")
	if err != nil {
		t.Fatalf("failed to create temp cache file: %v", err)
	}
	_ = tmpFile.Close()

	cache, err := bolt.Open(tmpFile.Name(), 0600, &bolt.Options{Timeout: 1 * time.Second})
	if err != nil {
		_ = os.Remove(tmpFile.Name())
		t.Fatalf("failed to open bbolt: %v", err)
	}

	oldCache := comm.BCache
	comm.BCache = cache
	t.Cleanup(func() {
		_ = cache.Close()
		comm.BCache = oldCache
		_ = os.Remove(tmpFile.Name())
	})
}

// makeGinCtxWithToken creates a gin.Context with an Authorization header
// containing the provided JWT token string.
func makeGinCtxWithToken(tokenStr string) *gin.Context {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	if tokenStr != "" {
		req.Header.Set("Authorization", "TOKEN "+tokenStr)
	}
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	// Build a gin.Context from the request
	ginCtx, _ := gin.CreateTestContext(w)
	ginCtx.Request = req
	return ginCtx
}

// makeCurrPermToken creates a JWT token with the given uid using the test login key.
func makeCurrPermToken(t *testing.T, uid string) string {
	t.Helper()
	loginKey := "test-login-key-currperm"
	token, err := util.CreateToken(
		map[string]any{"uid": uid},
		loginKey,
		time.Hour,
	)
	if err != nil {
		t.Fatalf("create token: %v", err)
	}
	return token
}

// insertCurrPermUser inserts a test user into comm.Db with a unique Aid.
var currPermAidCounter int64 = 100

func insertCurrPermUser(t *testing.T, id, name string) {
	t.Helper()
	currPermAidCounter++
	user := &model.TUser{
		Id:      id,
		Aid:     currPermAidCounter,
		Name:    name,
		Nick:    name,
		Created: time.Now(),
	}
	if _, err := comm.Db.Insert(user); err != nil {
		t.Fatalf("insert user: %v", err)
	}
}

// --- CheckCurrPermission Tests ---

func TestCheckCurrPermission_AdminWithValidToken(t *testing.T) {
	setupCurrPermTest(t)
	loginKey := "test-login-key-currperm"
	comm.Cfg.Server.LoginKey = loginKey

	insertCurrPermUser(t, "admin-uid-cp", "admin")

	token := makeCurrPermToken(t, "admin-uid-cp")
	ginCtx := makeGinCtxWithToken(token)

	if !CheckCurrPermission(ginCtx, PermAdmin) {
		t.Error("admin user with valid token should pass admin permission")
	}
}

func TestCheckCurrPermission_CommonPerm(t *testing.T) {
	setupCurrPermTest(t)
	loginKey := "test-login-key-currperm"
	comm.Cfg.Server.LoginKey = loginKey

	insertCurrPermUser(t, "user-uid-cp1", "alice")

	token := makeCurrPermToken(t, "user-uid-cp1")
	ginCtx := makeGinCtxWithToken(token)

	if !CheckCurrPermission(ginCtx, PermCommon) {
		t.Error("any valid user should pass common permission")
	}
}

func TestCheckCurrPermission_NoToken(t *testing.T) {
	setupCurrPermTest(t)
	comm.Cfg.Server.LoginKey = "test-login-key-currperm"

	ginCtx := makeGinCtxWithToken("")

	if CheckCurrPermission(ginCtx, PermCommon) {
		t.Error("request without token should fail permission check")
	}
}

func TestCheckCurrPermission_InvalidToken(t *testing.T) {
	setupCurrPermTest(t)
	comm.Cfg.Server.LoginKey = "test-login-key-currperm"

	ginCtx := makeGinCtxWithToken("invalid-token-string")

	if CheckCurrPermission(ginCtx, PermCommon) {
		t.Error("request with invalid token should fail permission check")
	}
}

func TestCheckCurrPermission_UserNotFound(t *testing.T) {
	setupCurrPermTest(t)
	loginKey := "test-login-key-currperm"
	comm.Cfg.Server.LoginKey = loginKey

	// Token references uid that doesn't exist in DB
	token := makeCurrPermToken(t, "nonexistent-uid-cp")
	ginCtx := makeGinCtxWithToken(token)

	if CheckCurrPermission(ginCtx, PermCommon) {
		t.Error("token with nonexistent uid should fail permission check")
	}
}

func TestCheckCurrPermission_NonAdminRequestAdmin(t *testing.T) {
	setupCurrPermTest(t)
	loginKey := "test-login-key-currperm"
	comm.Cfg.Server.LoginKey = loginKey

	insertCurrPermUser(t, "regular-uid-cp", "regular")

	token := makeCurrPermToken(t, "regular-uid-cp")
	ginCtx := makeGinCtxWithToken(token)

	if CheckCurrPermission(ginCtx, PermAdmin) {
		t.Error("non-admin user should not pass admin permission")
	}
}

func TestCheckCurrPermission_UnknownPermLevel(t *testing.T) {
	setupCurrPermTest(t)
	loginKey := "test-login-key-currperm"
	comm.Cfg.Server.LoginKey = loginKey

	insertCurrPermUser(t, "user-uid-cp2", "bob")

	token := makeCurrPermToken(t, "user-uid-cp2")
	ginCtx := makeGinCtxWithToken(token)

	if CheckCurrPermission(ginCtx, "superadmin") {
		t.Error("unknown permission level should return false")
	}
}

func TestCheckCurrPermission_TokenWithoutUid(t *testing.T) {
	setupCurrPermTest(t)
	loginKey := "test-login-key-currperm"
	comm.Cfg.Server.LoginKey = loginKey

	// Create a token without uid claim
	token, err := util.CreateToken(
		map[string]any{"role": "user"}, // no uid
		loginKey,
		time.Hour,
	)
	if err != nil {
		t.Fatalf("create token: %v", err)
	}
	ginCtx := makeGinCtxWithToken(token)

	if CheckCurrPermission(ginCtx, PermCommon) {
		t.Error("token without uid should fail permission check")
	}
}

func TestCheckCurrPermission_TokenWithEmptyUid(t *testing.T) {
	setupCurrPermTest(t)
	loginKey := "test-login-key-currperm"
	comm.Cfg.Server.LoginKey = loginKey

	// Create a token with empty uid
	token, err := util.CreateToken(
		map[string]any{"uid": ""},
		loginKey,
		time.Hour,
	)
	if err != nil {
		t.Fatalf("create token: %v", err)
	}
	ginCtx := makeGinCtxWithToken(token)

	if CheckCurrPermission(ginCtx, PermCommon) {
		t.Error("token with empty uid should fail permission check")
	}
}
