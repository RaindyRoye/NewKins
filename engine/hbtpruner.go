package engine

import (
	"fmt"
	"strconv"
	"time"

	"github.com/gokins/core/common"
	"github.com/gokins/core/utils"
	"github.com/gokins/gokins/comm"
	"github.com/gokins/runner/runners"
	hbtp "github.com/mgr9525/HyperByte-Transfer-Protocol"
	"github.com/sirupsen/logrus"
)

type HbtpRunner struct {
}

func (HbtpRunner) AuthFun() hbtp.AuthFun {
	return func(c *hbtp.Context) bool {
		cmds := c.Command()
		times := c.Args().Get("times")
		random := c.Args().Get("random")
		sign := c.Args().Get("sign")
		if cmds == "" || len(times) <= 5 || len(random) < 20 || sign == "" {
			_ = c.ResString(hbtp.ResStatusAuth, "auth param err")
			return false
		}
		signs := utils.Md5String(cmds + random + times + comm.Cfg.Server.Secret)
		if sign != signs {
			logrus.Warnf("hbtp runner auth failed: signature mismatch for command %q", cmds)
			_ = c.ResString(hbtp.ResStatusAuth, "authentication failed")
			return false
		}
		tm, err := strconv.ParseInt(times, 10, 64)
		if err != nil {
			_ = c.ResString(hbtp.ResStatusAuth, "times err:"+err.Error())
			return false
		}
		tms := time.Unix(tm, 0)
		/*if err != nil {
			c.ResString(hbtp.ResStatusAuth, "times err:"+err.Error())
			return false
		}*/
		hbtp.Debugf("HbtpRunnerAuth parse.times:%s", tms.Format(common.TimeFmt))
		return true
	}
}
func (HbtpRunner) ServerInfo(c *hbtp.Context) {
	rts, err := Mgr.brun.ServerInfo()
	if err != nil {
		_ = c.ResString(hbtp.ResStatusErr, err.Error())
		return
	}
	_ = c.ResJson(hbtp.ResStatusOk, rts)
}
func (HbtpRunner) PullJob(c *hbtp.Context, m *runners.ReqPullJob) {
	rts, err := Mgr.brun.PullJob(m.Name, m.Plugs)
	if err != nil {
		_ = c.ResString(hbtp.ResStatusErr, err.Error())
		return
	}
	_ = c.ResJson(hbtp.ResStatusOk, rts)
}
func (HbtpRunner) CheckCancel(c *hbtp.Context) {
	buildID := c.ReqHeader().GetString("buildID")
	_ = c.ResString(hbtp.ResStatusOk, fmt.Sprintf("%t", Mgr.brun.CheckCancel(buildID)))
}
func (HbtpRunner) Update(c *hbtp.Context, m *runners.UpdateJobInfo) {
	err := Mgr.brun.Update(m)
	if err != nil {
		_ = c.ResString(hbtp.ResStatusErr, err.Error())
		return
	}
	_ = c.ResString(hbtp.ResStatusOk, "ok")
}
func (HbtpRunner) UpdateCmd(c *hbtp.Context) {
	buildID := c.ReqHeader().GetString("buildID")
	jobID := c.ReqHeader().GetString("jobID")
	cmdID := c.ReqHeader().GetString("cmdID")
	fs, err := c.ReqHeader().GetInt("fs")
	code, _ := c.ReqHeader().GetInt("code")
	if err != nil {
		_ = c.ResString(hbtp.ResStatusErr, err.Error())
		return
	}
	err = Mgr.brun.UpdateCmd(buildID, jobID, cmdID, int(fs), int(code))
	if err != nil {
		_ = c.ResString(hbtp.ResStatusErr, err.Error())
		return
	}
	_ = c.ResString(hbtp.ResStatusOk, "ok")
}
func (HbtpRunner) PushOutLine(c *hbtp.Context) {
	buildID := c.ReqHeader().GetString("buildID")
	jobID := c.ReqHeader().GetString("jobID")
	cmdID := c.ReqHeader().GetString("cmdID")
	bs := c.ReqHeader().GetString("bs")
	iserr := c.ReqHeader().GetBool("iserr")
	err := Mgr.brun.PushOutLine(buildID, jobID, cmdID, bs, iserr)
	if err != nil {
		_ = c.ResString(hbtp.ResStatusErr, err.Error())
		return
	}
	_ = c.ResString(hbtp.ResStatusOk, "ok")
}
func (HbtpRunner) FindJobId(c *hbtp.Context) {
	buildID := c.ReqHeader().GetString("buildID")
	stgNm := c.ReqHeader().GetString("stgNm")
	stpNm := c.ReqHeader().GetString("stpNm")
	rts, ok := Mgr.brun.FindJobId(buildID, stgNm, stpNm)
	if !ok {
		_ = c.ResString(hbtp.ResStatusNotFound, "")
		return
	}
	_ = c.ResString(hbtp.ResStatusOk, rts)
}
func (HbtpRunner) ReadDir(c *hbtp.Context) {
	buildID := c.ReqHeader().GetString("buildID")
	pth := c.ReqHeader().GetString("pth")
	fs, err := c.ReqHeader().GetInt("fs")
	if err != nil {
		_ = c.ResString(hbtp.ResStatusErr, err.Error())
		return
	}
	rts, err := Mgr.brun.ReadDir(int(fs), buildID, pth)
	if err != nil {
		_ = c.ResString(hbtp.ResStatusErr, err.Error())
		return
	}
	_ = c.ResJson(hbtp.ResStatusOk, rts)
}
func (HbtpRunner) ReadFile(c *hbtp.Context) {
	buildID := c.ReqHeader().GetString("buildID")
	pth := c.ReqHeader().GetString("pth")
	fs, err := c.ReqHeader().GetInt("fs")
	start, _ := c.ReqHeader().GetInt("start")
	if err != nil {
		_ = c.ResString(hbtp.ResStatusErr, err.Error())
		return
	}
	flsz, flr, err := Mgr.brun.ReadFile(int(fs), buildID, pth, start)
	if err != nil {
		_ = c.ResString(hbtp.ResStatusErr, err.Error())
		return
	}
	defer func() { _ = flr.Close() }()
	_ = c.ResString(hbtp.ResStatusOk, fmt.Sprintf("%d", flsz))
	bts := make([]byte, 10240)
	for !hbtp.EndContext(comm.Ctx) {
		n, readErr := flr.Read(bts)
		if n <= 0 {
			break
		}
		if readErr != nil {
			break
		}
		_, err := c.Conn().Write(bts[:n])
		if err != nil {
			break
		}
	}
}
func (HbtpRunner) GetEnv(c *hbtp.Context) {
	buildID := c.ReqHeader().GetString("buildID")
	jobID := c.ReqHeader().GetString("jobID")
	key := c.ReqHeader().GetString("key")
	rts, ok := Mgr.brun.GetEnv(buildID, jobID, key)
	if !ok {
		_ = c.ResString(hbtp.ResStatusNotFound, "")
		return
	}
	_ = c.ResString(hbtp.ResStatusOk, rts)
}
func (HbtpRunner) GenEnv(c *hbtp.Context, env utils.EnvVal) {
	buildID := c.ReqHeader().GetString("buildID")
	jobID := c.ReqHeader().GetString("jobID")
	err := Mgr.brun.GenEnv(buildID, jobID, env)
	if err != nil {
		_ = c.ResString(hbtp.ResStatusErr, err.Error())
		return
	}
	_ = c.ResString(hbtp.ResStatusOk, "ok")
}
func (HbtpRunner) StatFile(c *hbtp.Context) {
	buildID := c.ReqHeader().GetString("buildID")
	jobID := c.ReqHeader().GetString("jobID")
	dir := c.ReqHeader().GetString("dir")
	pth := c.ReqHeader().GetString("pth")
	fs, err := c.ReqHeader().GetInt("fs")
	if err != nil {
		_ = c.ResString(hbtp.ResStatusErr, err.Error())
		return
	}
	stat, err := Mgr.brun.StatFile(int(fs), buildID, jobID, dir, pth)
	if err != nil {
		_ = c.ResString(hbtp.ResStatusErr, err.Error())
		return
	}
	_ = c.ResJson(hbtp.ResStatusOk, stat)
}
func (HbtpRunner) UploadFile(c *hbtp.Context) {
	buildID := c.ReqHeader().GetString("buildID")
	jobID := c.ReqHeader().GetString("jobID")
	dir := c.ReqHeader().GetString("dir")
	pth := c.ReqHeader().GetString("pth")
	start, _ := c.ReqHeader().GetInt("start")
	fs, err := c.ReqHeader().GetInt("fs")
	if err != nil {
		_ = c.ResString(hbtp.ResStatusErr, err.Error())
		return
	}
	flw, err := Mgr.brun.UploadFile(int(fs), buildID, jobID, dir, pth, start)
	if err != nil {
		_ = c.ResString(hbtp.ResStatusErr, err.Error())
		return
	}
	defer func() { _ = flw.Close() }()
	_ = c.ResString(hbtp.ResStatusOk, "ok")

	bts := make([]byte, 10240)
	for !hbtp.EndContext(comm.Ctx) {
		n, readErr := c.Conn().Read(bts)
		if n <= 0 {
			break
		}
		if readErr != nil {
			break
		}
		_, err := flw.Write(bts[:n])
		if err != nil {
			break
		}
	}
}
func (HbtpRunner) FindArtVersionId(c *hbtp.Context) {
	buildID := c.ReqHeader().GetString("buildID")
	idnt := c.ReqHeader().GetString("idnt")
	name := c.ReqHeader().GetString("name")
	rts, err := Mgr.brun.FindArtVersionId(buildID, idnt, name)
	if err != nil {
		_ = c.ResString(hbtp.ResStatusErr, err.Error())
		return
	}
	_ = c.ResString(hbtp.ResStatusOk, rts)
}
func (HbtpRunner) NewArtVersionId(c *hbtp.Context) {
	buildID := c.ReqHeader().GetString("buildID")
	idnt := c.ReqHeader().GetString("idnt")
	name := c.ReqHeader().GetString("name")
	rts, err := Mgr.brun.NewArtVersionId(buildID, idnt, name)
	if err != nil {
		_ = c.ResString(hbtp.ResStatusErr, err.Error())
		return
	}
	_ = c.ResString(hbtp.ResStatusOk, rts)
}
