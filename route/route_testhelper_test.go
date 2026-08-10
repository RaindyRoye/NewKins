package route

import (
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gokins/gokins/comm"
	"github.com/gokins/gokins/model"
	"github.com/gokins/gokins/service"
	_ "github.com/mattn/go-sqlite3"
	"xorm.io/xorm"
)

// setupRouteTestDB creates an in-memory SQLite database with all tables needed
// for route-level tests (org, artifact, user, pipeline, etc.). It restores
// comm.Db on cleanup so tests are isolated.
func setupRouteTestDB(t *testing.T) {
	t.Helper()
	origDb := comm.Db
	origWorkPath := comm.WorkPath
	t.Cleanup(func() {
		comm.Db = origDb
		comm.WorkPath = origWorkPath
	})

	db, err := xorm.NewEngine("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	tables := []string{
		`CREATE TABLE t_user (
			id VARCHAR(64) NOT NULL PRIMARY KEY,
			aid BIGINT,
			name VARCHAR(100),
			pass VARCHAR(255),
			nick VARCHAR(100),
			avatar VARCHAR(500),
			created DATETIME,
			login_time DATETIME,
			active INT DEFAULT 0
		)`,
		`CREATE TABLE t_user_info (
			id VARCHAR(64) NOT NULL PRIMARY KEY,
			phone VARCHAR(100),
			email VARCHAR(200),
			birthday DATETIME,
			remark TEXT,
			perm_user INT,
			perm_org INT,
			perm_pipe INT
		)`,
		`CREATE TABLE t_org (
			id VARCHAR(64) NOT NULL PRIMARY KEY,
			aid BIGINT,
			uid VARCHAR(64),
			name VARCHAR(200),
			"desc" TEXT,
			public INT DEFAULT 0,
			created DATETIME,
			updated DATETIME,
			deleted INT DEFAULT 0,
			deleted_time DATETIME
		)`,
		`CREATE TABLE t_user_org (
			aid INTEGER PRIMARY KEY AUTOINCREMENT,
			uid VARCHAR(64),
			org_id VARCHAR(64),
			created DATETIME,
			perm_adm INT DEFAULT 0,
			perm_rw INT DEFAULT 0,
			perm_exec INT DEFAULT 0,
			perm_down INT DEFAULT 0
		)`,
		`CREATE TABLE t_org_pipe (
			aid INTEGER PRIMARY KEY AUTOINCREMENT,
			org_id VARCHAR(64),
			pipe_id VARCHAR(64),
			created DATETIME,
			public INT DEFAULT 0
		)`,
		`CREATE TABLE t_org_var (
			aid INTEGER PRIMARY KEY AUTOINCREMENT,
			uid VARCHAR(64),
			org_id VARCHAR(64),
			name VARCHAR(255),
			value TEXT,
			remarks VARCHAR(255),
			public INT DEFAULT 0
		)`,
		`CREATE TABLE t_artifactory (
			id VARCHAR(64) NOT NULL PRIMARY KEY,
			aid BIGINT,
			uid VARCHAR(64),
			org_id VARCHAR(64),
			identifier VARCHAR(50),
			name VARCHAR(200),
			disabled INT DEFAULT 0,
			source VARCHAR(50),
			"desc" VARCHAR(500),
			logo VARCHAR(255),
			created DATETIME,
			updated DATETIME,
			deleted INT DEFAULT 0,
			deleted_time DATETIME
		)`,
		`CREATE TABLE t_artifact_package (
			id VARCHAR(64) NOT NULL PRIMARY KEY,
			aid BIGINT,
			repo_id VARCHAR(64),
			name VARCHAR(100),
			display_name VARCHAR(255),
			"desc" VARCHAR(500),
			created DATETIME,
			updated DATETIME,
			deleted INT DEFAULT 0,
			deleted_time DATETIME
		)`,
		`CREATE TABLE t_artifact_version (
			id VARCHAR(64) NOT NULL PRIMARY KEY,
			aid BIGINT,
			repo_id VARCHAR(64),
			package_id VARCHAR(64),
			name VARCHAR(100),
			version VARCHAR(100),
			sha VARCHAR(100),
			"desc" VARCHAR(500),
			preview INT DEFAULT 0,
			created DATETIME,
			updated DATETIME
		)`,
		`CREATE TABLE t_pipeline (
			id VARCHAR(64) NOT NULL PRIMARY KEY,
			aid BIGINT,
			uid VARCHAR(64),
			name VARCHAR(200),
			display_name VARCHAR(200),
			pipeline_type VARCHAR(50),
			created DATETIME,
			updated DATETIME,
			deleted INT DEFAULT 0,
			deleted_time DATETIME
		)`,
		`CREATE TABLE t_pipeline_conf (
			aid INTEGER PRIMARY KEY AUTOINCREMENT,
			pipeline_id VARCHAR(64),
			yml_content TEXT,
			url VARCHAR(500),
			username VARCHAR(100),
			access_token VARCHAR(500)
		)`,
	}

	for _, ddl := range tables {
		if _, err := db.Exec(ddl); err != nil {
			t.Fatalf("create table: %v\nSQL: %s", err, ddl)
		}
	}

	comm.Db = db
	comm.WorkPath = t.TempDir()
}

// makeRouteGinCtx creates a gin test context with an authenticated user set in
// the middleware key. This bypasses the actual JWT/middleware flow for unit
// testing of controller handlers.
func makeRouteGinCtx(t *testing.T, user *model.TUser) (*gin.Context, *httptest.ResponseRecorder) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	req := httptest.NewRequest("POST", "/test", nil)
	req.Header.Set("Content-Type", "application/json")
	c.Request = req
	if user != nil {
		c.Set(service.LgUserKey, user)
	}
	return c, w
}

// seedUser inserts a user and optional user_info record into the test DB.
//
//nolint:unparam // id parameter kept for test flexibility
func seedUser(t *testing.T, id, name, nick string, active int) *model.TUser {
	t.Helper()
	u := &model.TUser{
		Id:        id,
		Name:      name,
		Nick:      nick,
		Pass:      "testhash",
		Active:    active,
		Created:   time.Now(),
		LoginTime: time.Now(),
	}
	_, err := comm.Db.InsertOne(u)
	if err != nil {
		t.Fatalf("seed user: %v", err)
	}
	return u
}

// seedOrg inserts an org and returns it.
func seedOrg(t *testing.T, id, uid, name string, public int) *model.TOrg {
	t.Helper()
	org := &model.TOrg{
		Id:      id,
		Uid:     uid,
		Name:    name,
		Public:  public,
		Created: time.Now(),
		Updated: time.Now(),
	}
	_, err := comm.Db.InsertOne(org)
	if err != nil {
		t.Fatalf("seed org: %v", err)
	}
	return org
}

// seedUserOrg inserts a user-org membership record.
//
//nolint:unparam // uid parameter kept for test flexibility
func seedUserOrg(t *testing.T, uid, orgId string, adm, rw, exec_, down int) {
	t.Helper()
	uo := &model.TUserOrg{
		Uid:      uid,
		OrgId:    orgId,
		Created:  time.Now(),
		PermAdm:  adm,
		PermRw:   rw,
		PermExec: exec_,
		PermDown: down,
	}
	_, err := comm.Db.InsertOne(uo)
	if err != nil {
		t.Fatalf("seed user org: %v", err)
	}
}
