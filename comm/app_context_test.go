package comm

import (
	"context"
	"testing"
)

func TestAppFromContext_NilContext(t *testing.T) {
	app := AppFromContext(nil)
	if app == nil {
		t.Fatal("expected defaultApp, got nil")
	}
	if app != defaultApp {
		t.Error("expected defaultApp singleton when context is nil")
	}
}

func TestAppFromContext_EmptyContext(t *testing.T) {
	ctx := context.Background()
	app := AppFromContext(ctx)
	if app == nil {
		t.Fatal("expected defaultApp, got nil")
	}
	if app != defaultApp {
		t.Error("expected defaultApp when context has no App")
	}
}

func TestAppFromContext_WithApp(t *testing.T) {
	customApp := &App{
		WorkPath: "/custom/path",
		WebHost:  "http://custom.test",
	}
	ctx := WithApp(context.Background(), customApp)
	
	app := AppFromContext(ctx)
	if app == nil {
		t.Fatal("expected app from context, got nil")
	}
	if app != customApp {
		t.Error("expected custom app from context")
	}
	if app.WorkPath != "/custom/path" {
		t.Errorf("WorkPath = %q, want %q", app.WorkPath, "/custom/path")
	}
	if app.WebHost != "http://custom.test" {
		t.Errorf("WebHost = %q, want %q", app.WebHost, "http://custom.test")
	}
}

func TestAppFromContext_FallbackOnWrongType(t *testing.T) {
	// Manually create a context with wrong key type (simulating external code)
	ctx := context.WithValue(context.Background(), "wrong-key", &App{})
	app := AppFromContext(ctx)
	if app != defaultApp {
		t.Error("expected defaultApp fallback when key type is wrong")
	}
}

func TestWithApp_ChildContext(t *testing.T) {
	parentApp := &App{WorkPath: "/parent"}
	childApp := &App{WorkPath: "/child"}
	
	parentCtx := WithApp(context.Background(), parentApp)
	childCtx := WithApp(parentCtx, childApp)
	
	// Parent context should return parentApp
	if got := AppFromContext(parentCtx); got != parentApp {
		t.Error("parent context should return parentApp")
	}
	
	// Child context should return childApp (shadowing parent)
	if got := AppFromContext(childCtx); got != childApp {
		t.Error("child context should return childApp")
	}
	if got := AppFromContext(childCtx); got.WorkPath != "/child" {
		t.Errorf("child WorkPath = %q, want %q", got.WorkPath, "/child")
	}
}

func TestWithApp_MultipleValues(t *testing.T) {
	app := &App{
		WorkPath: "/test",
		WebHost:  "http://test.local",
		IsMySQL:  true,
	}
	ctx := WithApp(context.Background(), app)
	
	// Verify all fields are accessible
	got := AppFromContext(ctx)
	if got.WorkPath != app.WorkPath {
		t.Errorf("WorkPath mismatch")
	}
	if got.WebHost != app.WebHost {
		t.Errorf("WebHost mismatch")
	}
	if got.IsMySQL != app.IsMySQL {
		t.Errorf("IsMySQL mismatch")
	}
}

func TestGetApp_ReturnsDefault(t *testing.T) {
	app := GetApp()
	if app == nil {
		t.Fatal("GetApp returned nil")
	}
	if app != defaultApp {
		t.Error("GetApp should return defaultApp")
	}
}

func TestAppContextKey_Isolation(t *testing.T) {
	// Verify that appContextKey doesn't collide with string keys
	ctx := context.WithValue(context.Background(), "appContextKey", "string-value")
	
	// Should still fall back to defaultApp
	app := AppFromContext(ctx)
	if app != defaultApp {
		t.Error("appContextKey should not collide with string keys")
	}
	
	// Now add real App
	customApp := &App{WorkPath: "/custom"}
	ctx = WithApp(ctx, customApp)
	app = AppFromContext(ctx)
	if app != customApp {
		t.Error("should retrieve custom app after WithApp")
	}
}
