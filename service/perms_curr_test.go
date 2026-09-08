package service

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gokins/gokins/comm"
	"github.com/gokins/gokins/model"
	"github.com/gokins/gokins/util"
	bolt "go.etcd.io/bbolt"
	_ "github.com/mattn/go-sqlite3"
	"xorm.io/xorm"
)

// setupPermTestDB creates an isolated in-memory SQLite DB and bbolt cache for permission tests.
func setupPermTestDB(t *testing.T) (*xorm.Engine, *bolt.DB) {
	t.Helper()

	// Setup SQLite
	eng, err := xorm.NewEngine("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("failed to create test database: %v", err)
	}
	oldDb := comm.Db
	comm.Db = eng

	// Setup bbolt cache in temp directory
	tmpDir := t.TempDir()
	boltPath := filepath.Join(tmpDir, "test-cache.db")
	cache, err := bolt.Open(boltPath, 0600, &bolt.Options{Timeout: 1 * time.Second})
	if err != nil {
		eng.Close()
		t.Fatalf("failed to create test cache: %v", err)
	}
	oldCache := comm.BCache
	comm.BCache = cache

	t.Cleanup(func() {
		comm.Db = oldDb
		comm.BCache = oldCache
		_ = eng.Close()
		_ = cache.Close()
		_ = os.Remove(boltPath)
	})

	if err := eng.Sync2(&model.TUser{}); err != nil {
		t.Fatalf("failed to sync user schema: %v", err)
	}

	return eng, cache
}

// --- CheckCurrPermission tests ---

func TestCheckCurrPermission_AdminUser(t *testing.T) {
	eng, _ := setupPermTestDB(t)

	// Create admin user
	adminUser := &model.TUser{
		Id:     "admin",
		Aid:    1,
		Name:   "admin",
		Active: 1,
	}
	if _, err := eng.Insert(adminUser); err != nil {
		t.Fatalf("failed to insert admin user: %v", err)
	}

	// Setup config with login key
	oldCfg := comm.Cfg
	comm.Cfg.Server.LoginKey = "test-secret-key"
	defer func() { comm.Cfg = oldCfg }()

	// Create gin context with admin token
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/test", nil)

	// Create JWT token for admin
	token, err := util.CreateToken(
		map[string]interface{}{"uid": "admin"},
		comm.Cfg.Server.LoginKey,
		time.Hour,
	)
	if err != nil {
		t.Fatalf("failed to create token: %v", err)
	}
	c.Request.Header.Set("Authorization", "TOKEN "+token)

	// Test admin permission
	if !CheckCurrPermission(c, PermAdmin) {
		t.Error("CheckCurrPermission should return true for admin user with admin permission")
	}

	// Test common permission
	if !CheckCurrPermission(c, PermCommon) {
		t.Error("CheckCurrPermission should return true for admin user with common permission")
	}
}

func TestCheckCurrPermission_RegularUser(t *testing.T) {
	eng, _ := setupPermTestDB(t)

	// Create regular user
	regularUser := &model.TUser{
		Id:     "user1",
		Aid:    1,
		Name:   "alice",
		Active: 1,
	}
	if _, err := eng.Insert(regularUser); err != nil {
		t.Fatalf("failed to insert regular user: %v", err)
	}

	// Setup config with login key
	oldCfg := comm.Cfg
	comm.Cfg.Server.LoginKey = "test-secret-key"
	defer func() { comm.Cfg = oldCfg }()

	// Create gin context with regular user token
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/test", nil)

	// Create JWT token for regular user
	token, err := util.CreateToken(
		map[string]interface{}{"uid": "user1"},
		comm.Cfg.Server.LoginKey,
		time.Hour,
	)
	if err != nil {
		t.Fatalf("failed to create token: %v", err)
	}
	c.Request.Header.Set("Authorization", "TOKEN "+token)

	// Test common permission (should pass)
	if !CheckCurrPermission(c, PermCommon) {
		t.Error("CheckCurrPermission should return true for regular user with common permission")
	}

	// Test admin permission (should fail)
	if CheckCurrPermission(c, PermAdmin) {
		t.Error("CheckCurrPermission should return false for regular user with admin permission")
	}
}

func TestCheckCurrPermission_NoToken(t *testing.T) {
	_, _ = setupPermTestDB(t)

	// Setup config with login key
	oldCfg := comm.Cfg
	comm.Cfg.Server.LoginKey = "test-secret-key"
	defer func() { comm.Cfg = oldCfg }()

	// Create gin context without token
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/test", nil)

	// Test permissions (should all fail)
	if CheckCurrPermission(c, PermCommon) {
		t.Error("CheckCurrPermission should return false when no token is present")
	}
	if CheckCurrPermission(c, PermAdmin) {
		t.Error("CheckCurrPermission should return false when no token is present")
	}
}

