package comm

import (
	"context"
	"sync"

	"github.com/gin-gonic/gin"
	hbtp "github.com/mgr9525/HyperByte-Transfer-Protocol"
	bolt "go.etcd.io/bbolt"
	"xorm.io/xorm"
)

const Version = "1.3.7"

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
