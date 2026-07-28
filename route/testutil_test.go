package route

import (
	"testing"

	"github.com/gokins/gokins/engine"
)

// initTestJobEngine initializes a minimal job engine for testing.
// This is needed for tests that call engine.Mgr.Plugins().
func initTestJobEngine(t *testing.T) {
	t.Helper()
	// Create a minimal job engine without starting the background goroutine
	engine.Mgr.InitForTest()
}
