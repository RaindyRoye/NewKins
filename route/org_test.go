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
	"github.com/gokins/gokins/util"
	"github.com/golang-jwt/jwt/v5"
	_ "github.com/mattn/go-sqlite3"
	"xorm.io/xorm"
)

// ---------- helpers ----------

func setupOrgTestDB(t *testing.T) *xorm.Engine {
	t.Helper()
	db, err := xorm.NewEngine("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("create sqlite engine: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	// Tables used by org endpoints.
	tables := []string{
		`CREATE TABLE t_org (
			id VARCHAR(64) NOT NULL,
			aid INTEGER PRIMARY KEY AUTOINCREMENT,
			uid VARCHAR(64),
			name VARCHAR(200),
			"desc" TEXT,
			public INT DEFAULT 0,
			deleted INT DEFAULT 0,
			deleted_time DATETIME,
			created DATETIME,
			updated DATETIME
		)`,
		`CREATE TABLE t_user (
			id VARCHAR(64) NOT NULL,
			aid INTEGER PRIMARY KEY AUTOINCREMENT,
			name VARCHAR(64),
			nick VARCHAR(64),
			avatar VARCHAR(255),
			pass VARCHAR(64),
			active INT DEFAULT 1,
			login_time DATETIME,
			created DATETIME
		)`,
		`CREATE TABLE t_user_info (
			id VARCHAR(64) NOT NULL,
			phone VARCHAR(32),
			email VARCHAR(128),
			birthday DATETIME,
			remark VARCHAR(255),
			perm_user INT DEFAULT 0,
			perm_org INT DEFAULT 0,
			perm_pipe INT DEFAULT 0
		)`,
		`CREATE TABLE t_user_org (
			aid INTEGER PRIMARY KEY AUTOINCREMENT,
			uid VARCHAR(64),
			org_id VARCHAR(64),
			perm_adm INT DEFAULT 0,
			perm_rw INT DEFAULT 0,
			perm_exec INT DEFAULT 0,
			perm_down INT DEFAULT 0,
			created DATETIME
		)`,
		`CREATE TABLE t_org_pipe (
			aid INTEGER PRIMARY KEY AUTOINCREMENT,
			org_id VARCHAR(64),
			pipe_id VARCHAR(64),
			public INT DEFAULT 0,
			created DATETIME
		)`,
		`CREATE TABLE t_pipeline (
			id VARCHAR(64) NOT NULL PRIMARY KEY,
			uid VARCHAR(64),
			name VARCHAR(100),
			display_name VARCHAR(100),
			pipeline_type VARCHAR(32),
			deleted INT DEFAULT 0
		)`,
		`CREATE TABLE t_org_var (
			aid INTEGER PRIMARY KEY AUTOINCREMENT,
			uid VARCHAR(64),
			org_id VARCHAR(64),
			name VARCHAR(64),
			value VARCHAR(1024),
			remarks VARCHAR(255),
			public INT DEFAULT 0
		)`,
	}
	for _, sql := range tables {
		if _, err := db.Exec(sql); err != nil {
			t.Fatalf("exec %q: %v", sql[:40], err)
		}
	}
	return db
}

func seedOrgUser(t *testing.T, db *xorm.Engine) *model.TUser {
	t.Helper()
	u := &model.TUser{
		Id:        "user-owner",
		Name:      "owner",
		Nick:      "Owner",
		Pass:      "x",
		Active:    1,
		Created:   time.Now(),
		LoginTime: time.Now(),
	}
	if _, err := db.InsertOne(u); err != nil {
		t.Fatalf("insert user: %v", err)
	}
	return u
}

func seedOrg(t *testing.T, db *xorm.Engine, uid, id, name string, public bool) *model.TOrg {
	t.Helper()
	o := &model.TOrg{
		Id:      id,
		Uid:     uid,
		Name:    name,
		Created: time.Now(),
		Updated: time.Now(),
	}
	if public {
		o.Public = 1
	}
	if _, err := db.InsertOne(o); err != nil {
		t.Fatalf("insert org: %v", err)
	}
	return o
}

func seedAdminUser(t *testing.T, db *xorm.Engine) *model.TUser {
	t.Helper()
	// Admin is identified by Id="admin" (service.IsAdmin checks usr.Id == "admin")
	u := &model.TUser{
		Id:        "admin",
		Name:      "admin",
		Nick:      "Admin",
		Pass:      "x",
		Active:    1,
		Created:   time.Now(),
		LoginTime: time.Now(),
	}
	if _, err := db.InsertOne(u); err != nil {
		t.Fatalf("insert admin: %v", err)
	}
	return u
}

// setupOrgRouter builds a gin router with only the org routes.
// It swaps comm.Db and creates a valid JWT token for the supplied lgusr.
func setupOrgRouter(t *testing.T, db *xorm.Engine, lgusr *model.TUser) *gin.Engine {
	t.Helper()
	gin.SetMode(gin.TestMode)
	r := gin.New()

	origDb := comm.Db
	comm.Db = db
	t.Cleanup(func() { comm.Db = origDb })

	// Set up JWT token for authentication
	comm.Cfg.Server.LoginKey = "test-secret-key-for-unit-tests"
	token, err := util.CreateToken(jwt.MapClaims{
		"uid": lgusr.Id,
	}, comm.Cfg.Server.LoginKey, time.Hour*24)
	if err != nil {
		t.Fatalf("create token: %v", err)
	}

	// Middleware to inject token and user into context
	r.Use(func(c *gin.Context) {
		req := c.Request
		if req == nil {
			req = httptest.NewRequest("GET", "/", nil)
		}
		req.Header.Set("Authorization", "TOKEN "+token)
		c.Request = req
		c.Set(service.LgUserKey, lgusr)
		c.Next()
	})
	oc := &OrgController{}
	og := r.Group("/api/org")
	oc.Routes(og)
	return r
}

func orgPostJSON(t *testing.T, r http.Handler, path string, body any) *httptest.ResponseRecorder {
	t.Helper()
	b, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, path, bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

// ---------- tests ----------

func TestOrgController_New_MissingName(t *testing.T) {
	db := setupOrgTestDB(t)
	u := seedOrgUser(t, db)
	r := setupOrgRouter(t, db, u)

	w := orgPostJSON(t, r, "/api/org/new", map[string]any{
		"name": "", "desc": "d", "public": false,
	})
	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d; body = %s", w.Code, http.StatusBadRequest, w.Body.String())
	}
}

func TestOrgController_New_AdminCreatesPublic(t *testing.T) {
	db := setupOrgTestDB(t)
	admin := seedAdminUser(t, db)
	r := setupOrgRouter(t, db, admin)

	w := orgPostJSON(t, r, "/api/org/new", map[string]any{
		"name": "acme", "desc": "desc", "public": true,
	})
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body = %s", w.Code, w.Body.String())
	}

	var res struct {
		Id  string `json:"id"`
		Aid int64  `json:"aid"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &res); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if res.Id == "" {
		t.Error("returned id should not be empty")
	}

	// Verify persisted
	var got model.TOrg
	ok, err := db.Where("name = ?", "acme").Get(&got)
	if err != nil {
		t.Fatalf("query: %v", err)
	}
	if !ok {
		t.Fatal("org not persisted")
	}
	if got.Public != 1 {
		t.Errorf("public = %d, want 1", got.Public)
	}
	if got.Uid != admin.Id {
		t.Errorf("uid = %q, want %q", got.Uid, admin.Id)
	}
}

func TestOrgController_New_NonAdmin_NoPerm(t *testing.T) {
	db := setupOrgTestDB(t)
	u := seedOrgUser(t, db) // non-admin, no perm_org grant
	r := setupOrgRouter(t, db, u)

	w := orgPostJSON(t, r, "/api/org/new", map[string]any{
		"name": "team", "desc": "d", "public": false,
	})
	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("status = %d, want %d; body = %s", w.Code, http.StatusMethodNotAllowed, w.Body.String())
	}
}

func TestOrgController_List_Empty(t *testing.T) {
	db := setupOrgTestDB(t)
	admin := seedAdminUser(t, db)
	r := setupOrgRouter(t, db, admin)

	w := orgPostJSON(t, r, "/api/org/list", map[string]any{"q": "", "page": 1})
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body = %s", w.Code, w.Body.String())
	}
}

func TestOrgController_List_NonAdmin_SeesOnlyPublicAndOwned(t *testing.T) {
	db := setupOrgTestDB(t)
	owner := seedOrgUser(t, db)
	other := &model.TUser{Id: "other-user", Name: "other", Nick: "Other", Pass: "x", Active: 1, Created: time.Now(), LoginTime: time.Now()}
	if _, err := db.InsertOne(other); err != nil {
		t.Fatal(err)
	}

	// Seed 3 orgs: owned, public-other, private-other
	seedOrg(t, db, owner.Id, "org-owned", "owned", false)
	seedOrg(t, db, other.Id, "org-pub", "pub", true)
	seedOrg(t, db, other.Id, "org-priv", "priv", false)

	r := setupOrgRouter(t, db, owner)
	w := orgPostJSON(t, r, "/api/org/list", map[string]any{"q": "", "page": 1})
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d; body = %s", w.Code, w.Body.String())
	}

	var page struct {
		Content []*model.TOrg `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &page); err != nil {
		t.Fatalf("decode: %v; body = %s", err, w.Body.String())
	}
	ids := map[string]bool{}
	for _, o := range page.Content {
		ids[o.Id] = true
	}
	if !ids["org-owned"] {
		t.Error("expected owned org in list")
	}
	if !ids["org-pub"] {
		t.Error("expected public org in list")
	}
	if ids["org-priv"] {
		t.Error("private org of another user must not appear")
	}
}

