package server

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/gokins/core"
	utils2 "github.com/gokins/core/utils"
	"github.com/gokins/gokins/comm"
	"github.com/gokins/gokins/engine"
	"github.com/gokins/gokins/route"
	"github.com/gokins/gokins/util"
	hbtp "github.com/mgr9525/HyperByte-Transfer-Protocol"
	"github.com/sirupsen/logrus"
	"gopkg.in/yaml.v3"
)

// shutdownGracePeriod is the maximum time allowed for background goroutines
// to finish after the context is canceled.
const shutdownGracePeriod = 3 * time.Second

// webReadyCh is closed when the web server goroutine has initialized the gin
// engine and is ready to accept configuration (install routes, API registration).
var webReadyCh = make(chan struct{})

// Run starts the Gokins server. It coordinates startup between the web server,
// database initialization, and the build engine using proper synchronization
// instead of fixed-duration sleeps.
func Run() error {
	if comm.WorkPath == "" {
		pth := filepath.Join(utils2.HomePath(), ".gokins")
		comm.WorkPath = utils2.EnvDefault("GOKINS_WORKPATH", pth)
	}
	if !comm.NotUpPass {
		comm.NotUpPass = utils2.EnvDefault("GOKINS_NOTUPDATEPASS") == "true"
	}

	logrus.Infof("gokins Run workpath:%s", comm.WorkPath)

	if err := os.MkdirAll(comm.WorkPath, 0750); err != nil {
		logrus.Warnf("create work path err: %v", err)
	}
	core.InitLog(comm.WorkPath)

	// Reset the ready channel in case Run is called multiple times (e.g. in tests)
	webReadyCh = make(chan struct{})
	go runWeb()

	// Wait for the web server to initialize the gin engine before
	// registering install routes on it.
	select {
	case <-webReadyCh:
		// Web server is ready
	case <-comm.Ctx.Done():
		return fmt.Errorf("shutdown before web server ready: %w", comm.Ctx.Err())
	case <-time.After(10 * time.Second):
		return fmt.Errorf("web server did not start within 10 seconds")
	}

	err := parseConfig()
	if err != nil {
		logrus.Debugf("parseConfig err:%v", err)
		comm.WebEgn.GET("/install", route.Install)
		util.GinRegController(comm.WebEgn, &route.InstallController{})
		// Wait for installation to complete or context cancellation
		select {
		case <-comm.InstalledCh:
			// Installation completed
		case <-comm.Ctx.Done():
			return fmt.Errorf("shutdown during installation: %w", comm.Ctx.Err())
		}
	}

	err = initDb()
	if err != nil {
		return fmt.Errorf("initDb: %w", err)
	}
	ensureIndexes()
	err = initCache()
	if err != nil {
		return fmt.Errorf("initCache: %w", err)
	}
	defer func() { _ = comm.BCache.Close() }()

	regApi()
	comm.MarkInstalled()
	err = engine.Start()
	if err != nil {
		return fmt.Errorf("engine.Start: %w", err)
	}

	go runHbtp()
	hbtp.Infof("gokins running in %s", comm.WorkPath)
	// Block until context is canceled (signal received)
	<-comm.Ctx.Done()
	logrus.Info("Context canceled, waiting for background goroutines to finish...")

	// Wait for the web server goroutine to finish its shutdown sequence,
	// with a bounded timeout to prevent hanging indefinitely.
	select {
	case <-webDoneCh:
		logrus.Info("Web server shutdown complete")
	case <-time.After(shutdownGracePeriod):
		logrus.Warn("Shutdown grace period exceeded, forcing exit")
	}
	return nil
}
func parseConfig() error {
	bts, err := os.ReadFile(filepath.Join(comm.WorkPath, "app.yml"))
	if err != nil {
		bts, err = os.ReadFile(filepath.Join(comm.WorkPath, "app.yaml"))
	}
	if err != nil {
		return fmt.Errorf("read config file: %w", err)
	}
	if err := yaml.Unmarshal(bts, &comm.Cfg); err != nil {
		return fmt.Errorf("parse config yaml: %w", err)
	}
	if err := comm.Cfg.Validate(); err != nil {
		return fmt.Errorf("invalid configuration: %w", err)
	}
	return nil
}
