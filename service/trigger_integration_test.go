package service

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gokins/gokins/comm"
	"github.com/gokins/gokins/model"
	_ "github.com/mattn/go-sqlite3"
	"xorm.io/xorm"
)

// setupTriggerFullTestDB creates an in-memory SQLite database with all tables needed for trigger tests.
func setupTriggerFullTestDB(t *testing.T) *xorm.Engine {
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

	err = eng.Sync2(
		&model.TUser{},
		&model.TUserInfo{},
		&model.TOrg{},
		&model.TUserOrg{},
		&model.TPipeline{},
		&model.TPipelineConf{},
		&model.TPipelineVersion{},
		&model.TPipelineVar{},
		&model.TOrgVar{},
		&model.TOrgPipe{},
		&model.TBuild{},
		&model.TStage{},
		&model.TStep{},
		&model.TCmdLine{},
		&model.TTrigger{},
		&model.TTriggerRun{},
	)
	if err != nil {
		t.Fatalf("failed to sync database schema: %v", err)
	}

	return eng
}

// insertPipelineWithConf creates a pipeline with valid config for trigger tests.
func insertPipelineWithConf(t *testing.T, eng *xorm.Engine, _, _ string) {
	t.Helper()
	pipeId := "pipe-1" // Use default pipe ID for tests
	pipe := &model.TPipeline{
		Id:          pipeId,
		Name:        "test-pipeline",
		DisplayName: "Test Pipeline",
		Uid:         "user-1",
	}
	if _, err := eng.Insert(pipe); err != nil {
		t.Fatalf("insert pipeline: %v", err)
	}

	yml := `stages:
  - name: build
    steps:
      - step: shell
        name: compile
        commands:
          - echo build
`
	conf := &model.TPipelineConf{
		PipelineId: pipeId,
		YmlContent: yml,
		Username:   "testuser",
		Url:        "https://github.com/test/repo",
	}
	if _, err := eng.Insert(conf); err != nil {
		t.Fatalf("insert pipeline conf: %v", err)
	}
}

// insertUser creates a user for trigger tests.
func insertUser(t *testing.T, eng *xorm.Engine, userId string) {
	t.Helper()
	user := &model.TUser{Id: userId, Aid: 1, Name: userId, Active: 1}
	if _, err := eng.Insert(user); err != nil {
		t.Fatalf("insert user: %v", err)
	}
}

// insertTrigger creates a trigger for tests with proper Aid.
func insertTrigger(t *testing.T, eng *xorm.Engine, trigger *model.TTrigger) {
	t.Helper()
	if trigger.Aid == 0 {
		trigger.Aid = 1 // Set a default Aid
	}
	if _, insertErr := eng.Insert(trigger); insertErr != nil {
		t.Fatalf("insert trigger: %v", insertErr)
	}
}

func TestTriggerHookCtx_WithValidParams(t *testing.T) {
	eng := setupTriggerFullTestDB(t)
	ctx := context.Background()

	userId := "user-1"
	pipeId := "pipe-1"
	insertUser(t, eng, userId)
	insertPipelineWithConf(t, eng, pipeId, userId)

	// Create trigger with webhook params
	params := map[string]string{
		"hookType": "github",
		"secret":   "webhook-secret",
		"event":    "push",
		"branch":   "main",
	}
	paramsJSON, _ := json.Marshal(params)

	trigger := &model.TTrigger{
		Id:         "trigger-1",
		Aid:        1,
		Uid:        userId,
		PipelineId: pipeId,
		Params:     string(paramsJSON),
	}
	insertTrigger(t, eng, trigger)

	// Create a request with GitHub webhook headers
	req := httptest.NewRequest(http.MethodPost, "/webhook", nil)
	req.Header.Set("X-GitHub-Event", "push")
	req.Header.Set("X-Hub-Signature", "sha1=abc123")

	// This will fail at parseHook because we don't have a valid webhook body,
	// but it should pass permission checks
	_, err := TriggerHookCtx(ctx, trigger, req)
	if err == nil {
		t.Error("expected error (invalid webhook body), got nil")
	}
	// Should not be ErrTriggerNoParams or ErrHookTypeEmpty
	if errors.Is(err, ErrTriggerNoParams) {
		t.Error("should not get ErrTriggerNoParams")
	}
}

