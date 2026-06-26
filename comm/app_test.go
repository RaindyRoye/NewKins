package comm

import (
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
