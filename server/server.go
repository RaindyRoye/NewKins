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
	go runWeb()
	time.Sleep(time.Millisecond * 10)
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
	// Block until context is cancelled (signal received)
	<-comm.Ctx.Done()
	logrus.Info("Context cancelled, initiating shutdown...")
	// Give background goroutines time to clean up
	time.Sleep(time.Second)
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
	return nil
}