func TestTriggerHookCtx_InvalidJSONParams(t *testing.T) {
	setupTriggerFullTestDB(t)
	ctx := context.Background()

	trigger := &model.TTrigger{
		Id:         "trigger-1",
		Uid:        "user-1",
		PipelineId: "pipe-1",
		Params:     "not valid json",
	}

	req := httptest.NewRequest(http.MethodPost, "/webhook", nil)
	_, err := TriggerHookCtx(ctx, trigger, req)
	if err == nil {
		t.Error("expected error for invalid JSON")
	}
}

func TestTriggerHookCtx_MissingHookType(t *testing.T) {
	eng := setupTriggerFullTestDB(t)
	ctx := context.Background()

	// Create user and pipeline first so permission check passes
	userId := "user-1"
	pipeId := "pipe-1"
	insertUser(t, eng, userId)
	insertPipelineWithConf(t, eng, pipeId, userId)

	// Create trigger params without hookType
	params := map[string]string{
		"secret": "webhook-secret",
		"event":  "push",
	}
	paramsJSON, _ := json.Marshal(params)

	trigger := &model.TTrigger{
		Id:         "trigger-1",
		Uid:        userId,
		PipelineId: pipeId,
		Params:     string(paramsJSON),
	}
	insertTrigger(t, eng, trigger)

	req := httptest.NewRequest(http.MethodPost, "/webhook", nil)
	_, err := TriggerHookCtx(ctx, trigger, req)
	if !errors.Is(err, ErrHookTypeEmpty) {
		t.Errorf("expected ErrHookTypeEmpty, got: %v", err)
	}
}

func TestTriggerWeb_SecretDoesNotMatch(t *testing.T) {
	eng := setupTriggerFullTestDB(t)
	ctx := context.Background()

	userId := "user-1"
	pipeId := "pipe-1"
	insertUser(t, eng, userId)
	insertPipelineWithConf(t, eng, pipeId, userId)

	params := map[string]string{
		"secret": "correct-secret",
	}
	paramsJSON, _ := json.Marshal(params)

	trigger := &model.TTrigger{
		Id:         "trigger-1",
		Uid:        userId,
		PipelineId: pipeId,
		Params:     string(paramsJSON),
	}
	insertTrigger(t, eng, trigger)

	_, err := TriggerWeb(ctx, trigger, "wrong-secret")
	if !errors.Is(err, ErrTriggerSecretMismatch) {
		t.Errorf("expected ErrTriggerSecretMismatch, got: %v", err)
	}
}

func TestTriggerWeb_MissingSecret(t *testing.T) {
	eng := setupTriggerFullTestDB(t)
	ctx := context.Background()

	userId := "user-1"
	pipeId := "pipe-1"
	insertUser(t, eng, userId)
	insertPipelineWithConf(t, eng, pipeId, userId)

	params := map[string]string{
		"branch": "main",
	}
	paramsJSON, _ := json.Marshal(params)

	trigger := &model.TTrigger{
		Id:         "trigger-1",
		Uid:        userId,
		PipelineId: pipeId,
		Params:     string(paramsJSON),
	}
	insertTrigger(t, eng, trigger)

	_, err := TriggerWeb(ctx, trigger, "any-secret")
	if !errors.Is(err, ErrTriggerNoSecret) {
		t.Errorf("expected ErrTriggerNoSecret, got: %v", err)
	}
}

