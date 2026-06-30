package route

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gokins/gokins/comm"
	"github.com/gokins/gokins/model"
	"github.com/gokins/gokins/service"
	_ "github.com/mattn/go-sqlite3"
	hbtp "github.com/mgr9525/HyperByte-Transfer-Protocol"
	"github.com/stretchr/testify/assert"
	"xorm.io/xorm"
)

func setupPipelineVersionTestDB(t *testing.T) {
	t.Helper()
	origDb := comm.Db
	t.Cleanup(func() { comm.Db = origDb })

	db, err := xorm.NewEngine("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("failed to init test DB: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	// Create t_pipeline_version table
	_, err = db.Exec(`CREATE TABLE t_pipeline_version (
		id VARCHAR(64) NOT NULL PRIMARY KEY,
		uid VARCHAR(64),
		number BIGINT,
		events VARCHAR(100),
		sha VARCHAR(255),
		pipeline_name VARCHAR(255),
		pipeline_display_name VARCHAR(255),
		pipeline_id VARCHAR(64),
		version VARCHAR(255),
		content TEXT,
		created DATETIME,
		deleted INT DEFAULT 0,
		pr_number BIGINT,
		repo_clone_url VARCHAR(255)
	)`)
	if err != nil {
		t.Fatalf("failed to create t_pipeline_version table: %v", err)
	}

	// Create t_pipeline table
	_, err = db.Exec(`CREATE TABLE t_pipeline (
		id VARCHAR(64) NOT NULL PRIMARY KEY,
		uid VARCHAR(64),
		name VARCHAR(255),
		display_name VARCHAR(255),
		pipeline_type VARCHAR(255),
		deleted INT DEFAULT 0,
		deleted_time DATETIME,
		create_time DATETIME
	)`)
	if err != nil {
		t.Fatalf("failed to create t_pipeline table: %v", err)
	}

	// Create t_user table
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

	// Create t_user_info table
	_, err = db.Exec(`CREATE TABLE t_user_info (
		id VARCHAR(64) NOT NULL PRIMARY KEY,
		phone VARCHAR(100),
		email VARCHAR(200),
		birthday DATETIME,
		remark TEXT,
		perm_user INT,
		perm_org INT,
		perm_pipe INT
	)`)
	if err != nil {
		t.Fatalf("failed to create t_user_info table: %v", err)
	}

	comm.Db = db
}

func makePipelineVersionTestContext(t *testing.T, body interface{}, loggedInUser *model.TUser) (*gin.Context, *httptest.ResponseRecorder) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	var req *http.Request
	if body != nil {
		bodyBytes, _ := json.Marshal(body)
		req = httptest.NewRequest("POST", "/test", bytes.NewReader(bodyBytes))
	} else {
		req = httptest.NewRequest("POST", "/test", nil)
	}
	req.Header.Set("Content-Type", "application/json")
	c.Request = req

	if loggedInUser != nil {
		c.Set(service.LgUserKey, loggedInUser)
	}
	return c, w
}

func TestPipelineVersionController_delete_EmptyID(t *testing.T) {
	setupPipelineVersionTestDB(t)
	ctrl := PipelineVersionController{}
	adminUser := &model.TUser{Id: "admin", Name: "admin", Active: 1}

	c, w := makePipelineVersionTestContext(t, hbtp.Map{"id": ""}, adminUser)
	ctrl.delete(c, &hbtp.Map{"id": ""})

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "param err")
}

func TestPipelineVersionController_delete_MissingID(t *testing.T) {
	setupPipelineVersionTestDB(t)
	ctrl := PipelineVersionController{}
	adminUser := &model.TUser{Id: "admin", Name: "admin", Active: 1}

	c, w := makePipelineVersionTestContext(t, hbtp.Map{}, adminUser)
	ctrl.delete(c, &hbtp.Map{})

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "param err")
}

func TestPipelineVersionController_delete_NotFound(t *testing.T) {
	setupPipelineVersionTestDB(t)
	ctrl := PipelineVersionController{}
	adminUser := &model.TUser{Id: "admin", Name: "admin", Active: 1}

	c, w := makePipelineVersionTestContext(t, hbtp.Map{"id": "nonexistent"}, adminUser)
	ctrl.delete(c, &hbtp.Map{"id": "nonexistent"})

	assert.Equal(t, http.StatusNotFound, w.Code)
	assert.Contains(t, w.Body.String(), "not found")
}

