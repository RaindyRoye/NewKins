package server

import (
	"runtime/debug"

	"github.com/gokins/gokins/comm"
	"github.com/gokins/gokins/engine"
	hbtp "github.com/mgr9525/HyperByte-Transfer-Protocol"
	"github.com/sirupsen/logrus"
)

func runHbtp() {
	defer func() {
		if err := recover(); err != nil {
			logrus.Errorf("Hbtp recover:%v", err)
			logrus.Errorf("Hbtp stack:%s", string(debug.Stack()))
		}
	}()
	if comm.Cfg.Server.HbtpHost == "" {
		return
	}
	comm.HbtpEgn = hbtp.NewEngine(comm.Ctx)
	comm.HbtpEgn.RegGrpcFun(10, engine.Mgr.HRun())
	err := comm.HbtpEgn.Run(comm.Cfg.Server.HbtpHost)
	if err != nil {
		logrus.Errorf("Hbtp err:%v", err)
		comm.Cancel()
	}
}
