package service

import (
	"context"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gokins/gokins/comm"
	"github.com/gokins/gokins/model"
	_ "github.com/mattn/go-sqlite3"
	"xorm.io/xorm"
)

func setupPermsTestDB(t *testing.T) {
	t.Helper()
	origDb := comm.Db
	origCtx := comm.Ctx
	t.Cleanup(func() {
		comm.Db = origDb
		comm.Ctx = origCtx
	})

	db, err := xorm.NewEngine("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("failed to init test DB: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	_, err = db.Exec(`CREATE TABLE t_user (
		id VARCHAR(64) NOT NULL PRIMARY KEY,
		aid BIGINT,
		name VARCHAR(100),
		pass VARCHAR(255),
		nick VARCHAR(100),
		avatar VARCHAR(500),
		created DATETIME,
		login_time DATETIME,
		active INT DEFAULT 0
	)`)
	if err != nil {
		t.Fatalf("failed to create t_user table: %v", err)
	}

	comm.Db = db
	comm.Ctx = context.Background()
}

func TestCheckPermissionCtx_NilUser(t *testing.T) {
	setupPermsTestDB(t)
	if CheckPermissionCtx(context.Background(), "nonexistent", PermCommon) {
		t.Error("CheckPermissionCtx should return false for nonexistent user")
	}
}

func TestCheckPermissionCtx_CommonPerm(t *testing.T) {
	setupPermsTestDB(t)
	_, err := comm.Db.Insert(&model.TUser{
		Id:      "user1",
		Name:    "testuser",
		Active:  1,
		Created: time.Now(),
	})
	if err != nil {
		t.Fatalf("insert user: %v", err)
	}

	if !CheckPermissionCtx(context.Background(), "user1", PermCommon) {
		t.Error("CheckPermissionCtx should return true for PermCommon")
	}
}

func TestCheckPermissionCtx_AdminPerm(t *testing.T) {
	setupPermsTestDB(t)
	_, err := comm.Db.Insert(&model.TUser{
		Id:      "admin1",
		Name:    AdminUserName,
		Active:  1,
		Created: time.Now(),
	})
	if err != nil {
		t.Fatalf("insert admin user: %v", err)
	}

	if !CheckPermissionCtx(context.Background(), "admin1", PermAdmin) {
		t.Error("CheckPermissionCtx should return true for admin user with PermAdmin")
	}
}

func TestCheckPermissionCtx_NonAdminWithAdminPerm(t *testing.T) {
	setupPermsTestDB(t)
	_, err := comm.Db.Insert(&model.TUser{
		Id:      "user2",
		Name:    "regularuser",
		Active:  1,
		Created: time.Now(),
	})
	if err != nil {
		t.Fatalf("insert user: %v", err)
	}

	if CheckPermissionCtx(context.Background(), "user2", PermAdmin) {
		t.Error("CheckPermissionCtx should return false for non-admin user with PermAdmin")
	}
}

func TestCheckUPermission_NilUser(t *testing.T) {
	if CheckUPermission(nil, PermCommon) {
		t.Error("CheckUPermission should return false for nil user")
	}
}

func TestCheckUPermission_CommonPerm(t *testing.T) {
	usr := &model.TUser{Id: "user1", Name: "testuser"}
	if !CheckUPermission(usr, PermCommon) {
		t.Error("CheckUPermission should return true for PermCommon")
	}
}

func TestCheckUPermission_AdminPerm(t *testing.T) {
	admin := &model.TUser{Id: "admin1", Name: AdminUserName}
	if !CheckUPermission(admin, PermAdmin) {
		t.Error("CheckUPermission should return true for admin user")
	}

	regular := &model.TUser{Id: "user2", Name: "regularuser"}
	if CheckUPermission(regular, PermAdmin) {
		t.Error("CheckUPermission should return false for non-admin user")
	}
}

func TestCheckCurrPermission_NoUser(t *testing.T) {
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	req := httptest.NewRequest("GET", "/", nil)
	c.Request = req

	if CheckCurrPermission(c, PermCommon) {
		t.Error("CheckCurrPermission should return false when no user in context")
	}
}

// Note: Full CheckCurrPermission integration tests are skipped because
// CurrUserCache requires a valid JWT token in the request and a database
// lookup, which is better suited for end-to-end integration tests.
