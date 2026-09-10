package route

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gokins/core/utils"
	"github.com/gokins/gokins/comm"
	"github.com/gokins/gokins/model"
	"github.com/gokins/gokins/service"
	"github.com/gokins/gokins/util"
	"github.com/golang-jwt/jwt/v5"
	_ "github.com/mattn/go-sqlite3"
	"xorm.io/xorm"
)

func setupOrgTestDB(t *testing.T) {
	t.Helper()
	origDb := comm.Db
	t.Cleanup(func() { comm.Db = origDb })

	db, err := xorm.NewEngine("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	// Create all required tables
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
			name VARCHAR(100),
			desc TEXT,
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
			created DATETIME,
			public INT DEFAULT 0
		)`,
		`CREATE TABLE t_org_var (
			aid INTEGER PRIMARY KEY AUTOINCREMENT,
			uid VARCHAR(64),
			org_id VARCHAR(64),
			name VARCHAR(100),
			value TEXT,
			remarks TEXT,
			public INT DEFAULT 0
		)`,
	}

	for _, sql := range tables {
		if _, err := db.Exec(sql); err != nil {
			t.Fatalf("create table: %v", err)
		}
	}

	comm.Db = db
}

//nolint:unparam // test helper designed for reusability
func createOrgTestUser(t *testing.T, name, nick string, active int, permOrg int) *model.TUser {
	t.Helper()
	user := &model.TUser{
		Id:        utils.NewXid(),
		Name:      name,
		Nick:      nick,
		Active:    active,
		Created:   time.Now(),
		LoginTime: time.Now(),
	}
	if _, err := comm.Db.InsertOne(user); err != nil {
		t.Fatalf("create test user: %v", err)
	}

	if permOrg > 0 {
		info := &model.TUserInfo{
			Id:      user.Id,
			PermOrg: permOrg,
		}
		if _, err := comm.Db.InsertOne(info); err != nil {
			t.Fatalf("create user info: %v", err)
		}
	}

	return user
}

func makeOrgTestToken(t *testing.T, uid string) string {
	t.Helper()
	comm.Cfg.Server.LoginKey = "test-secret-key-for-org-tests"
	token, err := util.CreateToken(jwt.MapClaims{
		"uid": uid,
	}, comm.Cfg.Server.LoginKey, time.Hour*24)
	if err != nil {
		t.Fatalf("create token: %v", err)
	}
	return token
}

func setupOrgTestRouter(t *testing.T) *gin.Engine {
	t.Helper()
	gin.SetMode(gin.TestMode)
	r := gin.New()

	// Add middleware that injects lguser
	r.Use(func(c *gin.Context) {
		usr, ok := service.CurrUserCache(c)
		if !ok || (!service.IsAdmin(usr) && usr.Active != 1) {
			c.String(http.StatusForbidden, "Not Auth")
			c.Abort()
			return
		}
		c.Set(service.LgUserKey, usr)
		c.Next()
	})

	oc := &OrgController{}
	oc.Routes(r.Group("/api/org"))
	return r
}

func TestOrg_list_AsAdmin(t *testing.T) {
	setupOrgTestDB(t)
	admin := createOrgTestUser(t, "admin", "Admin User", 1, 1)

	// Create test orgs
	for i := 0; i < 3; i++ {
		org := &model.TOrg{
			Id:      utils.NewXid(),
			Uid:     admin.Id,
			Name:    utils.RandomString(8),
			Public:  1,
			Created: time.Now(),
			Updated: time.Now(),
		}
		if _, err := comm.Db.InsertOne(org); err != nil {
			t.Fatalf("create org: %v", err)
		}
	}

	r := setupOrgTestRouter(t)
	token := makeOrgTestToken(t, admin.Id)

	req := httptest.NewRequest(http.MethodPost, "/api/org/list", bytes.NewBufferString(`{"page":1}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "TOKEN "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	var resp map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if resp["data"] == nil {
		t.Error("expected data field in response")
	}
}