func TestCheckCurrPermission_InvalidToken(t *testing.T) {
	_, _ = setupPermTestDB(t)

	// Setup config with login key
	oldCfg := comm.Cfg
	comm.Cfg.Server.LoginKey = "test-secret-key"
	defer func() { comm.Cfg = oldCfg }()

	// Create gin context with invalid token
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/test", nil)
	c.Request.Header.Set("Authorization", "TOKEN invalid-token-here")

	// Test permissions (should all fail)
	if CheckCurrPermission(c, PermCommon) {
		t.Error("CheckCurrPermission should return false with invalid token")
	}
	if CheckCurrPermission(c, PermAdmin) {
		t.Error("CheckCurrPermission should return false with invalid token")
	}
}

func TestCheckCurrPermission_UserNotFound(t *testing.T) {
	_, _ = setupPermTestDB(t)

	// Setup config with login key
	oldCfg := comm.Cfg
	comm.Cfg.Server.LoginKey = "test-secret-key"
	defer func() { comm.Cfg = oldCfg }()

	// Create gin context with token for non-existent user
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/test", nil)

	// Create JWT token for non-existent user
	token, err := util.CreateToken(
		map[string]interface{}{"uid": "nonexistent-user"},
		comm.Cfg.Server.LoginKey,
		time.Hour,
	)
	if err != nil {
		t.Fatalf("failed to create token: %v", err)
	}
	c.Request.Header.Set("Authorization", "TOKEN "+token)

	// Test permissions (should all fail)
	if CheckCurrPermission(c, PermCommon) {
		t.Error("CheckCurrPermission should return false when user not found in DB")
	}
	if CheckCurrPermission(c, PermAdmin) {
		t.Error("CheckCurrPermission should return false when user not found in DB")
	}
}

func TestCheckCurrPermission_UnknownPermLevel(t *testing.T) {
	eng, _ := setupPermTestDB(t)

	// Create regular user
	regularUser := &model.TUser{
		Id:     "user1",
		Aid:    1,
		Name:   "alice",
		Active: 1,
	}
	if _, err := eng.Insert(regularUser); err != nil {
		t.Fatalf("failed to insert regular user: %v", err)
	}

	// Setup config with login key
	oldCfg := comm.Cfg
	comm.Cfg.Server.LoginKey = "test-secret-key"
	defer func() { comm.Cfg = oldCfg }()

	// Create gin context with regular user token
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/test", nil)

	// Create JWT token for regular user
	token, err := util.CreateToken(
		map[string]interface{}{"uid": "user1"},
		comm.Cfg.Server.LoginKey,
		time.Hour,
	)
	if err != nil {
		t.Fatalf("failed to create token: %v", err)
	}
	c.Request.Header.Set("Authorization", "TOKEN "+token)

	// Test unknown permission level (should fail)
	if CheckCurrPermission(c, "superadmin") {
		t.Error("CheckCurrPermission should return false for unknown permission level")
	}
}

func TestCheckCurrPermission_TokenWithCookie(t *testing.T) {
	eng, _ := setupPermTestDB(t)

	// Create admin user
	adminUser := &model.TUser{
		Id:     "admin",
		Aid:    1,
		Name:   "admin",
		Active: 1,
	}
	if _, err := eng.Insert(adminUser); err != nil {
		t.Fatalf("failed to insert admin user: %v", err)
	}

	// Setup config with login key
	oldCfg := comm.Cfg
	comm.Cfg.Server.LoginKey = "test-secret-key"
	defer func() { comm.Cfg = oldCfg }()

	// Create gin context with token in cookie
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/test", nil)

	// Create JWT token for admin
	token, err := util.CreateToken(
		map[string]interface{}{"uid": "admin"},
		comm.Cfg.Server.LoginKey,
		time.Hour,
	)
	if err != nil {
		t.Fatalf("failed to create token: %v", err)
	}

	// Set token in cookie instead of header
	c.Request.AddCookie(&http.Cookie{
		Name:  "gokinstk",
		Value: token,
	})

	// Test admin permission
	if !CheckCurrPermission(c, PermAdmin) {
		t.Error("CheckCurrPermission should return true for admin user with token in cookie")
	}
}

func TestCheckCurrPermission_TokenInQuery(t *testing.T) {
	eng, _ := setupPermTestDB(t)

	// Create regular user
	regularUser := &model.TUser{
		Id:     "user1",
		Aid:    1,
		Name:   "alice",
		Active: 1,
	}
	if _, err := eng.Insert(regularUser); err != nil {
		t.Fatalf("failed to insert regular user: %v", err)
	}

	// Setup config with login key
	oldCfg := comm.Cfg
	comm.Cfg.Server.LoginKey = "test-secret-key"
	defer func() { comm.Cfg = oldCfg }()

	// Create gin context with token in query parameter
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	// Create JWT token for regular user
	token, err := util.CreateToken(
		map[string]interface{}{"uid": "user1"},
		comm.Cfg.Server.LoginKey,
		time.Hour,
	)
	if err != nil {
		t.Fatalf("failed to create token: %v", err)
	}

	// Set token in query parameter
	c.Request = httptest.NewRequest("GET", "/test?authToken="+token, nil)

	// Test common permission
	if !CheckCurrPermission(c, PermCommon) {
		t.Error("CheckCurrPermission should return true for regular user with token in query")
	}
}
