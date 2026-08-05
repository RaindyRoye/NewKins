package route

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gokins/core/utils"
	"github.com/gokins/gokins/comm"
	_ "github.com/mattn/go-sqlite3"
	"xorm.io/xorm"
)

func setupArtPubTestRouter(t *testing.T, db *xorm.Engine) *gin.Engine {
	t.Helper()
	gin.SetMode(gin.TestMode)
	r := gin.New()

	origDb := comm.Db
	comm.Db = db
	t.Cleanup(func() { comm.Db = origDb })

	c := &ArtPublicController{}
	group := r.Group(c.GetPath())
	c.Routes(group)
	return r
}

func createArtPubTestDb(t *testing.T) *xorm.Engine {
	t.Helper()
	db, err := xorm.NewEngine("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("create sqlite engine: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	// Create artifact_version table
	_, err = db.Exec(`CREATE TABLE t_artifact_version (
		id VARCHAR(64) NOT NULL,
		aid BIGINT NOT NULL,
		repo_id VARCHAR(64),
		package_id VARCHAR(64),
		name VARCHAR(100),
		version VARCHAR(100),
		sha VARCHAR(100),
		desc VARCHAR(500),
		preview INT(1),
		created DATETIME,
		updated DATETIME,
		PRIMARY KEY (id, aid)
	)`)
	if err != nil {
		t.Fatalf("create artifact_version table: %v", err)
	}

	// Create artifactory table
	_, err = db.Exec(`CREATE TABLE t_artifactory (
		id VARCHAR(64) NOT NULL,
		aid BIGINT NOT NULL,
		uid VARCHAR(64),
		org_id VARCHAR(64),
		identifier VARCHAR(50),
		name VARCHAR(200),
		disabled INT(1) DEFAULT 0,
		source VARCHAR(50),
		desc VARCHAR(500),
		logo VARCHAR(255),
		created DATETIME,
		updated DATETIME,
		deleted INT(1) DEFAULT 0,
		deleted_time DATETIME,
		PRIMARY KEY (id, aid)
	)`)
	if err != nil {
		t.Fatalf("create artifactory table: %v", err)
	}

	// Create step table
	_, err = db.Exec(`CREATE TABLE t_step (
		id VARCHAR(64) NOT NULL,
		build_id VARCHAR(64),
		stage_id VARCHAR(100),
		display_name VARCHAR(255),
		pipeline_version_id VARCHAR(64),
		step VARCHAR(255),
		status VARCHAR(100),
		event VARCHAR(100),
		exit_code INT(11),
		error VARCHAR(500),
		name VARCHAR(100),
		started DATETIME,
		finished DATETIME,
		created DATETIME,
		updated DATETIME,
		version VARCHAR(255),
		errignore INT(11),
		commands TEXT,
		waits JSON,
		sort INT(11),
		PRIMARY KEY (id)
	)`)
	if err != nil {
		t.Fatalf("create step table: %v", err)
	}

	return db
}

func TestArtPublicController_GetPath_RouteDef(t *testing.T) {
	c := &ArtPublicController{}
	if got := c.GetPath(); got != "/api/art/pub" {
		t.Errorf("GetPath() = %q, want %q", got, "/api/art/pub")
	}
	// Routes should register both /down and /downs patterns
	gin.SetMode(gin.TestMode)
	r := gin.New()
	c.Routes(r)
	routes := r.Routes()
	if len(routes) < 2 {
		t.Errorf("expected at least 2 routes registered, got %d", len(routes))
	}
}

func TestDown_MissingParams(t *testing.T) {
	db := createArtPubTestDb(t)
	r := setupArtPubTestRouter(t, db)

	tests := []struct {
		name   string
		path   string
		params string
	}{
		{"missing times", "/api/art/pub/down/art1/test.txt?random=r1&sign=s1", "times"},
		{"missing random", "/api/art/pub/down/art1/test.txt?times=2025-01-01T00:00:00Z&sign=s1", "random"},
		{"missing sign", "/api/art/pub/down/art1/test.txt?times=2025-01-01T00:00:00Z&random=r1", "sign"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tt.path, nil)
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)

			if w.Code != http.StatusBadRequest {
				t.Errorf("expected 400 for missing %s, got %d", tt.params, w.Code)
			}
		})
	}
}

func TestDown_InvalidTimeFormat(t *testing.T) {
	db := createArtPubTestDb(t)
	r := setupArtPubTestRouter(t, db)

	req := httptest.NewRequest(http.MethodGet,
		"/api/art/pub/down/art1/test.txt?times=invalid&random=r1&sign=s1", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for invalid time, got %d", w.Code)
	}
}