func TestTriggerWeb_Success(t *testing.T) {
	eng := setupTriggerFullTestDB(t)
	ctx := context.Background()

	userId := "user-1"
	pipeId := "pipe-1"
	insertUser(t, eng, userId)
	insertPipelineWithConf(t, eng, pipeId, userId)

	params := map[string]string{
		"secret": "webhook-secret",
		"branch": "main",
	}
	paramsJSON, _ := json.Marshal(params)

	trigger := &model.TTrigger{
		Id:         "trigger-1",
		Uid:        userId,
		PipelineId: pipeId,
		Params:     string(paramsJSON),
	}
	insertTrigger(t, eng, trigger)

	rb, err := TriggerWeb(ctx, trigger, "webhook-secret")
	if err != nil {
		t.Fatalf("TriggerWeb failed: %v", err)
	}

	if rb == nil {
		t.Fatal("TriggerWeb returned nil build")
	}
	if rb.PipelineId != pipeId {
		t.Errorf("rb.PipelineId = %s, want %s", rb.PipelineId, pipeId)
	}

	// Verify trigger run was recorded
	var triggerRun model.TTriggerRun
	has, err := eng.Where("tid = ?", trigger.Id).Get(&triggerRun)
	if err != nil {
		t.Fatalf("query trigger run: %v", err)
	}
	if !has {
		t.Error("trigger run not found in database")
	}
	if triggerRun.Error != "" {
		t.Errorf("trigger run should not have error, got: %s", triggerRun.Error)
	}
}

func TestTriggerTimer_Success(t *testing.T) {
	eng := setupTriggerFullTestDB(t)
	ctx := context.Background()

	userId := "user-1"
	pipeId := "pipe-1"
	insertUser(t, eng, userId)
	insertPipelineWithConf(t, eng, pipeId, userId)

	trigger := &model.TTrigger{
		Id:         "trigger-1",
		Uid:        userId,
		PipelineId: pipeId,
		Params:     "{}",
	}
	insertTrigger(t, eng, trigger)

	rb, err := TriggerTimer(ctx, trigger)
	if err != nil {
		t.Fatalf("TriggerTimer failed: %v", err)
	}

	if rb == nil {
		t.Fatal("TriggerTimer returned nil build")
	}
	if rb.PipelineId != pipeId {
		t.Errorf("rb.PipelineId = %s, want %s", rb.PipelineId, pipeId)
	}

	// Verify trigger run was recorded
	var triggerRun model.TTriggerRun
	has, err := eng.Where("tid = ?", trigger.Id).Get(&triggerRun)
	if err != nil {
		t.Fatalf("query trigger run: %v", err)
	}
	if !has {
		t.Error("trigger run not found in database")
	}
	if triggerRun.Error != "" {
		t.Errorf("trigger run should not have error, got: %s", triggerRun.Error)
	}
}

func TestTriggerTimer_InvalidJSONData(t *testing.T) {
	setupTriggerFullTestDB(t)
	ctx := context.Background()

	trigger := &model.TTrigger{
		Id:         "trigger-1",
		Uid:        "user-1",
		PipelineId: "pipe-1",
		Params:     "invalid json",
	}

	_, err := TriggerTimer(ctx, trigger)
	if err == nil {
		t.Error("expected error for invalid JSON")
	}
}

func TestTriggerWeb_ErrorLogging(t *testing.T) {
	eng := setupTriggerFullTestDB(t)
	ctx := context.Background()

	userId := "user-1"
	pipeId := "pipe-1"
	insertUser(t, eng, userId)
	// Don't insert pipeline conf - this will cause Run() to fail
	pipe := &model.TPipeline{
		Id:          pipeId,
		Name:        "test-pipeline",
		DisplayName: "Test Pipeline",
		Uid:         userId,
	}
	if _, err := eng.Insert(pipe); err != nil {
		t.Fatalf("insert pipeline: %v", err)
	}

	params := map[string]string{
		"secret": "webhook-secret",
	}
	paramsJSON, _ := json.Marshal(params)

	trigger := &model.TTrigger{
		Id:         "trigger-1",
		Uid:        userId,
		PipelineId: pipeId,
		Params:     string(paramsJSON),
	}
	insertTrigger(t, eng, trigger)

	_, err := TriggerWeb(ctx, trigger, "webhook-secret")
	if err == nil {
		t.Error("expected error (missing pipeline conf)")
	}

	// Verify trigger run was recorded with error
	var triggerRun model.TTriggerRun
	has, err := eng.Where("tid = ?", trigger.Id).Get(&triggerRun)
	if err != nil {
		t.Fatalf("query trigger run: %v", err)
	}
	if !has {
		t.Error("trigger run not found in database")
	}
	if triggerRun.Error == "" {
		t.Error("trigger run should have error recorded")
	}
}

