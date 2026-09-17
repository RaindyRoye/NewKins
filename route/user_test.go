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
	_ "github.com/mattn/go-sqlite3"
	hbtp "github.com/mgr9525/HyperByte-Transfer-Protocol"
	"xorm.io/xorm"
)

func setupUserTestDB(t *testing.T) {
	t.Helper()
	origDb := comm.Db
	t.Cleanup(func() { comm.Db = origDb })

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
			perm_user INT DEFAULT 0,
			perm_org INT DEFAULT 0,
			perm_pipe INT DEFAULT 0
		)`,
	}

	for _, sql := range tables {
		if _, err := db.Exec(sql); err != nil {
			t.Fatalf("exec %q: %v", sql[:40], err)
		}
	}

	comm.Db = db
}

func createUserTestUser(t *testing.T, name, nick string) *model.TUser {
	t.Helper()
	usr := &model.TUser{
		Id:        utils.NewXid(),
		Name:      name,
		Nick:      nick,
		Active:    1,
		Created:   time.Now(),
		LoginTime: time.Now(),
	}
	if _, err := comm.Db.InsertOne(usr); err != nil {
		t.Fatalf("create user: %v", err)
	}
	return usr
}

func makeUserGinCtx(t *testing.T, body interface{}, lgUser *model.TUser) (*gin.Context, *httptest.ResponseRecorder) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	var req *http.Request
	if body != nil {
		bts, _ := json.Marshal(body)
		req = httptest.NewRequest("POST", "/test", bytes.NewReader(bts))
	} else {
		req = httptest.NewRequest("POST", "/test", nil)
	}
	req.Header.Set("Content-Type", "application/json")
	c.Request = req

	if lgUser != nil {
		c.Set(service.LgUserKey, lgUser)
	}
	return c, w
}

func TestUserController_page(t *testing.T) {
	setupUserTestDB(t)
	user1 := createUserTestUser(t, "alice", "Alice")
	createUserTestUser(t, "bob", "Bob")
	createUserTestUser(t, "charlie", "Charlie")

	ctrl := UserController{}
	m := &hbtp.Map{}
	m.Set("q", "")
	m.Set("page", int64(1))

	c, w := makeUserGinCtx(t, m, user1)
	ctrl.page(c, m)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, body: %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if resp["data"] == nil {
		t.Fatal("expected data in response")
	}
}

func TestUserController_page_withQuery(t *testing.T) {
	setupUserTestDB(t)
	user1 := createUserTestUser(t, "alice", "Alice")
	createUserTestUser(t, "bob", "Bob")

	ctrl := UserController{}
	m := &hbtp.Map{}
	m.Set("q", "alice")
	m.Set("page", int64(1))

	c, w := makeUserGinCtx(t, m, user1)
	ctrl.page(c, m)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, body: %s", w.Code, w.Body.String())
	}
}

func TestUserController_new_adminSuccess(t *testing.T) {
	setupUserTestDB(t)
	admin := createUserTestUser(t, "admin", "Admin")

	// Create user info with perm_user = 1
	uinfo := &model.TUserInfo{
		Id:       admin.Id,
		PermUser: 1,
	}
	if _, err := comm.Db.InsertOne(uinfo); err != nil {
		t.Fatalf("insert user info: %v", err)
	}

	ctrl := UserController{}
	m := &hbtp.Map{}
	m.Set("name", "newuser")
	m.Set("nick", "New User")
	m.Set("pass", "password123")

	c, w := makeUserGinCtx(t, m, admin)
	ctrl.new(c, m)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, body: %s", w.Code, w.Body.String())
	}

	// Verify user was created
	var usr model.TUser
	has, err := comm.Db.Where("name=?", "newuser").Get(&usr)
	if err != nil {
		t.Fatalf("query user: %v", err)
	}
	if !has {
		t.Fatal("user not created")
	}
}

func TestUserController_new_paramError(t *testing.T) {
	setupUserTestDB(t)
	admin := createUserTestUser(t, "admin", "Admin")

	ctrl := UserController{}
	m := &hbtp.Map{}
	m.Set("name", "")
	m.Set("nick", "New User")
	m.Set("pass", "password123")

	c, w := makeUserGinCtx(t, m, admin)
	ctrl.new(c, m)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestUserController_new_duplicateUser(t *testing.T) {
	setupUserTestDB(t)
	admin := createUserTestUser(t, "admin", "Admin")
	createUserTestUser(t, "alice", "Alice")

	uinfo := &model.TUserInfo{
		Id:       admin.Id,
		PermUser: 1,
	}
	if _, err := comm.Db.InsertOne(uinfo); err != nil {
		t.Fatalf("insert user info: %v", err)
	}

	ctrl := UserController{}
	m := &hbtp.Map{}
	m.Set("name", "alice")
	m.Set("nick", "New Alice")
	m.Set("pass", "password123")

	c, w := makeUserGinCtx(t, m, admin)
	ctrl.new(c, m)

	if w.Code != http.StatusConflict {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusConflict)
	}
}

func TestUserController_new_noPermission(t *testing.T) {
	setupUserTestDB(t)
	user1 := createUserTestUser(t, "regular", "Regular")

	ctrl := UserController{}
	m := &hbtp.Map{}
	m.Set("name", "newuser")
	m.Set("nick", "New User")
	m.Set("pass", "password123")

	c, w := makeUserGinCtx(t, m, user1)
	ctrl.new(c, m)

	if w.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusMethodNotAllowed)
	}
}

func TestUserController_info(t *testing.T) {
	setupUserTestDB(t)
	user1 := createUserTestUser(t, "alice", "Alice")

	ctrl := UserController{}
	m := &hbtp.Map{}
	m.Set("id", user1.Id)

	c, w := makeUserGinCtx(t, m, user1)
	ctrl.info(c, m)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, body: %s", w.Code, w.Body.String())
	}
}

func TestUserController_info_paramError(t *testing.T) {
	setupUserTestDB(t)
	user1 := createUserTestUser(t, "alice", "Alice")

	ctrl := UserController{}
	m := &hbtp.Map{}
	m.Set("id", "")

	c, w := makeUserGinCtx(t, m, user1)
	ctrl.info(c, m)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestUserController_info_notFound(t *testing.T) {
	setupUserTestDB(t)
	user1 := createUserTestUser(t, "alice", "Alice")

	ctrl := UserController{}
	m := &hbtp.Map{}
	m.Set("id", "nonexistent")

	c, w := makeUserGinCtx(t, m, user1)
	ctrl.info(c, m)

	if w.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestUserController_upinfo_success(t *testing.T) {
	setupUserTestDB(t)
	user1 := createUserTestUser(t, "alice", "Alice")

	ctrl := UserController{}
	m := &hbtp.Map{}
	m.Set("id", user1.Id)
	m.Set("nick", "Alice Updated")
	m.Set("phone", "1234567890")
	m.Set("email", "alice@example.com")
	m.Set("remark", "Test remark")

	c, w := makeUserGinCtx(t, m, user1)
	ctrl.upinfo(c, m)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, body: %s", w.Code, w.Body.String())
	}
}

func TestUserController_upinfo_paramError(t *testing.T) {
	setupUserTestDB(t)
	user1 := createUserTestUser(t, "alice", "Alice")

	ctrl := UserController{}
	m := &hbtp.Map{}
	m.Set("id", "")
	m.Set("nick", "Alice Updated")

	c, w := makeUserGinCtx(t, m, user1)
	ctrl.upinfo(c, m)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestUserController_upinfo_notYou(t *testing.T) {
	setupUserTestDB(t)
	user1 := createUserTestUser(t, "alice", "Alice")
	user2 := createUserTestUser(t, "bob", "Bob")

	ctrl := UserController{}
	m := &hbtp.Map{}
	m.Set("id", user2.Id)
	m.Set("nick", "Bob Updated")

	c, w := makeUserGinCtx(t, m, user1)
	ctrl.upinfo(c, m)

	if w.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusMethodNotAllowed)
	}
}

func TestUserController_upass_selfSuccess(t *testing.T) {
	setupUserTestDB(t)
	user1 := createUserTestUser(t, "alice", "Alice")
	user1.Pass = utils.Md5String("oldpass")
	if _, err := comm.Db.Where("id=?", user1.Id).Cols("pass").Update(user1); err != nil {
		t.Fatalf("update pass: %v", err)
	}

	ctrl := UserController{}
	m := &hbtp.Map{}
	m.Set("id", user1.Id)
	m.Set("olds", "oldpass")
	m.Set("pass", "newpass123")

	c, w := makeUserGinCtx(t, m, user1)
	ctrl.upass(c, m)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, body: %s", w.Code, w.Body.String())
	}
}

func TestUserController_upass_wrongOldPass(t *testing.T) {
	setupUserTestDB(t)
	user1 := createUserTestUser(t, "alice", "Alice")
	user1.Pass = utils.Md5String("oldpass")
	if _, err := comm.Db.Where("id=?", user1.Id).Cols("pass").Update(user1); err != nil {
		t.Fatalf("update pass: %v", err)
	}

	ctrl := UserController{}
	m := &hbtp.Map{}
	m.Set("id", user1.Id)
	m.Set("olds", "wrongpass")
	m.Set("pass", "newpass123")

	c, w := makeUserGinCtx(t, m, user1)
	ctrl.upass(c, m)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusUnauthorized)
	}
}

func TestUserController_upass_paramError(t *testing.T) {
	setupUserTestDB(t)
	user1 := createUserTestUser(t, "alice", "Alice")

	ctrl := UserController{}
	m := &hbtp.Map{}
	m.Set("id", "")
	m.Set("pass", "newpass123")

	c, w := makeUserGinCtx(t, m, user1)
	ctrl.upass(c, m)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestUserController_active_success(t *testing.T) {
	setupUserTestDB(t)
	admin := &model.TUser{
		Id:        "admin",
		Name:      "admin",
		Nick:      "Admin",
		Active:    1,
		Created:   time.Now(),
		LoginTime: time.Now(),
	}
	if _, err := comm.Db.InsertOne(admin); err != nil {
		t.Fatalf("insert admin: %v", err)
	}
	user1 := createUserTestUser(t, "alice", "Alice")

	ctrl := UserController{}
	m := &hbtp.Map{}
	m.Set("id", user1.Id)
	m.Set("act", "1")

	c, w := makeUserGinCtx(t, m, admin)
	ctrl.active(c, m)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, body: %s", w.Code, w.Body.String())
	}
}

func TestUserController_active_noPermission(t *testing.T) {
	setupUserTestDB(t)
	user1 := createUserTestUser(t, "alice", "Alice")
	user2 := createUserTestUser(t, "bob", "Bob")

	ctrl := UserController{}
	m := &hbtp.Map{}
	m.Set("id", user2.Id)
	m.Set("act", "1")

	c, w := makeUserGinCtx(t, m, user1)
	ctrl.active(c, m)

	if w.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusMethodNotAllowed)
	}
}

func TestUserController_perm_success(t *testing.T) {
	setupUserTestDB(t)
	admin := createUserTestUser(t, "admin", "Admin")
	user1 := createUserTestUser(t, "alice", "Alice")

	uinfo := &model.TUserInfo{
		Id:       admin.Id,
		PermUser: 1,
	}
	if _, err := comm.Db.InsertOne(uinfo); err != nil {
		t.Fatalf("insert user info: %v", err)
	}

	ctrl := UserController{}
	m := &hbtp.Map{}
	m.Set("id", user1.Id)
	m.Set("permUser", true)
	m.Set("permOrg", true)
	m.Set("permPipe", true)

	c, w := makeUserGinCtx(t, m, admin)
	ctrl.perm(c, m)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, body: %s", w.Code, w.Body.String())
	}
}

func TestUserController_perm_noPermission(t *testing.T) {
	setupUserTestDB(t)
	user1 := createUserTestUser(t, "alice", "Alice")
	user2 := createUserTestUser(t, "bob", "Bob")

	ctrl := UserController{}
	m := &hbtp.Map{}
	m.Set("id", user2.Id)
	m.Set("permUser", true)

	c, w := makeUserGinCtx(t, m, user1)
	ctrl.perm(c, m)

	if w.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusMethodNotAllowed)
	}
}

func TestUserController_perm_paramError(t *testing.T) {
	setupUserTestDB(t)
	admin := createUserTestUser(t, "admin", "Admin")

	uinfo := &model.TUserInfo{
		Id:       admin.Id,
		PermUser: 1,
	}
	if _, err := comm.Db.InsertOne(uinfo); err != nil {
		t.Fatalf("insert user info: %v", err)
	}

	ctrl := UserController{}
	m := &hbtp.Map{}
	m.Set("id", "")

	c, w := makeUserGinCtx(t, m, admin)
	ctrl.perm(c, m)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}
