package service

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gokins/gokins/comm"
	"github.com/gokins/gokins/model"
	_ "github.com/mattn/go-sqlite3"
	"xorm.io/xorm"
)

func setupHookTestDB(t *testing.T) *xorm.Engine {
	t.Helper()
	eng, err := xorm.NewEngine("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("create test engine: %v", err)
	}
	comm.Db = eng
	t.Cleanup(func() {
		_ = eng.Close()
		comm.Db = nil
	})

	tables := []interface{}{
		&model.TUser{},
		&model.TPipeline{},
		&model.TTrigger{},
		&model.TTriggerRun{},
		&model.TPipelineVersion{},
		&model.TPipelineConf{},
	}
	for _, table := range tables {
		if err := eng.Sync2(table); err != nil {
			t.Fatalf("sync table: %v", err)
		}
	}
	return eng
}

func TestTriggerHookCtx_NoParams(t *testing.T) {
	setupHookTestDB(t)

	trigger := &model.TTrigger{
		Id:         "trigger-1",
		PipelineId: "pipe-1",
		Params:     "",
	}

	req := httptest.NewRequest(http.MethodPost, "/webhook", nil)
	ctx := context.Background()

	_, err := TriggerHookCtx(ctx, trigger, req)
	if !errors.Is(err, ErrTriggerNoParams) {
		t.Errorf("expected ErrTriggerNoParams, got: %v", err)
	}
}

func TestTriggerHook_NoParams(t *testing.T) {
	setupHookTestDB(t)

	trigger := &model.TTrigger{
		Id:         "trigger-1",
		PipelineId: "pipe-1",
		Params:     "",
	}

	req := httptest.NewRequest(http.MethodPost, "/webhook", nil)

	_, err := TriggerHook(trigger, req)
	if !errors.Is(err, ErrTriggerNoParams) {
		t.Errorf("expected ErrTriggerNoParams, got: %v", err)
	}
}

func TestTriggerWeb_NoParams(t *testing.T) {
	setupHookTestDB(t)

	trigger := &model.TTrigger{
		Id:         "trigger-1",
		PipelineId: "pipe-1",
		Params:     "",
	}

	ctx := context.Background()
	_, err := TriggerWeb(ctx, trigger, "secret")
	if !errors.Is(err, ErrTriggerNoParams) {
		t.Errorf("expected ErrTriggerNoParams, got: %v", err)
	}
}

func TestTriggerTimer_NoParams(t *testing.T) {
	eng := setupHookTestDB(t)

	// Create a valid pipeline and user so permission check passes
	pipe := &model.TPipeline{Id: "pipe-1", Name: "test-pipeline"}
	if _, err := eng.Insert(pipe); err != nil {
		t.Fatalf("insert pipeline: %v", err)
	}

	user := &model.TUser{Id: "user-1", Aid: 1, Name: "admin", Active: 1}
	if _, err := eng.Insert(user); err != nil {
		t.Fatalf("insert user: %v", err)
	}

	trigger := &model.TTrigger{
		Id:         "trigger-1",
		Uid:        "user-1",
		PipelineId: "pipe-1",
		Params:     "", // Timer triggers don't use params
	}

	ctx := context.Background()
	// TriggerTimer should not panic with empty params
	// It will fail at Run() since we don't have full pipeline setup,
	// but that's expected - we're just testing it doesn't crash on empty params
	_, _ = TriggerTimer(ctx, trigger)
}

func TestTriggerHookCtx_InvalidJSON(t *testing.T) {
	setupHookTestDB(t)

	trigger := &model.TTrigger{
		Id:         "trigger-1",
		PipelineId: "pipe-1",
		Params:     "invalid json",
	}

	req := httptest.NewRequest(http.MethodPost, "/webhook", nil)
	ctx := context.Background()

	_, err := TriggerHookCtx(ctx, trigger, req)
	if err == nil {
		t.Error("expected error for invalid JSON params")
	}
}

func TestTriggerWeb_InvalidJSON(t *testing.T) {
	setupHookTestDB(t)

	trigger := &model.TTrigger{
		Id:         "trigger-1",
		PipelineId: "pipe-1",
		Params:     "invalid json",
	}

	ctx := context.Background()
	_, err := TriggerWeb(ctx, trigger, "secret")
	if err == nil {
		t.Error("expected error for invalid JSON params")
	}
}

func TestTriggerTimer_InvalidJSON(t *testing.T) {
	setupHookTestDB(t)

	trigger := &model.TTrigger{
		Id:         "trigger-1",
		PipelineId: "pipe-1",
		Params:     "invalid json",
	}

	ctx := context.Background()
	_, err := TriggerTimer(ctx, trigger)
	if err == nil {
		t.Error("expected error for invalid JSON params")
	}
}

func TestParseHookInteg_UnknownType(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/webhook", strings.NewReader("{}"))
	_, err := parseHook("unknown-type", req, "secret")
	if !errors.Is(err, ErrUnknownWebhookType) {
		t.Errorf("expected ErrUnknownWebhookType, got: %v", err)
	}
}