func TestPipelineVersionController_delete_PipelineNotFound(t *testing.T) {
	setupPipelineVersionTestDB(t)

	// Create a pipeline version without a corresponding pipeline
	pv := &model.TPipelineVersion{
		Id:         "pv-orphan",
		PipelineId: "nonexistent-pipe",
		Version:    "1.0.0",
		Created:    time.Now(),
	}
	_, err := comm.Db.InsertOne(pv)
	if err != nil {
		t.Fatalf("insert pipeline version: %v", err)
	}

	ctrl := PipelineVersionController{}
	adminUser := &model.TUser{Id: "admin", Name: "admin", Active: 1}

	c, w := makePipelineVersionTestContext(t, hbtp.Map{"id": "pv-orphan"}, adminUser)
	ctrl.delete(c, &hbtp.Map{"id": "pv-orphan"})

	assert.Equal(t, http.StatusNotFound, w.Code)
	assert.Contains(t, w.Body.String(), "not found pipe")
}

func TestPipelineVersionController_delete_NoPermission(t *testing.T) {
	setupPipelineVersionTestDB(t)

	// Create a pipeline owned by another user
	pipe := &model.TPipeline{
		Id:         "pipe-other",
		Uid:        "other-user",
		Name:       "other-pipeline",
		CreateTime: time.Now(),
	}
	_, err := comm.Db.InsertOne(pipe)
	if err != nil {
		t.Fatalf("insert pipeline: %v", err)
	}

	// Create a pipeline version for that pipeline
	pv := &model.TPipelineVersion{
		Id:         "pv-other",
		PipelineId: "pipe-other",
		Version:    "1.0.0",
		Created:    time.Now(),
	}
	_, err = comm.Db.InsertOne(pv)
	if err != nil {
		t.Fatalf("insert pipeline version: %v", err)
	}

	ctrl := PipelineVersionController{}
	regularUser := &model.TUser{Id: "regular", Name: "regular", Active: 1}

	c, w := makePipelineVersionTestContext(t, hbtp.Map{"id": "pv-other"}, regularUser)
	ctrl.delete(c, &hbtp.Map{"id": "pv-other"})

	assert.Equal(t, http.StatusForbidden, w.Code)
	assert.Contains(t, w.Body.String(), "no permission")
}

func TestPipelineVersionController_delete_Success(t *testing.T) {
	setupPipelineVersionTestDB(t)

	// Create a pipeline owned by admin
	pipe := &model.TPipeline{
		Id:         "pipe-admin",
		Uid:        "admin",
		Name:       "admin-pipeline",
		CreateTime: time.Now(),
	}
	_, err := comm.Db.InsertOne(pipe)
	if err != nil {
		t.Fatalf("insert pipeline: %v", err)
	}

	// Create a pipeline version for that pipeline
	pv := &model.TPipelineVersion{
		Id:         "pv-admin",
		PipelineId: "pipe-admin",
		Version:    "1.0.0",
		Created:    time.Now(),
	}
	_, err = comm.Db.InsertOne(pv)
	if err != nil {
		t.Fatalf("insert pipeline version: %v", err)
	}

	ctrl := PipelineVersionController{}
	adminUser := &model.TUser{Id: "admin", Name: "admin", Active: 1}

	c, w := makePipelineVersionTestContext(t, hbtp.Map{"id": "pv-admin"}, adminUser)
	ctrl.delete(c, &hbtp.Map{"id": "pv-admin"})

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "ok")

	// Verify the pipeline version is marked as deleted
	updated := &model.TPipelineVersion{}
	ok, err := comm.Db.Where("id = ?", "pv-admin").Get(updated)
	if err != nil {
		t.Fatalf("query updated pipeline version: %v", err)
	}
	if !ok {
		t.Fatal("pipeline version not found after delete")
	}
	if updated.Deleted != 1 {
		t.Errorf("pipeline version deleted = %d, want 1", updated.Deleted)
	}
}

func TestPipelineVersionController_Routes(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	c := &PipelineVersionController{}
	
	group := r.Group("/api/pipelineVersion")
	c.Routes(group)
	
	// Verify the route is registered by making a request
	// We expect it to fail with 403 (auth required) rather than 404
	reqBody := map[string]string{}
	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/api/pipelineVersion/delete", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	
	r.ServeHTTP(w, req)
	
	// Should not be 404 (route not found)
	assert.NotEqual(t, http.StatusNotFound, w.Code)
}