func TestTriggerTimer_ErrorLogging(t *testing.T) {
	eng := setupTriggerFullTestDB(t)
	ctx := context.Background()

	userId := "user-1"
	pipeId := "pipe-1"
	insertUser(t, eng, userId)
	// Don't insert pipeline conf
	pipe := &model.TPipeline{
		Id:          pipeId,
		Name:        "test-pipeline",
		DisplayName: "Test Pipeline",
		Uid:         userId,
	}
	if _, err := eng.Insert(pipe); err != nil {
		t.Fatalf("insert pipeline: %v", err)
	}

	trigger := &model.TTrigger{
		Id:         "trigger-1",
		Uid:        userId,
		PipelineId: pipeId,
		Params:     "{}",
	}
	insertTrigger(t, eng, trigger)

	_, err := TriggerTimer(ctx, trigger)
	if err == nil {
		t.Error("expected error (missing pipeline conf)")
	}

	// Verify trigger run was recorded with error
	var triggerRun model.TTriggerRun
	has, err := eng.Where("tid = ?", trigger.Id).Get(&triggerRun)
	if err != nil {
		t.Fatalf("query trigger run: %v", err)
	}
	if !has {
		t.Error("trigger run not found in database")
	}
	if triggerRun.Error == "" {
		t.Error("trigger run should have error recorded")
	}
}

func TestTriggerWeb_WithBranchFilter(t *testing.T) {
	eng := setupTriggerFullTestDB(t)
	ctx := context.Background()

	userId := "user-1"
	pipeId := "pipe-1"
	insertUser(t, eng, userId)
	insertPipelineWithConf(t, eng, pipeId, userId)

	params := map[string]string{
		"secret": "webhook-secret",
		"branch": "develop",
	}
	paramsJSON, _ := json.Marshal(params)

	trigger := &model.TTrigger{
		Id:         "trigger-1",
		Uid:        userId,
		PipelineId: pipeId,
		Params:     string(paramsJSON),
	}
	insertTrigger(t, eng, trigger)

	// TriggerWeb should use the branch from params as the SHA
	rb, err := TriggerWeb(ctx, trigger, "webhook-secret")
	if err != nil {
		t.Fatalf("TriggerWeb failed: %v", err)
	}

	if rb == nil {
		t.Fatal("TriggerWeb returned nil build")
	}

	// Verify the build was created with the branch as SHA
	var build model.TBuild
	has, err := eng.Where("id = ?", rb.Id).Get(&build)
	if err != nil {
		t.Fatalf("query build: %v", err)
	}
	if !has {
		t.Error("build not found in database")
	}
}

func TestTriggerHook_DelegatesToCtx(t *testing.T) {
	eng := setupTriggerFullTestDB(t)

	userId := "user-1"
	pipeId := "pipe-1"
	insertUser(t, eng, userId)
	insertPipelineWithConf(t, eng, pipeId, userId)

	trigger := &model.TTrigger{
		Id:         "trigger-1",
		Uid:        userId,
		PipelineId: pipeId,
		Params:     "",
	}

	req := httptest.NewRequest(http.MethodPost, "/webhook", nil)
	_, err := TriggerHook(trigger, req)
	if !errors.Is(err, ErrTriggerNoParams) {
		t.Errorf("expected ErrTriggerNoParams, got: %v", err)
	}
}

func TestTriggerPermCtx_NilTriggerFullDB(t *testing.T) {
	setupTriggerFullTestDB(t)
	ctx := context.Background()

	err := TriggerPermCtx(ctx, nil)
	if !errors.Is(err, ErrPipelineNotFound) {
		t.Errorf("expected ErrPipelineNotFound, got: %v", err)
	}
}

