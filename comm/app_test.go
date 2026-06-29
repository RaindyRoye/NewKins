package comm

import (
	"context"
	"sync"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestGetApp(t *testing.T) {
	app := GetApp()
	assert.NotNil(t, app, "GetApp should return a non-nil App instance")

	// Verify it returns the same singleton instance
	app2 := GetApp()
	assert.Equal(t, app, app2, "GetApp should return the same singleton instance")
}

func TestSyncGlobals(t *testing.T) {
	// Set up some test values in globals
	originalWorkPath := WorkPath
	originalIsMySQL := IsMySQL
	defer func() {
		WorkPath = originalWorkPath
		IsMySQL = originalIsMySQL
	}()

	WorkPath = "/test/path"
	IsMySQL = true

	// Sync globals to App
	SyncGlobals()

	app := GetApp()
	assert.Equal(t, "/test/path", app.WorkPath, "WorkPath should be synced to App")
	assert.Equal(t, true, app.IsMySQL, "IsMySQL should be synced to App")
}

func TestSyncToGlobals(t *testing.T) {
	// Set up some test values in App
	app := GetApp()
	originalWorkPath := app.WorkPath
	originalIsMySQL := app.IsMySQL
	defer func() {
		app.WorkPath = originalWorkPath
		app.IsMySQL = originalIsMySQL
	}()

	app.WorkPath = "/app/path"
	app.IsMySQL = false

	// Sync App back to globals
	SyncToGlobals()

	assert.Equal(t, "/app/path", WorkPath, "WorkPath should be synced from App to globals")
	assert.Equal(t, false, IsMySQL, "IsMySQL should be synced from App to globals")
}

func TestAppStructFields(t *testing.T) {
	// Test that all fields exist and can be set
	app := &App{
		Cfg:       Config{},
		Db:        nil,
		BCache:    nil,
		WebEgn:    gin.New(),
		HbtpEgn:   nil,
		IsMySQL:   true,
		Installed: true,
		NotUpPass: true,
		WorkPath:  "/test",
		WebHost:   "localhost:8080",
	}

	assert.NotNil(t, app, "App struct should be instantiable")
	assert.Equal(t, true, app.IsMySQL, "IsMySQL field should be settable")
	assert.Equal(t, true, app.Installed, "Installed field should be settable")
	assert.Equal(t, true, app.NotUpPass, "NotUpPass field should be settable")
	assert.Equal(t, "/test", app.WorkPath, "WorkPath field should be settable")
	assert.Equal(t, "localhost:8080", app.WebHost, "WebHost field should be settable")
	assert.NotNil(t, app.WebEgn, "WebEgn field should be settable")
}

func TestBidirectionalSync(t *testing.T) {
	// Test that bidirectional sync maintains consistency
	originalWorkPath := WorkPath
	originalWebHost := WebHost
	defer func() {
		WorkPath = originalWorkPath
		WebHost = originalWebHost
	}()

	// Set globals
	WorkPath = "/initial"
	WebHost = "0.0.0.0:3000"

	// Sync globals -> App
	SyncGlobals()
	app := GetApp()
	assert.Equal(t, "/initial", app.WorkPath)
	assert.Equal(t, "0.0.0.0:3000", app.WebHost)

	// Modify App
	app.WorkPath = "/modified"
	app.WebHost = "127.0.0.1:8080"

	// Sync App -> globals
	SyncToGlobals()
	assert.Equal(t, "/modified", WorkPath)
	assert.Equal(t, "127.0.0.1:8080", WebHost)

	// Sync globals -> App again
	WorkPath = "/final"
	SyncGlobals()
	assert.Equal(t, "/final", app.WorkPath)
}

func TestMarkInstalled(t *testing.T) {
	// Reset state for this test
	oldInstalled := Installed
	oldCh := InstalledCh
	defer func() {
		Installed = oldInstalled
		InstalledCh = oldCh
		installOnce = sync.Once{}
	}()

	Installed = false
	InstalledCh = make(chan struct{})
	installOnce = sync.Once{}

	MarkInstalled()
	assert.True(t, Installed, "Installed should be true after MarkInstalled")

	// Channel should be closed
	select {
	case <-InstalledCh:
		// OK, channel is closed
	default:
		t.Fatal("InstalledCh should be closed after MarkInstalled")
	}

	// Calling again should not panic
	MarkInstalled()
	assert.True(t, Installed, "Installed should remain true after second call")
}

func TestCancel(t *testing.T) {
	// Save original context state
	oldCtx := Ctx
	oldCncl := cncl
	defer func() {
		Ctx = oldCtx
		cncl = oldCncl
	}()

	// Create a fresh context
	ResetCtx()

	// Cancel should not panic
	Cancel()

	// Context should be canceled
	select {
	case <-Ctx.Done():
		// OK
	default:
		t.Fatal("Ctx should be done after Cancel")
	}
}

func TestResetCtx(t *testing.T) {
	// Save original context state
	oldCtx := Ctx
	oldCncl := cncl
	defer func() {
		Ctx = oldCtx
		cncl = oldCncl
	}()

	// Reset context
	ResetCtx()
	ctx1 := Ctx

	// Verify context is not canceled
	select {
	case <-ctx1.Done():
		t.Fatal("new context should not be canceled")
	default:
		// OK
	}

	// Reset again should create a new context
	ResetCtx()
	ctx2 := Ctx

	if ctx1 == ctx2 {
		t.Error("ResetCtx should create a new context instance")
	}

	// Old context should be canceled
	select {
	case <-ctx1.Done():
		// OK
	default:
		t.Fatal("old context should be canceled after ResetCtx")
	}
}

func TestCancel_NilCancel(t *testing.T) {
	// Save original state
	oldCncl := cncl
	defer func() { cncl = oldCncl }()

	// Set cncl to nil
	cncl = nil

	// Should not panic
	Cancel()
}

func TestCtx_IsBackground(t *testing.T) {
	// Ctx should be initialized in init()
	assert.NotNil(t, Ctx, "Ctx should be initialized")
	assert.NotNil(t, cncl, "cncl should be initialized")

	// Verify it's a valid context by accessing its Deadline and Value methods
	_, ok := Ctx.Deadline()
	assert.False(t, ok, "background context should have no deadline")
	assert.Nil(t, Ctx.Value("nonexistent"), "context should return nil for unknown keys")
}

func TestCtx_Cancellation(t *testing.T) {
	// Save original state
	oldCtx := Ctx
	oldCncl := cncl
	defer func() {
		Ctx = oldCtx
		cncl = oldCncl
	}()

	ResetCtx()

	// Verify context starts uncanceled
	err := Ctx.Err()
	assert.NoError(t, err, "new context should not be canceled")

	// Cancel it
	Cancel()

	// Verify context is now canceled
	err = Ctx.Err()
	assert.ErrorIs(t, err, context.Canceled, "context should be canceled")
}