func TestParseHookInteg_Gitee(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/webhook", strings.NewReader("{}"))
	_, err := parseHook("gitee", req, "secret")
	if errors.Is(err, ErrUnknownWebhookType) {
		t.Error("gitee should be a known webhook type")
	}
}

func TestParseHookInteg_GiteePremium(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/webhook", strings.NewReader("{}"))
	_, err := parseHook("giteepremium", req, "secret")
	if errors.Is(err, ErrUnknownWebhookType) {
		t.Error("giteepremium should be a known webhook type")
	}
}

func TestParseHookInteg_Github(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/webhook", strings.NewReader("{}"))
	_, err := parseHook("github", req, "secret")
	if errors.Is(err, ErrUnknownWebhookType) {
		t.Error("github should be a known webhook type")
	}
}

func TestParseHookInteg_Gitlab(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/webhook", strings.NewReader("{}"))
	_, err := parseHook("gitlab", req, "secret")
	if errors.Is(err, ErrUnknownWebhookType) {
		t.Error("gitlab should be a known webhook type")
	}
}

func TestParseHookInteg_Gitea(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/webhook", strings.NewReader("{}"))
	_, err := parseHook("gitea", req, "secret")
	if errors.Is(err, ErrUnknownWebhookType) {
		t.Error("gitea should be a known webhook type")
	}
}

func TestParseHookInteg_CaseInsensitive(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/webhook", strings.NewReader("{}"))

	testCases := []string{"GITEE", "Gitee", "GITHUB", "GitHub", "GITLAB", "GitLab", "GITEA", "Gitea"}
	for _, hookType := range testCases {
		_, err := parseHook(hookType, req, "secret")
		if errors.Is(err, ErrUnknownWebhookType) {
			t.Errorf("%s should be recognized as a known webhook type", hookType)
		}
	}
}

// --- TriggerWeb additional tests ---

func TestTriggerWeb_NoSecret(t *testing.T) {
	eng := setupHookTestDB(t)

	// Create pipeline and user so permission check passes
	pipe := &model.TPipeline{Id: "pipe-1", Name: "test-pipeline", Uid: "user-1"}
	if _, err := eng.Insert(pipe); err != nil {
		t.Fatalf("insert pipeline: %v", err)
	}

	user := &model.TUser{Id: "user-1", Aid: 1, Name: "admin", Active: 1}
	if _, err := eng.Insert(user); err != nil {
		t.Fatalf("insert user: %v", err)
	}

	trigger := &model.TTrigger{
		Id:         "trigger-1",
		Uid:        "user-1",
		PipelineId: "pipe-1",
		Params:     `{"hookType":"github","secret":"mysecret"}`,
	}

	ctx := context.Background()
	_, err := TriggerWeb(ctx, trigger, "wrongsecret")
	if !errors.Is(err, ErrTriggerSecretMismatch) {
		t.Errorf("expected ErrTriggerSecretMismatch, got: %v", err)
	}
}

func TestTriggerWeb_MissingSecretInParams(t *testing.T) {
	eng := setupHookTestDB(t)

	// Create pipeline and user so permission check passes
	pipe := &model.TPipeline{Id: "pipe-1", Name: "test-pipeline", Uid: "user-1"}
	if _, err := eng.Insert(pipe); err != nil {
		t.Fatalf("insert pipeline: %v", err)
	}

	user := &model.TUser{Id: "user-1", Aid: 1, Name: "admin", Active: 1}
	if _, err := eng.Insert(user); err != nil {
		t.Fatalf("insert user: %v", err)
	}

	trigger := &model.TTrigger{
		Id:         "trigger-1",
		Uid:        "user-1",
		PipelineId: "pipe-1",
		Params:     `{"hookType":"github"}`,
	}

	ctx := context.Background()
	_, err := TriggerWeb(ctx, trigger, "secret")
	if !errors.Is(err, ErrTriggerNoSecret) {
		t.Errorf("expected ErrTriggerNoSecret, got: %v", err)
	}
}

// --- TriggerTimer additional tests ---

func TestTriggerTimer_SuccessPath(t *testing.T) {
	eng := setupHookTestDB(t)

	// Create a valid pipeline and user
	pipe := &model.TPipeline{Id: "pipe-1", Name: "test-pipeline", Uid: "user-1"}
	if _, err := eng.Insert(pipe); err != nil {
		t.Fatalf("insert pipeline: %v", err)
	}

	user := &model.TUser{Id: "user-1", Aid: 1, Name: "admin", Active: 1}
	if _, err := eng.Insert(user); err != nil {
		t.Fatalf("insert user: %v", err)
	}

	trigger := &model.TTrigger{
		Id:         "trigger-1",
		Uid:        "user-1",
		PipelineId: "pipe-1",
		Params:     "",
	}

	ctx := context.Background()
	// This will fail at Run() because we don't have full pipeline config,
	// but it should pass the permission check
	_, err := TriggerTimer(ctx, trigger)
	// We expect an error from Run() since pipeline config doesn't exist
	if err == nil {
		t.Error("expected error from Run() due to missing pipeline config")
	}
	if !strings.Contains(err.Error(), "run pipeline") {
		t.Errorf("error should mention 'run pipeline', got: %v", err)
	}
}

