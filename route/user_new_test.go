package route

import (
	"bytes"
	"context"
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
	hbtp "github.com/mgr9525/HyperByte-Transfer-Protocol"
	_ "github.com/mattn/go-sqlite3"
	"xorm.io/xorm"
)

func setupUserNewTestDB(t *testing.T) *xorm.Engine {
	t.Helper()
	eng, err := xorm.NewEngine("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("failed to create test database: %v", err)
	}

	oldDb := comm.Db
	comm.Db = eng
	t.Cleanup(func() {
		comm.Db = oldDb
		_ = eng.Close()
	})

	// Create tables manually with proper autoincrement for SQLite
	// SQLite autoincrement only works on single-column INTEGER PRIMARY KEY
	_, err = eng.Exec(`
		CREATE TABLE t_user (
			id VARCHAR(64) NOT NULL,
			aid INTEGER PRIMARY KEY AUTOINCREMENT,
			name VARCHAR(100),
			pass VARCHAR(255),
			nick VARCHAR(100),
			avatar VARCHAR(500),
			created DATETIME,
			login_time DATETIME,
			active INT DEFAULT 0
		)
	`)
	if err != nil {
		t.Fatalf("failed to create t_user table: %v", err)
	}

	_, err = eng.Exec(`
		CREATE TABLE t_user_info (
			id VARCHAR(64) PRIMARY KEY NOT NULL,
			phone VARCHAR(100),
			email VARCHAR(200),
			birthday DATETIME,
			remark TEXT,
			perm_user INT DEFAULT 0,
			perm_org INT DEFAULT 0,
			perm_pipe INT DEFAULT 0
		)
	`)
	if err != nil {
		t.Fatalf("failed to create t_user_info table: %v", err)
	}

	_, err = eng.Exec(`
		CREATE TABLE t_user_org (
			id VARCHAR(64) NOT NULL,
			aid INTEGER PRIMARY KEY AUTOINCREMENT,
			uid VARCHAR(64),
			org_id VARCHAR(64),
			perm_adm INT DEFAULT 0,
			perm_rw INT DEFAULT 0,
			perm_exec INT DEFAULT 0,
			perm_down INT DEFAULT 0
		)
	`)
	if err != nil {
		t.Fatalf("failed to create t_user_org table: %v", err)
	}

	return eng
}

func TestUserNew_MissingParams(t *testing.T) {
	setupUserNewTestDB(t)
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name string
		body hbtp.Map
		want int
	}{
		{
			name: "missing name",
			body: hbtp.Map{"nick": "Test User", "pass": "password123"},
			want: http.StatusBadRequest,
		},
		{
			name: "missing nick",
			body: hbtp.Map{"name": "testuser", "pass": "password123"},
			want: http.StatusBadRequest,
		},
		{
			name: "missing pass",
			body: hbtp.Map{"name": "testuser", "nick": "Test User"},
			want: http.StatusBadRequest,
		},
		{
			name: "empty name",
			body: hbtp.Map{"name": "   ", "nick": "Test User", "pass": "password123"},
			want: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)

			bodyBytes, _ := json.Marshal(tt.body)
			c.Request = httptest.NewRequest("POST", "/api/user/new", bytes.NewReader(bodyBytes))
			c.Request.Header.Set("Content-Type", "application/json")

			// Set a logged-in admin user
			adminUser := &model.TUser{Id: "admin", Name: "admin"}
			c.Set(service.LgUserKey, adminUser)

			ctrl := UserController{}
			m := tt.body
			ctrl.new(c, &m)

			if w.Code != tt.want {
				t.Errorf("got status %d, want %d", w.Code, tt.want)
			}
		})
	}
}

func TestUserNew_NoPermission(t *testing.T) {
	eng := setupUserNewTestDB(t)
	gin.SetMode(gin.TestMode)

	// Create a non-admin user without perm_user
	regularUser := &model.TUser{
		Id:        "user1",
		Aid:       1,
		Name:      "regular",
		Nick:      "Regular User",
		Created:   time.Now(),
		LoginTime: time.Now(),
		Active:    1,
	}
	if _, err := eng.Insert(regularUser); err != nil {
		t.Fatalf("failed to insert regular user: %v", err)
	}

	// Create user info without perm_user
	userInfo := &model.TUserInfo{
		Id:       "user1",
		PermUser: 0, // No permission to create users
	}
	if _, err := eng.Insert(userInfo); err != nil {
		t.Fatalf("failed to insert user info: %v", err)
	}

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	body := hbtp.Map{
		"name": "newuser",
		"nick": "New User",
		"pass": "password123",
	}
	bodyBytes, _ := json.Marshal(body)
	c.Request = httptest.NewRequest("POST", "/api/user/new", bytes.NewReader(bodyBytes))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Request = c.Request.WithContext(context.Background())
	c.Set(service.LgUserKey, regularUser)

	ctrl := UserController{}
	ctrl.new(c, &body)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("got status %d, want %d", w.Code, http.StatusMethodNotAllowed)
	}
}

func TestUserNew_DuplicateName(t *testing.T) {
	eng := setupUserNewTestDB(t)
	gin.SetMode(gin.TestMode)

	// Create an existing user
	existingUser := &model.TUser{
		Id:        "existing1",
		Aid:       1,
		Name:      "existinguser",
		Nick:      "Existing User",
		Created:   time.Now(),
		LoginTime: time.Now(),
		Active:    1,
	}
	if _, err := eng.Insert(existingUser); err != nil {
		t.Fatalf("failed to insert existing user: %v", err)
	}

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	body := hbtp.Map{
		"name": "existinguser",
		"nick": "New User",
		"pass": "password123",
	}
	bodyBytes, _ := json.Marshal(body)
	c.Request = httptest.NewRequest("POST", "/api/user/new", bytes.NewReader(bodyBytes))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Request = c.Request.WithContext(context.Background())

	// Set admin user
	adminUser := &model.TUser{Id: "admin", Name: "admin"}
	c.Set(service.LgUserKey, adminUser)

	ctrl := UserController{}
	ctrl.new(c, &body)

	if w.Code != http.StatusConflict {
		t.Errorf("got status %d, want %d", w.Code, http.StatusConflict)
	}
}

