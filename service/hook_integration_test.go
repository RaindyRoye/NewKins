package service

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gokins/gokins/comm"
	"github.com/gokins/gokins/model"
	_ "github.com/mattn/go-sqlite3"
	"xorm.io/xorm"
)

// setupHookTestDB creates an in-memory SQLite database with all tables needed for hook tests.
func setupHookTestDB(t *testing.T) {
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

	// Sync all tables needed for hook tests
	err = eng.Sync2(
		&model.TUser{},
		&model.TUserOrg{},
		&model.TOrg{},
		&model.TOrgPipe{},
		&model.TPipeline{},
		&model.TPipelineConf{},
		&model.TPipelineVersion{},
		&model.TPipelineVar{},
		&model.TOrgVar{},
		&model.TTrigger{},
		&model.TTriggerRun{},
		&model.TBuild{},
		&model.TStage{},
		&model.TStep{},
	)
	if err != nil {
		t.Fatalf("failed to sync schema: %v", err)
	}
}

// TestTriggerPermCtx_Integ_WithDB tests TriggerPermCtx with a real database.
func TestTriggerPermCtx_Integ_WithDB(t *testing.T) {
	setupHookTestDB(t)
	ctx := context.Background()

	// Create admin user
	adminUser := &model.TUser{
		Id:      "admin",
		Aid:     1,
		Name:    "admin",
		Nick:    "Admin",
		Created: time.Now(),
	}
	if _, err := comm.Db.Insert(adminUser); err != nil {
		t.Fatalf("insert admin user: %v", err)
	}

	// Create regular user
	regularUser := &model.TUser{
		Id:      "user1",
		Aid:     2,
		Name:    "alice",
		Nick:    "Alice",
		Created: time.Now(),
	}
	if _, err := comm.Db.Insert(regularUser); err != nil {
		t.Fatalf("insert regular user: %v", err)
	}

	// Create pipeline
	pipeline := &model.TPipeline{
		Id:          "pipe-1",
		Uid:         "user1",
		Name:        "test-pipeline",
		DisplayName: "Test Pipeline",
		Deleted:     0,
	}
	if _, err := comm.Db.Insert(pipeline); err != nil {
		t.Fatalf("insert pipeline: %v", err)
	}

	// Test 1: Admin user can always trigger
	trigger := &model.TTrigger{
		Id:         "trigger-1",
		Aid:        1,
		Uid:        "admin",
		PipelineId: "pipe-1",
		Name:       "admin-trigger",
		Enabled:    1,
	}
	err := TriggerPermCtx(ctx, trigger)
	if err != nil {
		t.Errorf("admin should be able to trigger, got error: %v", err)
	}

	// Test 2: Pipeline owner can trigger
	trigger2 := &model.TTrigger{
		Id:         "trigger-2",
		Aid:        2,
		Uid:        "user1",
		PipelineId: "pipe-1",
		Name:       "owner-trigger",
		Enabled:    1,
	}
	err = TriggerPermCtx(ctx, trigger2)
	if err != nil {
		t.Errorf("pipeline owner should be able to trigger, got error: %v", err)
	}

	// Test 3: Non-owner without permission cannot trigger
	nonOwner := &model.TUser{
		Id:      "user2",
		Aid:     3,
		Name:    "bob",
		Nick:    "Bob",
		Created: time.Now(),
	}
	if _, err := comm.Db.Insert(nonOwner); err != nil {
		t.Fatalf("insert non-owner user: %v", err)
	}

	trigger3 := &model.TTrigger{
		Id:         "trigger-3",
		Aid:        3,
		Uid:        "user2",
		PipelineId: "pipe-1",
		Name:       "nonowner-trigger",
		Enabled:    1,
	}
	err = TriggerPermCtx(ctx, trigger3)
	if err == nil {
		t.Error("non-owner without permission should not be able to trigger")
	}
	if !errors.Is(err, ErrPermissionDenied) {
		t.Errorf("expected ErrPermissionDenied, got: %v", err)
	}

	// Test 4: Non-existent pipeline returns ErrPipelineNotFound
	trigger4 := &model.TTrigger{
		Id:         "trigger-4",
		Aid:        4,
		Uid:        "user1",
		PipelineId: "nonexistent-pipe",
		Name:       "bad-pipe-trigger",
		Enabled:    1,
	}
	err = TriggerPermCtx(ctx, trigger4)
	if err == nil {
		t.Error("trigger with nonexistent pipeline should fail")
	}
	if !errors.Is(err, ErrPipelineNotFound) {
		t.Errorf("expected ErrPipelineNotFound, got: %v", err)
	}
}

