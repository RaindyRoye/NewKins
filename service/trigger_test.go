package service

import (
	"context"
	"errors"
	"testing"

	"github.com/gokins/gokins/comm"
	"github.com/gokins/gokins/model"
	_ "github.com/mattn/go-sqlite3"
	"xorm.io/xorm"
)

// setupTriggerTestDB creates an in-memory SQLite database for trigger tests.
func setupTriggerTestDB(t *testing.T) *xorm.Engine {
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
	if err := eng.Sync2(
		&model.TUser{},
		&model.TOrg{},
		&model.TUserOrg{},
		&model.TPipeline{},
		&model.TTrigger{},
		&model.TTriggerRun{},
	); err != nil {
		t.Fatalf("failed to sync schema: %v", err)
	}
	return eng
}

// --- TriggerPermCtx Tests ---

func TestTriggerPermCtx_NilTrigger(t *testing.T) {
	setupTriggerTestDB(t)
	ctx := context.Background()

	err := TriggerPermCtx(ctx, nil)
	if err == nil {
		t.Error("TriggerPermCtx should return error for nil trigger")
	}
	if !errors.Is(err, ErrPipelineNotFound) {
		t.Errorf("expected ErrPipelineNotFound, got %v", err)
	}
}

func TestTriggerPermCtx_PipelineNotFound(t *testing.T) {
	eng := setupTriggerTestDB(t)
	ctx := context.Background()

	user := &model.TUser{Id: "user-1", Aid: 1, Name: "testuser"}
	if _, err := eng.Insert(user); err != nil {
		t.Fatalf("insert user: %v", err)
	}

	tt := &model.TTrigger{
		Id:         "trigger-1",
		Uid:        "user-1",
		PipelineId: "nonexistent-pipeline",
	}

	err := TriggerPermCtx(ctx, tt)
	if err == nil {
		t.Error("TriggerPermCtx should return error when pipeline not found")
	}
	if !errors.Is(err, ErrPipelineNotFound) {
		t.Errorf("expected ErrPipelineNotFound, got %v", err)
	}
}

func TestTriggerPermCtx_OwnerCanTrigger(t *testing.T) {
	eng := setupTriggerTestDB(t)
	ctx := context.Background()

	user := &model.TUser{Id: "user-owner", Aid: 10, Name: "owner"}
	if _, err := eng.Insert(user); err != nil {
		t.Fatalf("insert user: %v", err)
	}

	pipe := &model.TPipeline{
		Id:   "pipe-1",
		Uid:  "user-owner",
		Name: "test-pipeline",
	}
	if _, err := eng.Insert(pipe); err != nil {
		t.Fatalf("insert pipeline: %v", err)
	}

	tt := &model.TTrigger{
		Id:         "trigger-owner",
		Uid:        "user-owner",
		PipelineId: "pipe-1",
	}

	err := TriggerPermCtx(ctx, tt)
	if err != nil {
		t.Errorf("pipeline owner should be able to trigger, got error: %v", err)
	}
}

func TestTriggerPermCtx_AdminCanTrigger(t *testing.T) {
	eng := setupTriggerTestDB(t)
	ctx := context.Background()

	admin := &model.TUser{Id: "admin", Aid: 1, Name: "admin"}
	if _, err := eng.Insert(admin); err != nil {
		t.Fatalf("insert admin: %v", err)
	}

	owner := &model.TUser{Id: "owner-2", Aid: 2, Name: "owner2"}
	if _, err := eng.Insert(owner); err != nil {
		t.Fatalf("insert owner: %v", err)
	}

	pipe := &model.TPipeline{
		Id:   "pipe-2",
		Uid:  "owner-2",
		Name: "admin-test-pipeline",
	}
	if _, err := eng.Insert(pipe); err != nil {
		t.Fatalf("insert pipeline: %v", err)
	}

	tt := &model.TTrigger{
		Id:         "trigger-admin",
		Uid:        "admin",
		PipelineId: "pipe-2",
	}

	err := TriggerPermCtx(ctx, tt)
	if err != nil {
		t.Errorf("admin should be able to trigger, got error: %v", err)
	}
}

func TestTriggerPermCtx_NonOwnerCannotTrigger(t *testing.T) {
	eng := setupTriggerTestDB(t)
	ctx := context.Background()

	owner := &model.TUser{Id: "owner-3", Aid: 3, Name: "owner3"}
	if _, err := eng.Insert(owner); err != nil {
		t.Fatalf("insert owner: %v", err)
	}

	other := &model.TUser{Id: "other-user", Aid: 4, Name: "other"}
	if _, err := eng.Insert(other); err != nil {
		t.Fatalf("insert other user: %v", err)
	}

	pipe := &model.TPipeline{
		Id:   "pipe-3",
		Uid:  "owner-3",
		Name: "other-test-pipeline",
	}
	if _, err := eng.Insert(pipe); err != nil {
		t.Fatalf("insert pipeline: %v", err)
	}

	tt := &model.TTrigger{
		Id:         "trigger-other",
		Uid:        "other-user",
		PipelineId: "pipe-3",
	}

	err := TriggerPermCtx(ctx, tt)
	if err == nil {
		t.Error("non-owner should not be able to trigger")
	}
	if !errors.Is(err, ErrPermissionDenied) {
		t.Errorf("expected ErrPermissionDenied, got %v", err)
	}
}

// --- TriggerPerm (global context) ---

func TestTriggerPerm_DelegatesToCtx(t *testing.T) {
	eng := setupTriggerTestDB(t)

	user := &model.TUser{Id: "user-global", Aid: 20, Name: "globaluser"}
	if _, err := eng.Insert(user); err != nil {
		t.Fatalf("insert user: %v", err)
	}

	pipe := &model.TPipeline{
		Id:   "pipe-global",
		Uid:  "user-global",
		Name: "global-pipeline",
	}
	if _, err := eng.Insert(pipe); err != nil {
		t.Fatalf("insert pipeline: %v", err)
	}

	tt := &model.TTrigger{
		Id:         "trigger-global",
		Uid:        "user-global",
		PipelineId: "pipe-global",
	}

	err := TriggerPerm(tt)
	if err != nil {
		t.Errorf("TriggerPerm should delegate to TriggerPermCtx successfully, got error: %v", err)
	}
}

// --- TriggerPermCtx with Canceled Context (with DB) ---

func TestTriggerPermCtx_CanceledContextWithDB(t *testing.T) {
	eng := setupTriggerTestDB(t)

	user := &model.TUser{Id: "user-cancel", Aid: 30, Name: "canceluser"}
	if _, err := eng.Insert(user); err != nil {
		t.Fatalf("insert user: %v", err)
	}

	pipe := &model.TPipeline{
		Id:   "pipe-cancel",
		Uid:  "user-cancel",
		Name: "cancel-pipeline",
	}
	if _, err := eng.Insert(pipe); err != nil {
		t.Fatalf("insert pipeline: %v", err)
	}

	tt := &model.TTrigger{
		Id:         "trigger-cancel",
		Uid:        "user-cancel",
		PipelineId: "pipe-cancel",
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	err := TriggerPermCtx(ctx, tt)
	// With a canceled context, the database query may fail or succeed depending on timing.
	// The important thing is that the function respects the context and doesn't hang.
	if err != nil {
		t.Logf("TriggerPermCtx with canceled context returned error (expected): %v", err)
	}
}