func TestUserNew_Success(t *testing.T) {
	eng := setupUserNewTestDB(t)
	gin.SetMode(gin.TestMode)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	body := hbtp.Map{
		"name": "newuser",
		"nick": "New User",
		"pass": "password123",
	}
	bodyBytes, _ := json.Marshal(body)
	c.Request = httptest.NewRequest("POST", "/api/user/new", bytes.NewReader(bodyBytes))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Request = c.Request.WithContext(context.Background())

	// Set admin user
	adminUser := &model.TUser{Id: "admin", Name: "admin"}
	c.Set(service.LgUserKey, adminUser)

	ctrl := UserController{}
	ctrl.new(c, &body)

	if w.Code != http.StatusOK {
		t.Errorf("got status %d, want %d", w.Code, http.StatusOK)
	}

	// Verify user was created
	var count int64
	count, err := eng.Where("name = ?", "newuser").Count(&model.TUser{})
	if err != nil {
		t.Fatalf("failed to count users: %v", err)
	}
	if count != 1 {
		t.Errorf("expected 1 user with name 'newuser', got %d", count)
	}
}

func TestUserNew_WithPermUser(t *testing.T) {
	eng := setupUserNewTestDB(t)
	gin.SetMode(gin.TestMode)

	// Create a non-admin user WITH perm_user
	permUser := &model.TUser{
		Id:        "user1",
		Aid:       1,
		Name:      "permuser",
		Nick:      "Perm User",
		Created:   time.Now(),
		LoginTime: time.Now(),
		Active:    1,
	}
	if _, err := eng.Insert(permUser); err != nil {
		t.Fatalf("failed to insert perm user: %v", err)
	}

	// Create user info WITH perm_user
	userInfo := &model.TUserInfo{
		Id:       "user1",
		PermUser: 1, // Has permission to create users
	}
	if _, err := eng.Insert(userInfo); err != nil {
		t.Fatalf("failed to insert user info: %v", err)
	}

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	body := hbtp.Map{
		"name": "newuser2",
		"nick": "New User 2",
		"pass": "password123",
	}
	bodyBytes, _ := json.Marshal(body)
	c.Request = httptest.NewRequest("POST", "/api/user/new", bytes.NewReader(bodyBytes))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Request = c.Request.WithContext(context.Background())
	c.Set(service.LgUserKey, permUser)

	ctrl := UserController{}
	ctrl.new(c, &body)

	if w.Code != http.StatusOK {
		t.Errorf("got status %d, want %d", w.Code, http.StatusOK)
	}
}

func TestUserNew_TrimWhitespace(t *testing.T) {
	eng := setupUserNewTestDB(t)
	gin.SetMode(gin.TestMode)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	body := hbtp.Map{
		"name": "  trimmeduser  ",
		"nick": "  Trimmed User  ",
		"pass": "password123",
	}
	bodyBytes, _ := json.Marshal(body)
	c.Request = httptest.NewRequest("POST", "/api/user/new", bytes.NewReader(bodyBytes))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Request = c.Request.WithContext(context.Background())

	adminUser := &model.TUser{Id: "admin", Name: "admin"}
	c.Set(service.LgUserKey, adminUser)

	ctrl := UserController{}
	ctrl.new(c, &body)

	if w.Code != http.StatusOK {
		t.Errorf("got status %d, want %d", w.Code, http.StatusOK)
	}

	// Verify user was created with trimmed name
	user := &model.TUser{}
	ok, err := eng.Where("name = ?", "trimmeduser").Get(user)
	if err != nil {
		t.Fatalf("failed to query user: %v", err)
	}
	if !ok {
		t.Error("user with trimmed name not found")
	}
	if user.Nick != "Trimmed User" {
		t.Errorf("nick = %q, want 'Trimmed User'", user.Nick)
	}
}

func TestUserNew_PasswordHashed(t *testing.T) {
	eng := setupUserNewTestDB(t)
	gin.SetMode(gin.TestMode)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	body := hbtp.Map{
		"name": "hashuser",
		"nick": "Hash User",
		"pass": "mypassword",
	}
	bodyBytes, _ := json.Marshal(body)
	c.Request = httptest.NewRequest("POST", "/api/user/new", bytes.NewReader(bodyBytes))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Request = c.Request.WithContext(context.Background())

	adminUser := &model.TUser{Id: "admin", Name: "admin"}
	c.Set(service.LgUserKey, adminUser)

	ctrl := UserController{}
	ctrl.new(c, &body)

	if w.Code != http.StatusOK {
		t.Errorf("got status %d, want %d", w.Code, http.StatusOK)
	}

	// Verify password was hashed
	user := &model.TUser{}
	ok, err := eng.Where("name = ?", "hashuser").Get(user)
	if err != nil {
		t.Fatalf("failed to query user: %v", err)
	}
	if !ok {
		t.Fatal("user not found")
	}

	expectedHash := utils.Md5String("mypassword")
	if user.Pass != expectedHash {
		t.Errorf("password not hashed correctly: got %q, want %q", user.Pass, expectedHash)
	}
}