// TestTriggerWeb_Integ_NoParams tests TriggerWeb with empty params.
func TestTriggerWeb_Integ_NoParams(t *testing.T) {
	setupHookTestDB(t)
	ctx := context.Background()

	trigger := &model.TTrigger{
		Id:         "trigger-no-params",
		Aid:        1,
		Uid:        "user1",
		PipelineId: "pipe-1",
		Name:       "no-params",
		Params:     "",
		Enabled:    1,
	}

	_, err := TriggerWeb(ctx, trigger, "secret")
	if err == nil {
		t.Error("TriggerWeb should fail with empty params")
	}
	if !errors.Is(err, ErrTriggerNoParams) {
		t.Errorf("expected ErrTriggerNoParams, got: %v", err)
	}
}

// TestTriggerWeb_Integ_InvalidParams tests TriggerWeb with invalid JSON params.
func TestTriggerWeb_Integ_InvalidParams(t *testing.T) {
	setupHookTestDB(t)
	ctx := context.Background()

	// Create admin user and pipeline for permission check
	adminUser := &model.TUser{
		Id:      "admin",
		Aid:     1,
		Name:    "admin",
		Nick:    "Admin",
		Created: time.Now(),
	}
	if _, err := comm.Db.Insert(adminUser); err != nil {
		t.Fatalf("insert admin user: %v", err)
	}

	pipeline := &model.TPipeline{
		Id:      "pipe-1",
		Uid:     "admin",
		Name:    "test-pipeline",
		Deleted: 0,
	}
	if _, err := comm.Db.Insert(pipeline); err != nil {
		t.Fatalf("insert pipeline: %v", err)
	}

	trigger := &model.TTrigger{
		Id:         "trigger-invalid-json",
		Aid:        1,
		Uid:        "admin",
		PipelineId: "pipe-1",
		Name:       "invalid-json",
		Params:     "{invalid json}",
		Enabled:    1,
	}

	_, err := TriggerWeb(ctx, trigger, "secret")
	if err == nil {
		t.Error("TriggerWeb should fail with invalid JSON params")
	}
	// Should be a wrapped error containing "unmarshal"
	if !strings.Contains(err.Error(), "unmarshal") {
		t.Errorf("error should contain 'unmarshal', got: %v", err)
	}
}

// TestTriggerWeb_Integ_NoSecret tests TriggerWeb when trigger params have no secret.
func TestTriggerWeb_Integ_NoSecret(t *testing.T) {
	setupHookTestDB(t)
	ctx := context.Background()

	// Create admin user and pipeline
	adminUser := &model.TUser{
		Id:      "admin",
		Aid:     1,
		Name:    "admin",
		Nick:    "Admin",
		Created: time.Now(),
	}
	if _, err := comm.Db.Insert(adminUser); err != nil {
		t.Fatalf("insert admin user: %v", err)
	}

	pipeline := &model.TPipeline{
		Id:      "pipe-1",
		Uid:     "admin",
		Name:    "test-pipeline",
		Deleted: 0,
	}
	if _, err := comm.Db.Insert(pipeline); err != nil {
		t.Fatalf("insert pipeline: %v", err)
	}

	params := map[string]string{
		"branch": "main",
	}
	paramsJSON, _ := json.Marshal(params)

	trigger := &model.TTrigger{
		Id:         "trigger-no-secret",
		Aid:        1,
		Uid:        "admin",
		PipelineId: "pipe-1",
		Name:       "no-secret",
		Params:     string(paramsJSON),
		Enabled:    1,
	}

	_, err := TriggerWeb(ctx, trigger, "my-secret")
	if err == nil {
		t.Error("TriggerWeb should fail when trigger has no secret configured")
	}
	if !errors.Is(err, ErrTriggerNoSecret) {
		t.Errorf("expected ErrTriggerNoSecret, got: %v", err)
	}
}

// TestTriggerWeb_Integ_SecretMismatch tests TriggerWeb with mismatched secret.
func TestTriggerWeb_Integ_SecretMismatch(t *testing.T) {
	setupHookTestDB(t)
	ctx := context.Background()

	// Create admin user and pipeline
	adminUser := &model.TUser{
		Id:      "admin",
		Aid:     1,
		Name:    "admin",
		Nick:    "Admin",
		Created: time.Now(),
	}
	if _, err := comm.Db.Insert(adminUser); err != nil {
		t.Fatalf("insert admin user: %v", err)
	}

	pipeline := &model.TPipeline{
		Id:      "pipe-1",
		Uid:     "admin",
		Name:    "test-pipeline",
		Deleted: 0,
	}
	if _, err := comm.Db.Insert(pipeline); err != nil {
		t.Fatalf("insert pipeline: %v", err)
	}

	params := map[string]string{
		"secret": "correct-secret",
		"branch": "main",
	}
	paramsJSON, _ := json.Marshal(params)

	trigger := &model.TTrigger{
		Id:         "trigger-secret-mismatch",
		Aid:        1,
		Uid:        "admin",
		PipelineId: "pipe-1",
		Name:       "secret-mismatch",
		Params:     string(paramsJSON),
		Enabled:    1,
	}

	_, err := TriggerWeb(ctx, trigger, "wrong-secret")
	if err == nil {
		t.Error("TriggerWeb should fail with mismatched secret")
	}
	if !errors.Is(err, ErrTriggerSecretMismatch) {
		t.Errorf("expected ErrTriggerSecretMismatch, got: %v", err)
	}
}