func TestOrgController_Info_NotFound(t *testing.T) {
	db := setupOrgTestDB(t)
	u := seedOrgUser(t, db)
	r := setupOrgRouter(t, db, u)

	w := orgPostJSON(t, r, "/api/org/info", map[string]any{"id": "nope"})
	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want 404; body = %s", w.Code, w.Body.String())
	}
}

func TestOrgController_Info_EmptyId(t *testing.T) {
	db := setupOrgTestDB(t)
	u := seedOrgUser(t, db)
	r := setupOrgRouter(t, db, u)

	w := orgPostJSON(t, r, "/api/org/info", map[string]any{"id": ""})
	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400; body = %s", w.Code, w.Body.String())
	}
}

func TestOrgController_Info_Deleted(t *testing.T) {
	db := setupOrgTestDB(t)
	u := seedOrgUser(t, db)
	o := seedOrg(t, db, u.Id, "org-x", "X", true)
	if _, err := db.Where("id = ?", o.Id).Cols("deleted").Update(&model.TOrg{Deleted: 1}); err != nil {
		t.Fatal(err)
	}
	r := setupOrgRouter(t, db, u)

	w := orgPostJSON(t, r, "/api/org/info", map[string]any{"id": o.Id})
	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want 404 (deleted org should not be visible)", w.Code)
	}
}

