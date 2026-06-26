package comm

import (
	"context"
	"sync"

	"github.com/gin-gonic/gin"
	hbtp "github.com/mgr9525/HyperByte-Transfer-Protocol"
	bolt "go.etcd.io/bbolt"
	"xorm.io/xorm"
)

// Version is the semantic version of the application.
// It can be overridden at build time via -ldflags.
var Version = "1.3.7"

// BuildTime is the UTC timestamp of the build.
// Set at build time via -ldflags: -X 'github.com/gokins/gokins/comm.BuildTime=...'
var BuildTime = "unknown"

// GitCommit is the short git commit SHA of the build.
// Set at build time via -ldflags: -X 'github.com/gokins/gokins/comm.GitCommit=...'
var GitCommit = "unknown"

// MaskedValue is used to redact sensitive data in API responses and logs.
const MaskedValue = "***"

var (
	Ctx  context.Context
	cncl context.CancelFunc
)

// InstalledCh is closed when the installation process completes.
// It is used to signal the main server loop instead of busy-waiting.
var InstalledCh = make(chan struct{})

var installOnce sync.Once

// MarkInstalled signals that installation is complete. Safe to call multiple times.
func MarkInstalled() {
	Installed = true
	installOnce.Do(func() {
		close(InstalledCh)
	})
}

// App holds application-wide dependencies. This struct groups the global
// state that was previously scattered as package-level variables, providing
// a migration path toward dependency injection while maintaining backward
// compatibility with existing code that references the globals directly.
//
// New code should prefer accessing state through an *App instance passed
// explicitly, rather than through the package-level globals.
type App struct {
	Cfg       Config
	Db        *xorm.Engine
	BCache    *bolt.DB
	WebEgn    *gin.Engine
	HbtpEgn   *hbtp.Engine
	IsMySQL   bool
	Installed bool
	NotUpPass bool
	WorkPath  string
	WebHost   string
}

var (
	Cfg       = Config{}
	Db        *xorm.Engine
	BCache    *bolt.DB
	WebEgn    *gin.Engine
	HbtpEgn   *hbtp.Engine
	IsMySQL   = false
	Installed = false
	NotUpPass = false
	WorkPath  = ""
	WebHost   = ""
	// HbtpHost = ""
)

// defaultApp is the singleton App instance that mirrors the package-level
// globals. It enables a gradual migration path to dependency injection.
var defaultApp = &App{}

// GetApp returns the default App instance. This is the entry point for code
// that wants to use dependency injection instead of global variables.
func GetApp() *App {
	return defaultApp
}

// SyncGlobals copies the package-level globals into the default App instance.
// Call this after initializing the globals (e.g., after database setup) to
// ensure the App struct reflects the current state.
func SyncGlobals() {
	defaultApp.Cfg = Cfg
	defaultApp.Db = Db
	defaultApp.BCache = BCache
	defaultApp.WebEgn = WebEgn
	defaultApp.HbtpEgn = HbtpEgn
	defaultApp.IsMySQL = IsMySQL
	defaultApp.Installed = Installed
	defaultApp.NotUpPass = NotUpPass
	defaultApp.WorkPath = WorkPath
	defaultApp.WebHost = WebHost
}

// SyncToGlobals copies the App struct fields back to the package-level globals.
// This is useful during migration when some code uses the App struct while
// other code still references the globals directly.
func SyncToGlobals() {
	Cfg = defaultApp.Cfg
	Db = defaultApp.Db
	BCache = defaultApp.BCache
	WebEgn = defaultApp.WebEgn
	HbtpEgn = defaultApp.HbtpEgn
	IsMySQL = defaultApp.IsMySQL
	Installed = defaultApp.Installed
	NotUpPass = defaultApp.NotUpPass
	WorkPath = defaultApp.WorkPath
	WebHost = defaultApp.WebHost
}

func init() {
	Ctx, cncl = context.WithCancel(context.Background())
}
func Cancel() {
	if cncl != nil {
		cncl()
	}
}

// ResetCtx creates a fresh context and cancel function. This is primarily
// useful in tests to isolate context state between test cases.
func ResetCtx() {
	if cncl != nil {
		cncl()
	}
	Ctx, cncl = context.WithCancel(context.Background())
}