func TestTriggerTimer_PermissionDenied(t *testing.T) {
	eng := setupHookTestDB(t)

	// Create pipeline but user doesn't have write permission
	pipe := &model.TPipeline{Id: "pipe-1", Name: "test-pipeline", Uid: "other-user"}
	if _, err := eng.Insert(pipe); err != nil {
		t.Fatalf("insert pipeline: %v", err)
	}

	user := &model.TUser{Id: "user-1", Aid: 1, Name: "regular", Active: 1}
	if _, err := eng.Insert(user); err != nil {
		t.Fatalf("insert user: %v", err)
	}

	trigger := &model.TTrigger{
		Id:         "trigger-1",
		Uid:        "user-1",
		PipelineId: "pipe-1",
		Params:     "",
	}

	ctx := context.Background()
	_, err := TriggerTimer(ctx, trigger)
	if !errors.Is(err, ErrPermissionDenied) {
		t.Errorf("expected ErrPermissionDenied, got: %v", err)
	}
}

// --- TriggerHookCtx additional coverage tests ---

func TestTriggerHookCtx_MissingHookType(t *testing.T) {
	eng := setupHookTestDB(t)

	// Create pipeline and user so permission check passes
	pipe := &model.TPipeline{Id: "pipe-1", Name: "test-pipeline", Uid: "user-1"}
	if _, err := eng.Insert(pipe); err != nil {
		t.Fatalf("insert pipeline: %v", err)
	}

	user := &model.TUser{Id: "user-1", Aid: 1, Name: "admin", Active: 1}
	if _, err := eng.Insert(user); err != nil {
		t.Fatalf("insert user: %v", err)
	}

	trigger := &model.TTrigger{
		Id:         "trigger-1",
		Uid:        "user-1",
		PipelineId: "pipe-1",
		Params:     `{"secret":"test"}`,
	}

	req := httptest.NewRequest(http.MethodPost, "/webhook", nil)
	ctx := context.Background()

	_, err := TriggerHookCtx(ctx, trigger, req)
	if !errors.Is(err, ErrHookTypeEmpty) {
		t.Errorf("expected ErrHookTypeEmpty, got: %v", err)
	}
}

func TestTriggerHookCtx_WebhookParseFailed(t *testing.T) {
	eng := setupHookTestDB(t)

	// Create pipeline and user
	pipe := &model.TPipeline{Id: "pipe-1", Name: "test-pipeline", Uid: "user-1"}
	if _, err := eng.Insert(pipe); err != nil {
		t.Fatalf("insert pipeline: %v", err)
	}

	user := &model.TUser{Id: "user-1", Aid: 1, Name: "admin", Active: 1}
	if _, err := eng.Insert(user); err != nil {
		t.Fatalf("insert user: %v", err)
	}

	trigger := &model.TTrigger{
		Id:         "trigger-1",
		Uid:        "user-1",
		PipelineId: "pipe-1",
		Params:     `{"hookType":"unknown-type"}`,
	}

	req := httptest.NewRequest(http.MethodPost, "/webhook", nil)
	ctx := context.Background()

	_, err := TriggerHookCtx(ctx, trigger, req)
	if err == nil {
		t.Error("expected error for unknown webhook type")
	}
	if !errors.Is(err, ErrUnknownWebhookType) {
		t.Errorf("expected ErrUnknownWebhookType, got: %v", err)
	}
}

func TestTriggerHookCtx_PermissionDenied(t *testing.T) {
	eng := setupHookTestDB(t)

	// Create pipeline owned by different user
	pipe := &model.TPipeline{Id: "pipe-1", Name: "test-pipeline", Uid: "other-user"}
	if _, err := eng.Insert(pipe); err != nil {
		t.Fatalf("insert pipeline: %v", err)
	}

	user := &model.TUser{Id: "user-1", Aid: 1, Name: "regular", Active: 1}
	if _, err := eng.Insert(user); err != nil {
		t.Fatalf("insert user: %v", err)
	}

	trigger := &model.TTrigger{
		Id:         "trigger-1",
		Uid:        "user-1",
		PipelineId: "pipe-1",
		Params:     `{"hookType":"github"}`,
	}

	req := httptest.NewRequest(http.MethodPost, "/webhook", nil)
	ctx := context.Background()

	_, err := TriggerHookCtx(ctx, trigger, req)
	if !errors.Is(err, ErrPermissionDenied) {
		t.Errorf("expected ErrPermissionDenied, got: %v", err)
	}
}