func TestOrgController_Rm_NonAdmin(t *testing.T) {
	db := setupOrgTestDB(t)
	owner := seedOrgUser(t, db)
	other := &model.TUser{Id: "other-user", Name: "other", Nick: "Other", Pass: "x", Active: 1, Created: time.Now(), LoginTime: time.Now()}
	if _, err := db.InsertOne(other); err != nil {
		t.Fatal(err)
	}

	// Owner creates org, other user tries to delete
	o := seedOrg(t, db, owner.Id, "org-rm", "rm", true)

	// Login as other user (not owner, not admin)
	r := setupOrgRouter(t, db, other)
	w := orgPostJSON(t, r, "/api/org/rm", map[string]any{"id": o.Id})

	// other is not owner and not org admin → should fail
	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("status = %d, want 405; body = %s", w.Code, w.Body.String())
	}
}

func TestOrgController_VarSave_ValidationErrors(t *testing.T) {
	db := setupOrgTestDB(t)
	u := seedOrgUser(t, db)
	r := setupOrgRouter(t, db, u)

	cases := []map[string]any{
		{"name": "", "value": "v", "orgId": "o"},
		{"name": "n", "value": "", "orgId": "o"},
		{"name": "n", "value": "v", "orgId": ""},
	}
	for i, c := range cases {
		w := orgPostJSON(t, r, "/api/org/var/save", c)
		if w.Code != http.StatusBadRequest {
			t.Errorf("case %d: status = %d, want 400", i, w.Code)
		}
	}
}

func TestOrgController_VarSave_OrgNotFound(t *testing.T) {
	db := setupOrgTestDB(t)
	u := seedOrgUser(t, db)
	r := setupOrgRouter(t, db, u)

	w := orgPostJSON(t, r, "/api/org/var/save", map[string]any{
		"name": "k", "value": "v", "orgId": "nope",
	})
	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want 404; body = %s", w.Code, w.Body.String())
	}
}

func TestOrgController_VarDel_InvalidAid(t *testing.T) {
	db := setupOrgTestDB(t)
	u := seedOrgUser(t, db)
	r := setupOrgRouter(t, db, u)

	for _, aid := range []any{0, -1, "not-a-number"} {
		w := orgPostJSON(t, r, "/api/org/var/del", map[string]any{"aid": aid})
		if w.Code != http.StatusBadRequest {
			t.Errorf("aid=%v: status = %d, want 400", aid, w.Code)
		}
	}
}

func TestOrgController_VarDel_NotFound(t *testing.T) {
	db := setupOrgTestDB(t)
	u := seedOrgUser(t, db)
	r := setupOrgRouter(t, db, u)

	w := orgPostJSON(t, r, "/api/org/var/del", map[string]any{"aid": 999})
	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want 404", w.Code)
	}
}