// TestTriggerTimer_Integ_NoPermission tests TriggerTimer without proper permissions.
func TestTriggerTimer_Integ_NoPermission(t *testing.T) {
	setupHookTestDB(t)
	ctx := context.Background()

	// Create non-admin user
	regularUser := &model.TUser{
		Id:      "user1",
		Aid:     1,
		Name:    "alice",
		Nick:    "Alice",
		Created: time.Now(),
	}
	if _, err := comm.Db.Insert(regularUser); err != nil {
		t.Fatalf("insert user: %v", err)
	}

	// Create pipeline owned by another user
	pipeline := &model.TPipeline{
		Id:      "pipe-1",
		Uid:     "other-user",
		Name:    "test-pipeline",
		Deleted: 0,
	}
	if _, err := comm.Db.Insert(pipeline); err != nil {
		t.Fatalf("insert pipeline: %v", err)
	}

	trigger := &model.TTrigger{
		Id:         "trigger-timer-no-perm",
		Aid:        1,
		Uid:        "user1",
		PipelineId: "pipe-1",
		Name:       "timer-no-perm",
		Params:     "{}",
		Enabled:    1,
		Created:    time.Now(),
	}

	_, err := TriggerTimer(ctx, trigger)
	if err == nil {
		t.Error("TriggerTimer should fail without proper permissions")
	}
	// Should be a permission error
	if !errors.Is(err, ErrPermissionDenied) {
		t.Errorf("expected ErrPermissionDenied, got: %v", err)
	}
}

// TestTriggerHook_Integ_NoParams tests TriggerHook with empty params.
func TestTriggerHook_Integ_NoParams(t *testing.T) {
	setupHookTestDB(t)

	trigger := &model.TTrigger{
		Id:         "trigger-hook-no-params",
		Aid:        1,
		Uid:        "user1",
		PipelineId: "pipe-1",
		Name:       "hook-no-params",
		Params:     "",
		Enabled:    1,
	}

	req := httptest.NewRequest(http.MethodPost, "/hook", strings.NewReader("{}"))
	_, err := TriggerHook(trigger, req)
	if err == nil {
		t.Error("TriggerHook should fail with empty params")
	}
	if !errors.Is(err, ErrTriggerNoParams) {
		t.Errorf("expected ErrTriggerNoParams, got: %v", err)
	}
}

// TestTriggerHookCtx_Integ_InvalidParams tests TriggerHookCtx with invalid JSON.
func TestTriggerHookCtx_Integ_InvalidParams(t *testing.T) {
	setupHookTestDB(t)
	ctx := context.Background()

	// Create admin user and pipeline
	adminUser := &model.TUser{
		Id:      "admin",
		Aid:     1,
		Name:    "admin",
		Nick:    "Admin",
		Created: time.Now(),
	}
	if _, err := comm.Db.Insert(adminUser); err != nil {
		t.Fatalf("insert admin user: %v", err)
	}

	pipeline := &model.TPipeline{
		Id:      "pipe-1",
		Uid:     "admin",
		Name:    "test-pipeline",
		Deleted: 0,
	}
	if _, err := comm.Db.Insert(pipeline); err != nil {
		t.Fatalf("insert pipeline: %v", err)
	}

	trigger := &model.TTrigger{
		Id:         "trigger-hook-invalid",
		Aid:        1,
		Uid:        "admin",
		PipelineId: "pipe-1",
		Name:       "hook-invalid",
		Params:     "{invalid json}",
		Enabled:    1,
	}

	req := httptest.NewRequest(http.MethodPost, "/hook", strings.NewReader("{}"))
	_, err := TriggerHookCtx(ctx, trigger, req)
	if err == nil {
		t.Error("TriggerHookCtx should fail with invalid JSON params")
	}
	if !strings.Contains(err.Error(), "unmarshal") {
		t.Errorf("error should contain 'unmarshal', got: %v", err)
	}
}