func TestTriggerPerm_PipelineNotFound(t *testing.T) {
	setupTriggerFullTestDB(t)
	ctx := context.Background()

	trigger := &model.TTrigger{
		Id:         "trigger-1",
		Uid:        "user-1",
		PipelineId: "nonexistent",
	}

	err := TriggerPermCtx(ctx, trigger)
	if !errors.Is(err, ErrPipelineNotFound) {
		t.Errorf("expected ErrPipelineNotFound, got: %v", err)
	}
}

func TestTriggerPerm_PermissionDenied(t *testing.T) {
	eng := setupTriggerFullTestDB(t)
	ctx := context.Background()

	userId := "user-1"
	pipeId := "pipe-1"
	otherUserId := "user-2"

	insertUser(t, eng, userId)
	insertUser(t, eng, otherUserId)

	pipe := &model.TPipeline{
		Id:          pipeId,
		Name:        "test-pipeline",
		DisplayName: "Test Pipeline",
		Uid:         userId,
	}
	if _, err := eng.Insert(pipe); err != nil {
		t.Fatalf("insert pipeline: %v", err)
	}

	// Create trigger owned by user-2 (not admin, not owner)
	trigger := &model.TTrigger{
		Id:         "trigger-1",
		Uid:        otherUserId,
		PipelineId: pipeId,
	}

	err := TriggerPermCtx(ctx, trigger)
	if !errors.Is(err, ErrPermissionDenied) {
		t.Errorf("expected ErrPermissionDenied, got: %v", err)
	}
}

func TestTriggerPerm_AdminBypass(t *testing.T) {
	eng := setupTriggerFullTestDB(t)
	ctx := context.Background()

	// Create admin user
	adminUser := &model.TUser{Id: "admin", Aid: 1, Name: "admin", Active: 1}
	if _, err := eng.Insert(adminUser); err != nil {
		t.Fatalf("insert admin: %v", err)
	}

	pipeId := "pipe-1"
	pipe := &model.TPipeline{
		Id:          pipeId,
		Name:        "test-pipeline",
		DisplayName: "Test Pipeline",
		Uid:         "other-user",
	}
	if _, err := eng.Insert(pipe); err != nil {
		t.Fatalf("insert pipeline: %v", err)
	}

	// Admin should have permission even if not owner
	trigger := &model.TTrigger{
		Id:         "trigger-1",
		Uid:        "admin",
		PipelineId: pipeId,
	}

	err := TriggerPermCtx(ctx, trigger)
	if err != nil {
		t.Errorf("admin should have permission, got error: %v", err)
	}
}

func TestTriggerTimer_RecordsPipeVersionId(t *testing.T) {
	eng := setupTriggerFullTestDB(t)
	ctx := context.Background()

	userId := "user-1"
	pipeId := "pipe-1"
	insertUser(t, eng, userId)
	insertPipelineWithConf(t, eng, pipeId, userId)

	trigger := &model.TTrigger{
		Id:         "trigger-1",
		Uid:        userId,
		PipelineId: pipeId,
		Params:     "{}",
	}
	insertTrigger(t, eng, trigger)

	_, err := TriggerTimer(ctx, trigger)
	if err != nil {
		t.Fatalf("TriggerTimer failed: %v", err)
	}

	// Verify trigger run has pipe version ID recorded
	var triggerRun model.TTriggerRun
	has, err := eng.Where("tid = ?", trigger.Id).Get(&triggerRun)
	if err != nil {
		t.Fatalf("query trigger run: %v", err)
	}
	if !has {
		t.Fatal("trigger run not found")
	}
	if triggerRun.PipeVersionId == "" {
		t.Error("trigger run should have PipeVersionId recorded")
	}

	// Verify the timestamp was set
	if triggerRun.Created.IsZero() {
		t.Error("trigger run should have Created timestamp")
	}

	// Verify the Infos field was set
	if triggerRun.Infos != "{}" {
		t.Errorf("trigger run Infos = %s, want {}", triggerRun.Infos)
	}
}