func TestOrgController_VarDel_OrgNotFound(t *testing.T) {
	db := setupOrgTestDB(t)
	u := seedOrgUser(t, db)
	// Insert a var whose org_id doesn't exist — var lookup succeeds, org lookup fails.
	if _, err := db.Exec(`INSERT INTO t_org_var (org_id, name, value) VALUES ('ghost', 'k', 'v')`); err != nil {
		t.Fatal(err)
	}
	r := setupOrgRouter(t, db, u)

	// Look up the inserted row to get its aid.
	var row struct{ Aid int64 }
	ok, err := db.SQL("SELECT aid FROM t_org_var WHERE org_id='ghost'").Get(&row)
	if err != nil || !ok {
		t.Fatalf("seed var: ok=%v err=%v", ok, err)
	}

	w := orgPostJSON(t, r, "/api/org/var/del", map[string]any{"aid": row.Aid})
	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want 404; body = %s", w.Code, w.Body.String())
	}
}

func TestOrgController_UserRm_CannotRemoveSelf(t *testing.T) {
	db := setupOrgTestDB(t)
	u := seedOrgUser(t, db)
	o := seedOrg(t, db, u.Id, "org-self", "self", true)
	// Make u an org admin so permission check passes.
	if _, err := db.Exec(`INSERT INTO t_user_org (uid, org_id, perm_adm, created) VALUES (?,?,1,?)`,
		u.Id, o.Id, time.Now()); err != nil {
		t.Fatal(err)
	}
	r := setupOrgRouter(t, db, u)

	w := orgPostJSON(t, r, "/api/org/user/rm", map[string]any{
		"id": o.Id, "uid": u.Id,
	})
	if w.Code != http.StatusConflict {
		t.Errorf("status = %d, want 409; body = %s", w.Code, w.Body.String())
	}
}

func TestOrgController_UserEdit_CannotEditSelf(t *testing.T) {
	db := setupOrgTestDB(t)
	u := seedOrgUser(t, db)
	o := seedOrg(t, db, u.Id, "org-edit", "edit", true)
	if _, err := db.Exec(`INSERT INTO t_user_org (uid, org_id, perm_adm, perm_rw, perm_exec, perm_down, created)
		VALUES (?,?,1,1,1,1,?)`, u.Id, o.Id, time.Now()); err != nil {
		t.Fatal(err)
	}
	r := setupOrgRouter(t, db, u)

	w := orgPostJSON(t, r, "/api/org/user/edit", map[string]any{
		"id": o.Id, "uid": u.Id, "adm": false, "rw": true, "ex": true, "dw": true, "add": false,
	})
	if w.Code != http.StatusConflict {
		t.Errorf("status = %d, want 409; body = %s", w.Code, w.Body.String())
	}
}

func TestOrgController_PipeRm_NonAdmin(t *testing.T) {
	db := setupOrgTestDB(t)
	owner := seedOrgUser(t, db)
	other := &model.TUser{Id: "other-user", Name: "other", Nick: "Other", Pass: "x", Active: 1, Created: time.Now(), LoginTime: time.Now()}
	if _, err := db.InsertOne(other); err != nil {
		t.Fatal(err)
	}

	// Owner creates org, other user tries to remove pipe
	o := seedOrg(t, db, owner.Id, "org-pr", "pr", true)

	// Login as other user (not owner, not admin)
	r := setupOrgRouter(t, db, other)
	w := orgPostJSON(t, r, "/api/org/pipe/rm", map[string]any{"id": o.Id, "pipeId": "p1"})

	// other is not owner and not org admin → should fail
	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("status = %d, want 405; body = %s", w.Code, w.Body.String())
	}
}

func TestOrgController_Vars_EmptyOrgId(t *testing.T) {
	db := setupOrgTestDB(t)
	u := seedOrgUser(t, db)
	r := setupOrgRouter(t, db, u)

	w := orgPostJSON(t, r, "/api/org/vars", map[string]any{"orgId": ""})
	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", w.Code)
	}
}

func TestOrgController_Vars_OrgNotFound(t *testing.T) {
	db := setupOrgTestDB(t)
	u := seedOrgUser(t, db)
	r := setupOrgRouter(t, db, u)

	w := orgPostJSON(t, r, "/api/org/vars", map[string]any{"orgId": "nope"})
	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want 404", w.Code)
	}
}

func TestOrgController_Save_MissingName(t *testing.T) {
	db := setupOrgTestDB(t)
	u := seedOrgUser(t, db)
	o := seedOrg(t, db, u.Id, "org-save", "save", true)
	r := setupOrgRouter(t, db, u)

	w := orgPostJSON(t, r, "/api/org/save", map[string]any{"id": o.Id, "name": ""})
	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", w.Code)
	}
}

func TestOrgController_Users_EmptyId(t *testing.T) {
	db := setupOrgTestDB(t)
	u := seedOrgUser(t, db)
	r := setupOrgRouter(t, db, u)

	w := orgPostJSON(t, r, "/api/org/users", map[string]any{"id": ""})
	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", w.Code)
	}
}