// TestTriggerHookCtx_Integ_NoHookType tests TriggerHookCtx when hookType is missing.
func TestTriggerHookCtx_Integ_NoHookType(t *testing.T) {
	setupHookTestDB(t)
	ctx := context.Background()

	// Create admin user and pipeline
	adminUser := &model.TUser{
		Id:      "admin",
		Aid:     1,
		Name:    "admin",
		Nick:    "Admin",
		Created: time.Now(),
	}
	if _, err := comm.Db.Insert(adminUser); err != nil {
		t.Fatalf("insert admin user: %v", err)
	}

	pipeline := &model.TPipeline{
		Id:      "pipe-1",
		Uid:     "admin",
		Name:    "test-pipeline",
		Deleted: 0,
	}
	if _, err := comm.Db.Insert(pipeline); err != nil {
		t.Fatalf("insert pipeline: %v", err)
	}

	params := map[string]string{
		"secret": "my-secret",
		"event":  "push",
	}
	paramsJSON, _ := json.Marshal(params)

	trigger := &model.TTrigger{
		Id:         "trigger-hook-no-type",
		Aid:        1,
		Uid:        "admin",
		PipelineId: "pipe-1",
		Name:       "hook-no-type",
		Params:     string(paramsJSON),
		Enabled:    1,
	}

	req := httptest.NewRequest(http.MethodPost, "/hook", strings.NewReader("{}"))
	_, err := TriggerHookCtx(ctx, trigger, req)
	if err == nil {
		t.Error("TriggerHookCtx should fail when hookType is missing")
	}
	if !errors.Is(err, ErrHookTypeEmpty) {
		t.Errorf("expected ErrHookTypeEmpty, got: %v", err)
	}
}

// TestTriggerHookCtx_Integ_UnknownHookType tests TriggerHookCtx with unknown hook type.
func TestTriggerHookCtx_Integ_UnknownHookType(t *testing.T) {
	setupHookTestDB(t)
	ctx := context.Background()

	// Create admin user and pipeline
	adminUser := &model.TUser{
		Id:      "admin",
		Aid:     1,
		Name:    "admin",
		Nick:    "Admin",
		Created: time.Now(),
	}
	if _, err := comm.Db.Insert(adminUser); err != nil {
		t.Fatalf("insert admin user: %v", err)
	}

	pipeline := &model.TPipeline{
		Id:      "pipe-1",
		Uid:     "admin",
		Name:    "test-pipeline",
		Deleted: 0,
	}
	if _, err := comm.Db.Insert(pipeline); err != nil {
		t.Fatalf("insert pipeline: %v", err)
	}

	params := map[string]string{
		"hookType": "bitbucket",
		"secret":   "my-secret",
		"event":    "push",
	}
	paramsJSON, _ := json.Marshal(params)

	trigger := &model.TTrigger{
		Id:         "trigger-hook-unknown-type",
		Aid:        1,
		Uid:        "admin",
		PipelineId: "pipe-1",
		Name:       "hook-unknown-type",
		Params:     string(paramsJSON),
		Enabled:    1,
	}

	req := httptest.NewRequest(http.MethodPost, "/hook", strings.NewReader("{}"))
	_, err := TriggerHookCtx(ctx, trigger, req)
	if err == nil {
		t.Error("TriggerHookCtx should fail with unknown hook type")
	}
	if !errors.Is(err, ErrUnknownWebhookType) {
		t.Errorf("expected ErrUnknownWebhookType, got: %v", err)
	}
}

// TestTriggerHookCtx_Integ_NoPermission tests TriggerHookCtx without proper permissions.
func TestTriggerHookCtx_Integ_NoPermission(t *testing.T) {
	setupHookTestDB(t)
	ctx := context.Background()

	// Create non-admin user
	regularUser := &model.TUser{
		Id:      "user1",
		Aid:     1,
		Name:    "alice",
		Nick:    "Alice",
		Created: time.Now(),
	}
	if _, err := comm.Db.Insert(regularUser); err != nil {
		t.Fatalf("insert user: %v", err)
	}

	// Create pipeline owned by another user
	pipeline := &model.TPipeline{
		Id:      "pipe-1",
		Uid:     "other-user",
		Name:    "test-pipeline",
		Deleted: 0,
	}
	if _, err := comm.Db.Insert(pipeline); err != nil {
		t.Fatalf("insert pipeline: %v", err)
	}

	params := map[string]string{
		"hookType": "github",
		"secret":   "my-secret",
		"event":    "push",
	}
	paramsJSON, _ := json.Marshal(params)

	trigger := &model.TTrigger{
		Id:         "trigger-hook-no-perm",
		Aid:        1,
		Uid:        "user1",
		PipelineId: "pipe-1",
		Name:       "hook-no-perm",
		Params:     string(paramsJSON),
		Enabled:    1,
	}

	req := httptest.NewRequest(http.MethodPost, "/hook", strings.NewReader("{}"))
	_, err := TriggerHookCtx(ctx, trigger, req)
	if err == nil {
		t.Error("TriggerHookCtx should fail without proper permissions")
	}
	if !errors.Is(err, ErrPermissionDenied) {
		t.Errorf("expected ErrPermissionDenied, got: %v", err)
	}
}