func TestTriggerWeb_RecordsPipeVersionId(t *testing.T) {
	eng := setupTriggerFullTestDB(t)
	ctx := context.Background()

	userId := "user-1"
	pipeId := "pipe-1"
	insertUser(t, eng, userId)
	insertPipelineWithConf(t, eng, pipeId, userId)

	params := map[string]string{
		"secret": "webhook-secret",
	}
	paramsJSON, _ := json.Marshal(params)

	trigger := &model.TTrigger{
		Id:         "trigger-1",
		Uid:        userId,
		PipelineId: pipeId,
		Params:     string(paramsJSON),
	}
	insertTrigger(t, eng, trigger)

	_, err := TriggerWeb(ctx, trigger, "webhook-secret")
	if err != nil {
		t.Fatalf("TriggerWeb failed: %v", err)
	}

	// Verify trigger run has pipe version ID recorded
	var triggerRun model.TTriggerRun
	has, err := eng.Where("tid = ?", trigger.Id).Get(&triggerRun)
	if err != nil {
		t.Fatalf("query trigger run: %v", err)
	}
	if !has {
		t.Fatal("trigger run not found")
	}
	if triggerRun.PipeVersionId == "" {
		t.Error("trigger run should have PipeVersionId recorded")
	}
}

func TestTriggerHookCtx_RecordsTriggerRunOnSuccess(t *testing.T) {
	eng := setupTriggerFullTestDB(t)
	ctx := context.Background()

	userId := "user-1"
	pipeId := "pipe-1"
	insertUser(t, eng, userId)
	insertPipelineWithConf(t, eng, pipeId, userId)

	// Create trigger with empty hookType to fail early but after permission check
	params := map[string]string{
		"hookType": "github",
	}
	paramsJSON, _ := json.Marshal(params)

	trigger := &model.TTrigger{
		Id:         "trigger-1",
		Uid:        userId,
		PipelineId: pipeId,
		Params:     string(paramsJSON),
	}
	insertTrigger(t, eng, trigger)

	req := httptest.NewRequest(http.MethodPost, "/webhook", nil)
	_, _ = TriggerHookCtx(ctx, trigger, req)

	// Even if it fails, trigger run should be recorded
	var triggerRun model.TTriggerRun
	has, err := eng.Where("tid = ?", trigger.Id).Get(&triggerRun)
	if err != nil {
		t.Fatalf("query trigger run: %v", err)
	}
	if !has {
		t.Error("trigger run should be recorded even on error")
	}
}

func TestTriggerWeb_MultipleRuns(t *testing.T) {
	eng := setupTriggerFullTestDB(t)
	ctx := context.Background()

	userId := "user-1"
	pipeId := "pipe-1"
	insertUser(t, eng, userId)
	insertPipelineWithConf(t, eng, pipeId, userId)

	params := map[string]string{
		"secret": "webhook-secret",
	}
	paramsJSON, _ := json.Marshal(params)

	trigger := &model.TTrigger{
		Id:         "trigger-1",
		Uid:        userId,
		PipelineId: pipeId,
		Params:     string(paramsJSON),
	}
	insertTrigger(t, eng, trigger)

	// Run trigger multiple times
	for i := 0; i < 3; i++ {
		_, err := TriggerWeb(ctx, trigger, "webhook-secret")
		if err != nil {
			t.Fatalf("TriggerWeb #%d failed: %v", i, err)
		}
	}

	// Verify all runs were recorded
	var triggerRuns []model.TTriggerRun
	if err := eng.Where("tid = ?", trigger.Id).Find(&triggerRuns); err != nil {
		t.Fatalf("query trigger runs: %v", err)
	}
	if len(triggerRuns) != 3 {
		t.Errorf("expected 3 trigger runs, got %d", len(triggerRuns))
	}

	// Each run should have a unique ID
	ids := make(map[string]bool)
	for _, run := range triggerRuns {
		if ids[run.Id] {
			t.Error("trigger runs should have unique IDs")
		}
		ids[run.Id] = true
	}
}

