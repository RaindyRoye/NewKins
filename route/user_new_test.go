package route

import (
	"context"
	"net/http"
	"testing"

	"github.com/gokins/gokins/comm"
	"github.com/gokins/gokins/model"
	"github.com/gokins/gokins/service"
	hbtp "github.com/mgr9525/HyperByte-Transfer-Protocol"
)

// --- UserController.new tests ---

func TestUserNew_EmptyName(t *testing.T) {
	setupPipelineTestDB(t)
	ctrl := UserController{}
	m := &hbtp.Map{}
	m.Set("name", "")
	m.Set("nick", "Test User")
	m.Set("pass", "password123")
	c, w := makePipelineTestContext(t, m)
	ctrl.new(c, m)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for empty name, got %d: %s", w.Code, w.Body.String())
	}
}

func TestUserNew_EmptyNick(t *testing.T) {
	setupPipelineTestDB(t)
	ctrl := UserController{}
	m := &hbtp.Map{}
	m.Set("name", "testuser")
	m.Set("nick", "")
	m.Set("pass", "password123")
	c, w := makePipelineTestContext(t, m)
	ctrl.new(c, m)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for empty nick, got %d: %s", w.Code, w.Body.String())
	}
}

func TestUserNew_EmptyPass(t *testing.T) {
	setupPipelineTestDB(t)
	ctrl := UserController{}
	m := &hbtp.Map{}
	m.Set("name", "testuser")
	m.Set("nick", "Test User")
	m.Set("pass", "")
	c, w := makePipelineTestContext(t, m)
	ctrl.new(c, m)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for empty pass, got %d: %s", w.Code, w.Body.String())
	}
}

func TestUserNew_DuplicateUser(t *testing.T) {
	setupPipelineTestDB(t)

	// Insert an existing user
	_, err := comm.Db.Exec(`INSERT INTO t_user (id, name, nick, pass, active) 
		VALUES ('existing-user', 'existing', 'Existing User', 'hashed', 1)`)
	if err != nil {
		t.Fatalf("insert user: %v", err)
	}

	ctrl := UserController{}
	m := &hbtp.Map{}
	m.Set("name", "existing")
	m.Set("nick", "New User")
	m.Set("pass", "password123")

	// Create context with admin user
	c, w := makePipelineTestContext(t, m)
	c.Set(service.LgUserKey, &model.TUser{
		Id:     "admin",
		Name:   "admin",
		Nick:   "Admin User",
		Active: 1,
	})

	ctrl.new(c, m)

	if w.Code != http.StatusConflict {
		t.Errorf("expected 409 for duplicate user, got %d: %s", w.Code, w.Body.String())
	}
}

func TestUserNew_Success(t *testing.T) {
	setupPipelineTestDB(t)
	ctrl := UserController{}
	m := &hbtp.Map{}
	m.Set("name", "newuser")
	m.Set("nick", "New User")
	m.Set("pass", "password123")

	// Create context with admin user
	c, w := makePipelineTestContext(t, m)
	c.Set(service.LgUserKey, &model.TUser{
		Id:     "admin",
		Name:   "admin",
		Nick:   "Admin User",
		Active: 1,
	})

	ctrl.new(c, m)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200 for successful user creation, got %d: %s", w.Code, w.Body.String())
	}

	// Verify user was created
	var user model.TUser
	has, err := comm.Db.Where("name = ?", "newuser").Get(&user)
	if err != nil || !has {
		t.Fatalf("query user after creation: %v, has=%v", err, has)
	}
	if user.Nick != "New User" {
		t.Errorf("expected nick 'New User', got '%s'", user.Nick)
	}
}

// --- batchRunPipelineVersions tests (package-level function) ---

func TestBatchRunPipelineVersions_EmptyIds(t *testing.T) {
	setupPipelineTestDB(t)

	result, err := batchRunPipelineVersions(context.Background(), []string{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) != 0 {
		t.Errorf("expected empty map, got %d entries", len(result))
	}
}

func TestBatchRunPipelineVersions_NonexistentIds(t *testing.T) {
	setupPipelineTestDB(t)

	result, err := batchRunPipelineVersions(context.Background(), []string{"nonexistent-id"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) != 0 {
		t.Errorf("expected empty map for nonexistent ids, got %d entries", len(result))
	}
}

func TestBatchRunPipelineVersions_WithRealData(t *testing.T) {
	setupPipelineTestDB(t)

	// Insert test pipeline versions
	for _, id := range []string{"pv-1", "pv-2"} {
		_, err := comm.Db.Exec(`INSERT INTO t_pipeline_version (id, pipeline_id, number, deleted) 
			VALUES (?, 'pipe-1', 1, 0)`, id)
		if err != nil {
			t.Fatalf("insert pipeline_version: %v", err)
		}
	}

	result, err := batchRunPipelineVersions(context.Background(), []string{"pv-1", "pv-2"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) != 2 {
		t.Errorf("expected 2 entries, got %d", len(result))
	}
}