func TestDown_TimeoutRequest(t *testing.T) {
	db := createArtPubTestDb(t)
	r := setupArtPubTestRouter(t, db)

	// Time more than 20 hours ago - use UTC to avoid '+' characters
	oldTime := time.Now().UTC().Add(-25 * time.Hour).Format(time.RFC3339Nano)
	req := httptest.NewRequest(http.MethodGet,
		"/api/art/pub/down/art1/test.txt?times="+oldTime+"&random=r1&sign=s1", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusRequestTimeout {
		t.Errorf("expected 408 for timeout, got %d", w.Code)
	}
}

func TestDown_InvalidSignature(t *testing.T) {
	db := createArtPubTestDb(t)
	r := setupArtPubTestRouter(t, db)

	// Set up test config
	origToken := comm.Cfg.Server.DownToken
	comm.Cfg.Server.DownToken = "test-secret"
	t.Cleanup(func() { comm.Cfg.Server.DownToken = origToken })

	// Use UTC to avoid '+' characters that get decoded as spaces by URL parser
	now := time.Now().UTC().Format(time.RFC3339Nano)
	req := httptest.NewRequest(http.MethodGet,
		"/api/art/pub/down/art1/test.txt?times="+now+"&random=r1&sign=wrong-sign", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("expected 403 for invalid signature, got %d", w.Code)
	}
}

func TestDown_ValidSignature_NotFound(t *testing.T) {
	db := createArtPubTestDb(t)
	r := setupArtPubTestRouter(t, db)

	origToken := comm.Cfg.Server.DownToken
	comm.Cfg.Server.DownToken = "test-secret"
	t.Cleanup(func() { comm.Cfg.Server.DownToken = origToken })

	id := "nonexistent-art"
	// Use UTC time to avoid '+' characters that get decoded as spaces by URL parser
	now := time.Now().UTC().Format(time.RFC3339Nano)
	random := "random123"
	sign := utils.Md5String(id + now + random + comm.Cfg.Server.DownToken)

	req := httptest.NewRequest(http.MethodGet,
		"/api/art/pub/down/"+id+"/test.txt?times="+now+"&random="+random+"&sign="+sign, nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404 for nonexistent artifact, got %d", w.Code)
	}
}

func TestDowns_MissingParams(t *testing.T) {
	db := createArtPubTestDb(t)
	r := setupArtPubTestRouter(t, db)

	tests := []struct {
		name   string
		path   string
		params string
	}{
		{"missing times", "/api/art/pub/downs/step1/pkg/file.txt?random=r1&sign=s1", "times"},
		{"missing random", "/api/art/pub/downs/step1/pkg/file.txt?times=2025-01-01T00:00:00Z&sign=s1", "random"},
		{"missing sign", "/api/art/pub/downs/step1/pkg/file.txt?times=2025-01-01T00:00:00Z&random=r1", "sign"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tt.path, nil)
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)

			if w.Code != http.StatusBadRequest {
				t.Errorf("expected 400 for missing %s, got %d", tt.params, w.Code)
			}
		})
	}
}

func TestDowns_InvalidTimeFormat(t *testing.T) {
	db := createArtPubTestDb(t)
	r := setupArtPubTestRouter(t, db)

	req := httptest.NewRequest(http.MethodGet,
		"/api/art/pub/downs/step1/pkg/file.txt?times=invalid&random=r1&sign=s1", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for invalid time, got %d", w.Code)
	}
}

func TestDowns_TimeoutRequest(t *testing.T) {
	db := createArtPubTestDb(t)
	r := setupArtPubTestRouter(t, db)

	// Use UTC to avoid '+' characters that get decoded as spaces by URL parser
	oldTime := time.Now().UTC().Add(-25 * time.Hour).Format(time.RFC3339Nano)
	req := httptest.NewRequest(http.MethodGet,
		"/api/art/pub/downs/step1/pkg/file.txt?times="+oldTime+"&random=r1&sign=s1", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusRequestTimeout {
		t.Errorf("expected 408 for timeout, got %d", w.Code)
	}
}

func TestDowns_InvalidSignature(t *testing.T) {
	db := createArtPubTestDb(t)
	r := setupArtPubTestRouter(t, db)

	origToken := comm.Cfg.Server.DownToken
	comm.Cfg.Server.DownToken = "test-secret"
	t.Cleanup(func() { comm.Cfg.Server.DownToken = origToken })

	// Use UTC to avoid '+' characters that get decoded as spaces by URL parser
	now := time.Now().UTC().Format(time.RFC3339Nano)
	req := httptest.NewRequest(http.MethodGet,
		"/api/art/pub/downs/step1/pkg/file.txt?times="+now+"&random=r1&sign=wrong-sign", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("expected 403 for invalid signature, got %d", w.Code)
	}
}

func TestDowns_ValidSignature_NotFound(t *testing.T) {
	db := createArtPubTestDb(t)
	r := setupArtPubTestRouter(t, db)

	origToken := comm.Cfg.Server.DownToken
	comm.Cfg.Server.DownToken = "test-secret"
	t.Cleanup(func() { comm.Cfg.Server.DownToken = origToken })

	id := "nonexistent-step"
	name := "package"
	// Use UTC time to avoid '+' characters that get decoded as spaces by URL parser
	now := time.Now().UTC().Format(time.RFC3339Nano)
	random := "random123"
	sign := utils.Md5String(id + name + now + random + comm.Cfg.Server.DownToken)

	req := httptest.NewRequest(http.MethodGet,
		"/api/art/pub/downs/"+id+"/"+name+"/file.txt?times="+now+"&random="+random+"&sign="+sign, nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404 for nonexistent step, got %d", w.Code)
	}
}

func TestDowns_WithDatabase(t *testing.T) {
	db := createArtPubTestDb(t)
	r := setupArtPubTestRouter(t, db)

	origToken := comm.Cfg.Server.DownToken
	comm.Cfg.Server.DownToken = "test-secret"
	t.Cleanup(func() { comm.Cfg.Server.DownToken = origToken })

	// Insert a test step
	_, err := db.Exec(`INSERT INTO t_step (id, build_id) VALUES ('step-1', 'build-1')`)
	if err != nil {
		t.Fatalf("insert step: %v", err)
	}

	id := "step-1"
	name := "artifact"
	// Use UTC time to avoid '+' characters that get decoded as spaces by URL parser
	now := time.Now().UTC().Format(time.RFC3339Nano)
	random := "random123"
	sign := utils.Md5String(id + name + now + random + comm.Cfg.Server.DownToken)

	req := httptest.NewRequest(http.MethodGet,
		"/api/art/pub/downs/"+id+"/"+name+"/file.txt?times="+now+"&random="+random+"&sign="+sign, nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	// Should get 404 because the file doesn't exist on disk
	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404 for missing file, got %d", w.Code)
	}
}