func TestOrg_list_AsNonAdmin_PublicOrgs(t *testing.T) {
	setupOrgTestDB(t)
	user := createOrgTestUser(t, "user1", "User One", 1, 0)

	// Create public org
	publicOrg := &model.TOrg{
		Id:      utils.NewXid(),
		Uid:     user.Id,
		Name:    "public-org",
		Public:  1,
		Created: time.Now(),
		Updated: time.Now(),
	}
	if _, err := comm.Db.InsertOne(publicOrg); err != nil {
		t.Fatalf("create public org: %v", err)
	}

	r := setupOrgTestRouter(t)
	token := makeOrgTestToken(t, user.Id)

	req := httptest.NewRequest(http.MethodPost, "/api/org/list", bytes.NewBufferString(`{}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "TOKEN "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}
}

func TestOrg_new_Success(t *testing.T) {
	setupOrgTestDB(t)
	admin := createOrgTestUser(t, "admin", "Admin", 1, 1)

	r := setupOrgTestRouter(t)
	token := makeOrgTestToken(t, admin.Id)

	body := map[string]any{
		"name":   "test-org",
		"desc":   "Test organization",
		"public": true,
	}
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/api/org/new", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "TOKEN "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	var resp map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if resp["id"] == nil || resp["id"] == "" {
		t.Error("expected id in response")
	}
}

func TestOrg_new_EmptyName(t *testing.T) {
	setupOrgTestDB(t)
	user := createOrgTestUser(t, "user1", "User", 1, 1)

	r := setupOrgTestRouter(t)
	token := makeOrgTestToken(t, user.Id)

	body := map[string]any{
		"name": "",
		"desc": "desc",
	}
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/api/org/new", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "TOKEN "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d, body: %s", w.Code, http.StatusBadRequest, w.Body.String())
	}
}

func TestOrg_info_Success(t *testing.T) {
	setupOrgTestDB(t)
	user := createOrgTestUser(t, "user1", "User", 1, 1)

	org := &model.TOrg{
		Id:      utils.NewXid(),
		Uid:     user.Id,
		Name:    "test-org",
		Public:  1,
		Created: time.Now(),
		Updated: time.Now(),
	}
	if _, err := comm.Db.InsertOne(org); err != nil {
		t.Fatalf("create org: %v", err)
	}

	r := setupOrgTestRouter(t)
	token := makeOrgTestToken(t, user.Id)

	body := map[string]any{"id": org.Id}
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/api/org/info", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "TOKEN "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	var resp map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if resp["org"] == nil {
		t.Error("expected org in response")
	}
}

func TestOrg_info_NotFound(t *testing.T) {
	setupOrgTestDB(t)
	user := createOrgTestUser(t, "user1", "User", 1, 1)

	r := setupOrgTestRouter(t)
	token := makeOrgTestToken(t, user.Id)

	body := map[string]any{"id": "nonexistent"}
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/api/org/info", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "TOKEN "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d, body: %s", w.Code, http.StatusNotFound, w.Body.String())
	}
}

func TestOrg_users_Success(t *testing.T) {
	setupOrgTestDB(t)
	user := createOrgTestUser(t, "user1", "User", 1, 1)

	org := &model.TOrg{
		Id:      utils.NewXid(),
		Uid:     user.Id,
		Name:    "test-org",
		Public:  1,
		Created: time.Now(),
		Updated: time.Now(),
	}
	if _, err := comm.Db.InsertOne(org); err != nil {
		t.Fatalf("create org: %v", err)
	}

	// Add user to org as admin
	userOrg := &model.TUserOrg{
		Uid:      user.Id,
		OrgId:    org.Id,
		PermAdm:  1,
		PermRw:   1,
		PermExec: 1,
		Created:  time.Now(),
	}
	if _, err := comm.Db.InsertOne(userOrg); err != nil {
		t.Fatalf("create user org: %v", err)
	}

	r := setupOrgTestRouter(t)
	token := makeOrgTestToken(t, user.Id)

	body := map[string]any{"id": org.Id}
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/api/org/users", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "TOKEN "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	var resp map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if resp["adms"] == nil {
		t.Error("expected adms field in response")
	}
}

func TestOrg_save_Success(t *testing.T) {
	setupOrgTestDB(t)
	user := createOrgTestUser(t, "user1", "User", 1, 1)

	org := &model.TOrg{
		Id:      utils.NewXid(),
		Uid:     user.Id,
		Name:    "test-org",
		Public:  1,
		Created: time.Now(),
		Updated: time.Now(),
	}
	if _, err := comm.Db.InsertOne(org); err != nil {
		t.Fatalf("create org: %v", err)
	}

	userOrg := &model.TUserOrg{
		Uid:     user.Id,
		OrgId:   org.Id,
		PermAdm: 1,
		Created: time.Now(),
	}
	if _, err := comm.Db.InsertOne(userOrg); err != nil {
		t.Fatalf("create user org: %v", err)
	}

	r := setupOrgTestRouter(t)
	token := makeOrgTestToken(t, user.Id)

	body := map[string]any{
		"id":     org.Id,
		"name":   "updated-org",
		"desc":   "Updated description",
		"public": false,
	}
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/api/org/save", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "TOKEN "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}
}

func TestOrg_rm_Success(t *testing.T) {
	setupOrgTestDB(t)
	user := createOrgTestUser(t, "user1", "User", 1, 1)

	org := &model.TOrg{
		Id:      utils.NewXid(),
		Uid:     user.Id,
		Name:    "test-org",
		Public:  1,
		Created: time.Now(),
		Updated: time.Now(),
	}
	if _, err := comm.Db.InsertOne(org); err != nil {
		t.Fatalf("create org: %v", err)
	}

	userOrg := &model.TUserOrg{
		Uid:     user.Id,
		OrgId:   org.Id,
		PermAdm: 1,
		Created: time.Now(),
	}
	if _, err := comm.Db.InsertOne(userOrg); err != nil {
		t.Fatalf("create user org: %v", err)
	}

	r := setupOrgTestRouter(t)
	token := makeOrgTestToken(t, user.Id)

	body := map[string]any{"id": org.Id}
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/api/org/rm", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "TOKEN "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	// Verify org is soft-deleted
	updatedOrg := &model.TOrg{}
	if ok, err := comm.Db.Where("id=?", org.Id).Get(updatedOrg); err != nil || !ok {
		t.Fatalf("query org: %v", err)
	}
	if updatedOrg.Deleted != 1 {
		t.Error("expected org to be soft-deleted")
	}
}

func TestOrg_rm_NotFound(t *testing.T) {
	setupOrgTestDB(t)
	user := createOrgTestUser(t, "user1", "User", 1, 1)

	r := setupOrgTestRouter(t)
	token := makeOrgTestToken(t, user.Id)

	body := map[string]any{"id": "nonexistent"}
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/api/org/rm", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "TOKEN "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d, body: %s", w.Code, http.StatusNotFound, w.Body.String())
	}
}

func TestOrg_userEdit_AddMember(t *testing.T) {
	setupOrgTestDB(t)
	admin := createOrgTestUser(t, "admin", "Admin", 1, 1)
	newMember := createOrgTestUser(t, "member", "Member", 1, 0)

	org := &model.TOrg{
		Id:      utils.NewXid(),
		Uid:     admin.Id,
		Name:    "test-org",
		Public:  1,
		Created: time.Now(),
		Updated: time.Now(),
	}
	if _, err := comm.Db.InsertOne(org); err != nil {
		t.Fatalf("create org: %v", err)
	}

	adminUserOrg := &model.TUserOrg{
		Uid:     admin.Id,
		OrgId:   org.Id,
		PermAdm: 1,
		Created: time.Now(),
	}
	if _, err := comm.Db.InsertOne(adminUserOrg); err != nil {
		t.Fatalf("create admin user org: %v", err)
	}

	r := setupOrgTestRouter(t)
	token := makeOrgTestToken(t, admin.Id)

	body := map[string]any{
		"id":  org.Id,
		"uid": newMember.Id,
		"adm": false,
		"rw":  true,
		"ex":  true,
		"dw":  true,
		"add": false,
	}
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/api/org/user/edit", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "TOKEN "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	// Verify member was added
	userOrg := &model.TUserOrg{}
	ok, err := comm.Db.Where("uid=? and org_id=?", newMember.Id, org.Id).Get(userOrg)
	if err != nil {
		t.Fatalf("query user org: %v", err)
	}
	if !ok {
		t.Error("expected user to be added to org")
	}
}

func TestOrg_userEdit_CannotEditSelf(t *testing.T) {
	setupOrgTestDB(t)
	admin := createOrgTestUser(t, "admin", "Admin", 1, 1)

	org := &model.TOrg{
		Id:      utils.NewXid(),
		Uid:     admin.Id,
		Name:    "test-org",
		Public:  1,
		Created: time.Now(),
		Updated: time.Now(),
	}
	if _, err := comm.Db.InsertOne(org); err != nil {
		t.Fatalf("create org: %v", err)
	}

	adminUserOrg := &model.TUserOrg{
		Uid:     admin.Id,
		OrgId:   org.Id,
		PermAdm: 1,
		Created: time.Now(),
	}
	if _, err := comm.Db.InsertOne(adminUserOrg); err != nil {
		t.Fatalf("create admin user org: %v", err)
	}

	r := setupOrgTestRouter(t)
	token := makeOrgTestToken(t, admin.Id)

	body := map[string]any{
		"id":  org.Id,
		"uid": admin.Id,
		"adm": false,
		"rw":  true,
	}
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/api/org/user/edit", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "TOKEN "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusConflict {
		t.Errorf("status = %d, want %d, body: %s", w.Code, http.StatusConflict, w.Body.String())
	}
}

func TestOrg_userRm_Success(t *testing.T) {
	setupOrgTestDB(t)
	admin := createOrgTestUser(t, "admin", "Admin", 1, 1)
	member := createOrgTestUser(t, "member", "Member", 1, 0)

	org := &model.TOrg{
		Id:      utils.NewXid(),
		Uid:     admin.Id,
		Name:    "test-org",
		Public:  1,
		Created: time.Now(),
		Updated: time.Now(),
	}
	if _, err := comm.Db.InsertOne(org); err != nil {
		t.Fatalf("create org: %v", err)
	}

	adminUserOrg := &model.TUserOrg{
		Uid:     admin.Id,
		OrgId:   org.Id,
		PermAdm: 1,
		Created: time.Now(),
	}
	if _, err := comm.Db.InsertOne(adminUserOrg); err != nil {
		t.Fatalf("create admin user org: %v", err)
	}

	memberUserOrg := &model.TUserOrg{
		Uid:     member.Id,
		OrgId:   org.Id,
		PermRw:  1,
		Created: time.Now(),
	}
	if _, err := comm.Db.InsertOne(memberUserOrg); err != nil {
		t.Fatalf("create member user org: %v", err)
	}

	r := setupOrgTestRouter(t)
	token := makeOrgTestToken(t, admin.Id)

	body := map[string]any{
		"id":  org.Id,
		"uid": member.Id,
	}
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/api/org/user/rm", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "TOKEN "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	// Verify member was removed
	count, err := comm.Db.Where("uid=? and org_id=?", member.Id, org.Id).Count(&model.TUserOrg{})
	if err != nil {
		t.Fatalf("count user org: %v", err)
	}
	if count != 0 {
		t.Error("expected user to be removed from org")
	}
}

func TestOrg_pipeAdd_Success(t *testing.T) {
	setupOrgTestDB(t)
	user := createOrgTestUser(t, "user1", "User", 1, 1)

	org := &model.TOrg{
		Id:      utils.NewXid(),
		Uid:     user.Id,
		Name:    "test-org",
		Public:  1,
		Created: time.Now(),
		Updated: time.Now(),
	}
	if _, err := comm.Db.InsertOne(org); err != nil {
		t.Fatalf("create org: %v", err)
	}

	userOrg := &model.TUserOrg{
		Uid:     user.Id,
		OrgId:   org.Id,
		PermAdm: 1,
		Created: time.Now(),
	}
	if _, err := comm.Db.InsertOne(userOrg); err != nil {
		t.Fatalf("create user org: %v", err)
	}

	pipeId := utils.NewXid()

	r := setupOrgTestRouter(t)
	token := makeOrgTestToken(t, user.Id)

	body := map[string]any{
		"id":     org.Id,
		"pipeId": pipeId,
	}
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/api/org/pipe/add", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "TOKEN "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	// Verify pipeline was added
	orgPipe := &model.TOrgPipe{}
	ok, err := comm.Db.Where("org_id=? and pipe_id=?", org.Id, pipeId).Get(orgPipe)
	if err != nil {
		t.Fatalf("query org pipe: %v", err)
	}
	if !ok {
		t.Error("expected pipeline to be added to org")
	}
}

func TestOrg_pipeAdd_Duplicate(t *testing.T) {
	setupOrgTestDB(t)
	user := createOrgTestUser(t, "user1", "User", 1, 1)

	org := &model.TOrg{
		Id:      utils.NewXid(),
		Uid:     user.Id,
		Name:    "test-org",
		Public:  1,
		Created: time.Now(),
		Updated: time.Now(),
	}
	if _, err := comm.Db.InsertOne(org); err != nil {
		t.Fatalf("create org: %v", err)
	}

	userOrg := &model.TUserOrg{
		Uid:     user.Id,
		OrgId:   org.Id,
		PermAdm: 1,
		Created: time.Now(),
	}
	if _, err := comm.Db.InsertOne(userOrg); err != nil {
		t.Fatalf("create user org: %v", err)
	}

	pipeId := utils.NewXid()
	existingOrgPipe := &model.TOrgPipe{
		OrgId:   org.Id,
		PipeId:  pipeId,
		Created: time.Now(),
	}
	if _, err := comm.Db.InsertOne(existingOrgPipe); err != nil {
		t.Fatalf("create existing org pipe: %v", err)
	}

	r := setupOrgTestRouter(t)
	token := makeOrgTestToken(t, user.Id)

	body := map[string]any{
		"id":     org.Id,
		"pipeId": pipeId,
	}
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/api/org/pipe/add", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "TOKEN "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusConflict {
		t.Errorf("status = %d, want %d, body: %s", w.Code, http.StatusConflict, w.Body.String())
	}
}

func TestOrg_pipeRm_Success(t *testing.T) {
	setupOrgTestDB(t)
	user := createOrgTestUser(t, "user1", "User", 1, 1)

	org := &model.TOrg{
		Id:      utils.NewXid(),
		Uid:     user.Id,
		Name:    "test-org",
		Public:  1,
		Created: time.Now(),
		Updated: time.Now(),
	}
	if _, err := comm.Db.InsertOne(org); err != nil {
		t.Fatalf("create org: %v", err)
	}

	userOrg := &model.TUserOrg{
		Uid:     user.Id,
		OrgId:   org.Id,
		PermAdm: 1,
		Created: time.Now(),
	}
	if _, err := comm.Db.InsertOne(userOrg); err != nil {
		t.Fatalf("create user org: %v", err)
	}

	pipeId := utils.NewXid()
	orgPipe := &model.TOrgPipe{
		OrgId:   org.Id,
		PipeId:  pipeId,
		Created: time.Now(),
	}
	if _, err := comm.Db.InsertOne(orgPipe); err != nil {
		t.Fatalf("create org pipe: %v", err)
	}

	r := setupOrgTestRouter(t)
	token := makeOrgTestToken(t, user.Id)

	body := map[string]any{
		"id":     org.Id,
		"pipeId": pipeId,
	}
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/api/org/pipe/rm", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "TOKEN "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	// Verify pipeline was removed
	count, err := comm.Db.Where("org_id=? and pipe_id=?", org.Id, pipeId).Count(&model.TOrgPipe{})
	if err != nil {
		t.Fatalf("count org pipes: %v", err)
	}
	if count != 0 {
		t.Error("expected pipeline to be removed from org")
	}
}

func TestOrg_vars_EmptyOrgId(t *testing.T) {
	setupOrgTestDB(t)
	user := createOrgTestUser(t, "user1", "User", 1, 1)

	r := setupOrgTestRouter(t)
	token := makeOrgTestToken(t, user.Id)

	body := map[string]any{"orgId": ""}
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/api/org/vars", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "TOKEN "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d, body: %s", w.Code, http.StatusBadRequest, w.Body.String())
	}
}

func TestOrg_varDel_Success(t *testing.T) {
	setupOrgTestDB(t)
	user := createOrgTestUser(t, "user1", "User", 1, 1)

	org := &model.TOrg{
		Id:      utils.NewXid(),
		Uid:     user.Id,
		Name:    "test-org",
		Public:  1,
		Created: time.Now(),
		Updated: time.Now(),
	}
	if _, err := comm.Db.InsertOne(org); err != nil {
		t.Fatalf("create org: %v", err)
	}

	userOrg := &model.TUserOrg{
		Uid:     user.Id,
		OrgId:   org.Id,
		PermAdm: 1,
		PermRw:  1,
		Created: time.Now(),
	}
	if _, err := comm.Db.InsertOne(userOrg); err != nil {
		t.Fatalf("create user org: %v", err)
	}

	orgVar := &model.TOrgVar{
		OrgId:  org.Id,
		Name:   "test-var",
		Value:  "test-value",
		Public: 1,
	}
	if _, err := comm.Db.InsertOne(orgVar); err != nil {
		t.Fatalf("create org var: %v", err)
	}

	r := setupOrgTestRouter(t)
	token := makeOrgTestToken(t, user.Id)

	body := map[string]any{"aid": orgVar.Aid}
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/api/org/var/del", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "TOKEN "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	// Verify var was deleted
	count, err := comm.Db.Where("aid=?", orgVar.Aid).Count(&model.TOrgVar{})
	if err != nil {
		t.Fatalf("count org vars: %v", err)
	}
	if count != 0 {
		t.Error("expected org var to be deleted")
	}
}

func TestOrg_varDel_NotFound(t *testing.T) {
	setupOrgTestDB(t)
	user := createOrgTestUser(t, "user1", "User", 1, 1)

	r := setupOrgTestRouter(t)
	token := makeOrgTestToken(t, user.Id)

	body := map[string]any{"aid": 99999}
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/api/org/var/del", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "TOKEN "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d, body: %s", w.Code, http.StatusNotFound, w.Body.String())
	}
}