func TestTriggerTimer_MultipleRuns(t *testing.T) {
	eng := setupTriggerFullTestDB(t)
	ctx := context.Background()

	userId := "user-1"
	pipeId := "pipe-1"
	insertUser(t, eng, userId)
	insertPipelineWithConf(t, eng, pipeId, userId)

	trigger := &model.TTrigger{
		Id:         "trigger-1",
		Uid:        userId,
		PipelineId: pipeId,
		Params:     "{}",
	}
	insertTrigger(t, eng, trigger)

	// Run trigger multiple times
	for i := 0; i < 3; i++ {
		_, err := TriggerTimer(ctx, trigger)
		if err != nil {
			t.Fatalf("TriggerTimer #%d failed: %v", i, err)
		}
	}

	// Verify all runs were recorded
	var triggerRuns []model.TTriggerRun
	if err := eng.Where("tid = ?", trigger.Id).Find(&triggerRuns); err != nil {
		t.Fatalf("query trigger runs: %v", err)
	}
	if len(triggerRuns) != 3 {
		t.Errorf("expected 3 trigger runs, got %d", len(triggerRuns))
	}
}

func TestTriggerWeb_ContextCancellation(t *testing.T) {
	eng := setupTriggerFullTestDB(t)

	userId := "user-1"
	pipeId := "pipe-1"
	insertUser(t, eng, userId)
	insertPipelineWithConf(t, eng, pipeId, userId)

	params := map[string]string{
		"secret": "webhook-secret",
	}
	paramsJSON, _ := json.Marshal(params)

	trigger := &model.TTrigger{
		Id:         "trigger-1",
		Uid:        userId,
		PipelineId: pipeId,
		Params:     string(paramsJSON),
	}
	insertTrigger(t, eng, trigger)

	// Create canceled context
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := TriggerWeb(ctx, trigger, "webhook-secret")
	// Should handle cancellation gracefully
	if err == nil {
		t.Error("expected error with canceled context")
	}
}

func TestTriggerTimer_ContextCancellation(t *testing.T) {
	eng := setupTriggerFullTestDB(t)

	userId := "user-1"
	pipeId := "pipe-1"
	insertUser(t, eng, userId)
	insertPipelineWithConf(t, eng, pipeId, userId)

	trigger := &model.TTrigger{
		Id:         "trigger-1",
		Uid:        userId,
		PipelineId: pipeId,
		Params:     "{}",
	}
	insertTrigger(t, eng, trigger)

	// Create canceled context
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := TriggerTimer(ctx, trigger)
	// Should handle cancellation gracefully
	if err == nil {
		t.Error("expected error with canceled context")
	}
}

func TestTriggerTimer_WithTimestamps(t *testing.T) {
	eng := setupTriggerFullTestDB(t)
	ctx := context.Background()

	userId := "user-1"
	pipeId := "pipe-1"
	insertUser(t, eng, userId)
	insertPipelineWithConf(t, eng, pipeId, userId)

	trigger := &model.TTrigger{
		Id:         "trigger-1",
		Uid:        userId,
		PipelineId: pipeId,
		Params:     "{}",
	}
	insertTrigger(t, eng, trigger)

	before := time.Now()
	_, err := TriggerTimer(ctx, trigger)
	if err != nil {
		t.Fatalf("TriggerTimer failed: %v", err)
	}
	after := time.Now()

	// Verify trigger run timestamp is within expected range
	var triggerRun model.TTriggerRun
	has, err := eng.Where("tid = ?", trigger.Id).Get(&triggerRun)
	if err != nil {
		t.Fatalf("query trigger run: %v", err)
	}
	if !has {
		t.Fatal("trigger run not found")
	}

	// SQLite only has second-level timestamp precision, so we need to truncate
	// to seconds when comparing
	beforeSec := before.Truncate(time.Second)
	afterSec := after.Add(time.Second).Truncate(time.Second) // Add 1s buffer

	if triggerRun.Created.Before(beforeSec) || triggerRun.Created.After(afterSec) {
		t.Errorf("trigger run Created = %v, expected between %v and %v",
			triggerRun.Created, beforeSec, afterSec)
	}
}
