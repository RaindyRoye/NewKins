package engine

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gokins/gokins/service"

	"github.com/gokins/core/common"
	"github.com/gokins/core/utils"
	"github.com/gokins/gokins/bean"
	"github.com/gokins/gokins/comm"
	"github.com/gokins/gokins/model"
	"github.com/gokins/runner/runners"
	hbtp "github.com/mgr9525/HyperByte-Transfer-Protocol"
)

type baseRunner struct{}

func (c *baseRunner) ServerInfo() (*runners.ServerInfo, error) {
	return &runners.ServerInfo{
		WebHost:   comm.Cfg.Server.Host,
		DownToken: comm.Cfg.Server.DownToken,
	}, nil
}

func (c *baseRunner) PullJob(name string, plugs []string) (*runners.RunJob, error) {
	tms := time.Now()
	for time.Since(tms).Seconds() < 5 {
		v := Mgr.jobEgn.Pull(name, plugs)
		if v != nil {
			return v, nil
		}
		time.Sleep(time.Millisecond * 100)
	}
	return nil, fmt.Errorf("no job found for runner %q with plugins %v after 5s", name, plugs)
}
func (c *baseRunner) CheckCancel(buildID string) bool {
	v, ok := Mgr.buildEgn.Get(buildID)
	if !ok {
		return true
	}
	return v.stopd()
}
func (c *baseRunner) Update(m *runners.UpdateJobInfo) error {
	tsk, ok := Mgr.buildEgn.Get(m.BuildId)
	if !ok {
		return fmt.Errorf("update: build %q not found", m.BuildId)
	}
	job, ok := tsk.GetJob(m.JobId)
	if !ok {
		return fmt.Errorf("update: job %q not found in build %q", m.JobId, m.BuildId)
	}
	tsk.UpJob(job, m.Status, m.Error, m.ExitCode)
	return nil
}

func (c *baseRunner) UpdateCmd(buildID, jobId, cmdId string, fs, code int) error {
	tsk, ok := Mgr.buildEgn.Get(buildID)
	if !ok {
		return fmt.Errorf("updateCmd: build %q not found", buildID)
	}
	job, ok := tsk.GetJob(jobId)
	if !ok {
		return fmt.Errorf("updateCmd: job %q not found in build %q", jobId, buildID)
	}
	job.RLock()
	cmd, ok := job.cmdmp[cmdId]
	job.RUnlock()
	if !ok {
		return fmt.Errorf("updateCmd: cmd %q not found in job %q", cmdId, jobId)
	}
	tsk.UpJobCmd(cmd, fs, code)
	return nil
}
func (c *baseRunner) PushOutLine(buildID, jobId, cmdId, bs string, iserr bool) error {
	tsk, ok := Mgr.buildEgn.Get(buildID)
	if !ok {
		return fmt.Errorf("pushOutLine: build %q not found", buildID)
	}
	job, ok := tsk.GetJob(jobId)
	if !ok {
		return fmt.Errorf("pushOutLine: job %q not found in build %q", jobId, buildID)
	}

	bts, err := json.Marshal(&bean.LogOutJson{
		Id:      cmdId,
		Content: bs,
		Times:   time.Now(),
		Errs:    iserr,
	})
	if err != nil {
		return fmt.Errorf("marshal log json: %w", err)
	}

	dir := filepath.Join(comm.WorkPath, common.PathBuild, job.step.BuildId, common.PathJobs, job.step.Id)
	logpth := filepath.Join(dir, "build.log")
	if err = os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("create log dir %s: %w", dir, err)
	}
	logfl, err := os.OpenFile(logpth, os.O_CREATE|os.O_APPEND|os.O_RDWR, 0644)
	if err != nil {
		return fmt.Errorf("open log file %s: %w", logpth, err)
	}
	defer func() { _ = logfl.Close() }()
	if _, err := logfl.Write(bts); err != nil {
		return fmt.Errorf("write log entry: %w", err)
	}
	if _, err := logfl.WriteString("\n"); err != nil {
		return fmt.Errorf("write log newline: %w", err)
	}
	return nil
}
func (c *baseRunner) FindJobId(buildID, stgNm, stpNm string) (string, bool) {
	if buildID == "" || stgNm == "" || stpNm == "" {
		return "", false
	}
	build, ok := Mgr.buildEgn.Get(buildID)
	if !ok {
		return "", false
	}
	build.staglk.RLock()
	defer build.staglk.RUnlock()
	stg, ok := build.stages[stgNm]
	if !ok {
		return "", false
	}
	stg.RLock()
	defer stg.RUnlock()
	for _, v := range stg.jobs {
		if v.step.Name == stpNm {
			return v.step.Id, true
		}
	}
	return "", false
}

func (c *baseRunner) ReadDir(fs int, buildID string, pth string) ([]*runners.DirEntry, error) {
	if buildID == "" || pth == "" {
		return nil, fmt.Errorf("readDir: buildID=%q and path=%q must not be empty", buildID, pth)
	}
	build, ok := Mgr.buildEgn.Get(buildID)
	if !ok {
		return nil, fmt.Errorf("readDir: build %q not found", buildID)
	}
	pths := ""
	switch fs {
	case 1:
		pths = filepath.Join(build.repoPaths, pth)
	case 2:
		pths = filepath.Join(comm.WorkPath, common.PathArtifacts, pth)
	case 3:
		pths = filepath.Join(build.buildPath, common.PathJobs, pth)
	}
	fls, err := os.ReadDir(pths)
	if err != nil {
		if build.repoPath == "" {
			return nil, nil
		}
		return nil, fmt.Errorf("read dir %s: %w", pths, err)
	}
	var ls []*runners.DirEntry
	for _, v := range fls {
		e := &runners.DirEntry{
			Name:  v.Name(),
			IsDir: v.IsDir(),
		}
		ifo, err := v.Info()
		if err == nil {
			e.Size = ifo.Size()
		}
		ls = append(ls, e)
	}
	return ls, nil
}
func (c *baseRunner) ReadFile(fs int, buildID string, pth string, start int64) (int64, io.ReadCloser, error) {
	if buildID == "" || pth == "" {
		return 0, nil, fmt.Errorf("readFile: buildID=%q and path=%q must not be empty", buildID, pth)
	}
	build, ok := Mgr.buildEgn.Get(buildID)
	if !ok {
		return 0, nil, fmt.Errorf("readFile: build %q not found", buildID)
	}
	pths := ""
	switch fs {
	case 1:
		pths = filepath.Join(build.repoPaths, pth)
	case 2:
		pths = filepath.Join(comm.WorkPath, common.PathArtifacts, pth)
	case 3:
		pths = filepath.Join(build.buildPath, common.PathJobs, pth)
	}
	if pths == "" {
		return 0, nil, errors.New("readFile: invalid filesystem type, no path resolved")
	}
	stat, err := os.Stat(pths)
	if err != nil {
		return 0, nil, fmt.Errorf("stat file %s: %w", pths, err)
	}
	fl, err := os.Open(pths)
	if err != nil {
		return 0, nil, fmt.Errorf("open file %s: %w", pths, err)
	}
	if start > 0 {
		if _, err := fl.Seek(start, io.SeekStart); err != nil {
			return 0, nil, fmt.Errorf("seek file to %d: %w", start, err)
		}
	}
	return stat.Size(), fl, nil
}

func (c *baseRunner) GetEnv(buildID, jobId, key string) (string, bool) {
	if jobId == "" || key == "" {
		return "", false
	}
	tsk, ok := Mgr.buildEgn.Get(buildID)
	if !ok {
		return "", false
	}
	job, ok := tsk.GetJob(jobId)
	if !ok {
		return "", false
	}
	dir := filepath.Join(comm.WorkPath, common.PathBuild, job.step.BuildId, common.PathJobs, job.step.Id)
	bts, err := os.ReadFile(filepath.Join(dir, "build.env"))
	if err != nil {
		return "", false
	}
	mp := hbtp.NewMaps(bts)
	v, ok := mp.Get(key)
	if !ok {
		return "", false
	}
	switch val := v.(type) {
	case string:
		return val, true
	}
	return fmt.Sprintf("%v", v), true
}
func (c *baseRunner) GenEnv(buildID, jobId string, env utils.EnvVal) error {
	if jobId == "" || env == nil {
		return fmt.Errorf("genEnv: jobId=%q must not be empty and env must not be nil", jobId)
	}
	tsk, ok := Mgr.buildEgn.Get(buildID)
	if !ok {
		return fmt.Errorf("genEnv: build %q not found", buildID)
	}
	job, ok := tsk.GetJob(jobId)
	if !ok {
		return fmt.Errorf("genEnv: job %q not found in build %q", jobId, buildID)
	}
	bts, err := json.Marshal(env)
	if err != nil {
		return fmt.Errorf("marshal env json: %w", err)
	}
	dir := filepath.Join(comm.WorkPath, common.PathBuild, job.step.BuildId, common.PathJobs, job.step.Id)
	err = os.WriteFile(filepath.Join(dir, "build.env"), bts, 0640)
	if err != nil {
		return fmt.Errorf("write env file: %w", err)
	}
	return nil
}

func (c *baseRunner) StatFile(fs int, buildID, jobId string, dir, pth string) (*runners.FileStat, error) {
	if jobId == "" || pth == "" {
		return nil, fmt.Errorf("statFile: jobId=%q and path=%q must not be empty", jobId, pth)
	}
	build, ok := Mgr.buildEgn.Get(buildID)
	if !ok {
		return nil, fmt.Errorf("statFile: build %q not found", buildID)
	}
	job, ok := build.GetJob(jobId)
	if !ok {
		return nil, fmt.Errorf("statFile: job %q not found in build %q", jobId, buildID)
	}
	pths := ""
	switch fs {
	case 1:
		pths = filepath.Join(comm.WorkPath, common.PathArtifacts, dir, pth)
	case 2:
		pths = filepath.Join(job.task.buildPath, common.PathJobs, job.step.Id, common.PathArts, dir, pth)
	case 3:
		pths = filepath.Join(build.repoPaths, dir, pth)
	}
	if pths == "" {
		return nil, errors.New("statFile: invalid filesystem type, no path resolved")
	}
	stat, err := os.Stat(pths)
	if err != nil {
		return nil, fmt.Errorf("stat file %s: %w", pths, err)
	}
	return &runners.FileStat{
		Name:  stat.Name(),
		IsDir: stat.IsDir(),
		Size:  stat.Size(),
	}, err
}
func (c *baseRunner) UploadFile(fs int, buildID, jobId string, dir, pth string, start int64) (io.WriteCloser, error) {
	if jobId == "" || pth == "" {
		return nil, fmt.Errorf("uploadFile: jobId=%q and path=%q must not be empty", jobId, pth)
	}
	build, ok := Mgr.buildEgn.Get(buildID)
	if !ok {
		return nil, fmt.Errorf("uploadFile: build %q not found", buildID)
	}
	job, ok := build.GetJob(jobId)
	if !ok {
		return nil, fmt.Errorf("uploadFile: job %q not found in build %q", jobId, buildID)
	}
	pths := ""
	switch fs {
	case 1:
		pths = filepath.Join(comm.WorkPath, common.PathArtifacts, dir, pth)
	case 2:
		pths = filepath.Join(job.task.buildPath, common.PathJobs, job.step.Id, common.PathArts, dir, pth)
	case 3:
		pths = filepath.Join(build.repoPaths, dir, pth)
	}
	if pths == "" {
		return nil, errors.New("uploadFile: invalid filesystem type, no path resolved")
	}
	dirs := filepath.Dir(pths)
	if err := os.MkdirAll(dirs, 0750); err != nil {
		return nil, fmt.Errorf("create upload dirs %s: %w", dirs, err)
	}
	fl, err := os.OpenFile(pths, os.O_CREATE|os.O_RDWR, 0640)
	if err != nil {
		return nil, fmt.Errorf("open upload file %s: %w", pths, err)
	}
	if start > 0 {
		if _, err := fl.Seek(start, io.SeekStart); err != nil {
			_ = fl.Close()
			return nil, fmt.Errorf("seek upload file to %d: %w", start, err)
		}
	}
	return fl, nil
}
func (c *baseRunner) FindArtVersionId(buildID, idnt string, names string) (string, error) {
	tnms := strings.Split(strings.TrimSpace(names), "@")
	name := tnms[0]
	vers := ""
	if len(tnms) > 1 {
		vers = tnms[1]
	}
	if buildID == "" || idnt == "" || name == "" {
		return "", fmt.Errorf("findArtVersionId: buildID=%q, identifier=%q, name=%q must not be empty", buildID, idnt, name)
	}
	build, ok := Mgr.buildEgn.Get(buildID)
	if !ok {
		return "", fmt.Errorf("findArtVersionId: build %q not found", buildID)
	}

	arty := &model.TArtifactory{}
	ok, _ = comm.Db.Where("deleted!=1 and identifier=? and org_id in (select org_id from t_org_pipe where pipe_id=?)",
		idnt, build.build.PipelineId).Get(arty)
	if !ok {
		return "", fmt.Errorf("findArtVersionId: artifactory %q not found", idnt)
	}

	pv := &model.TPipelineVersion{}
	ok = service.GetIdOrAid(build.build.PipelineVersionId, pv)
	if !ok {
		return "", fmt.Errorf("findArtVersionId: pipeline version %q not found", build.build.PipelineVersionId)
	}
	usr := &model.TUser{}
	ok = service.GetIdOrAid(pv.Uid, usr)
	if !ok {
		return "", fmt.Errorf("findArtVersionId: user %q not found", pv.Uid)
	}
	perm := service.NewOrgPerm(usr, arty.OrgId)
	if !perm.CanExec() {
		return "", fmt.Errorf("user put '%s' no permission", idnt)
	}

	artp := &model.TArtifactPackage{}
	ok, _ = comm.Db.Where("deleted!=1 and repo_id=? and name=?", arty.Id, name).Get(artp)
	if !ok {
		return "", fmt.Errorf("not found artifact '%s'", names)
	}
	artv := &model.TArtifactVersion{}
	ses := comm.Db.Where("package_id=?", artp.Id)
	if vers != "" {
		ses.And("version=? or sha=?", vers)
	}
	ok, _ = ses.OrderBy("aid DESC").Get(artv)
	if !ok {
		return "", fmt.Errorf("not found artifacts '%s'", names)
	}
	return artv.Id, nil
}
func (c *baseRunner) NewArtVersionId(buildID, idnt string, name string) (string, error) {
	name = strings.Split(strings.TrimSpace(name), "@")[0]
	if buildID == "" || idnt == "" || name == "" {
		return "", fmt.Errorf("newArtVersionId: buildID=%q, identifier=%q, name=%q must not be empty", buildID, idnt, name)
	}
	build, ok := Mgr.buildEgn.Get(buildID)
	if !ok {
		return "", fmt.Errorf("newArtVersionId: build %q not found", buildID)
	}

	arty := &model.TArtifactory{}
	ok, _ = comm.Db.Where("deleted!=1 and identifier=? and org_id in (select org_id from t_org_pipe where pipe_id=?)",
		idnt, build.build.PipelineId).Get(arty)
	if !ok {
		return "", fmt.Errorf("newArtVersionId: artifactory %q not found", idnt)
	}
	if arty.Disabled == 1 {
		return "", fmt.Errorf("newArtVersionId: artifactory %q is disabled", idnt)
	}

	artp := &model.TArtifactPackage{}
	ok, _ = comm.Db.Where("deleted!=1 and repo_id=? and name=?", arty.Id, name).Get(artp)
	if !ok {
		artp.Id = utils.NewXid()
		artp.RepoId = arty.Id
		artp.Name = name
		artp.Created = time.Now()
		artp.Updated = time.Now()
		_, err := comm.Db.InsertOne(artp)
		if err != nil {
			return "", fmt.Errorf("insert artifact package: %w", err)
		}
	}
	artv := &model.TArtifactVersion{
		Id:        utils.NewXid(),
		RepoId:    arty.Id,
		PackageId: artp.Id,
		Name:      artp.Name,
		Preview:   1,
		Created:   time.Now(),
		Updated:   time.Now(),
	}
	artv.Sha = artv.Id
	_, err := comm.Db.InsertOne(artv)
	if err != nil {
		return "", fmt.Errorf("insert artifact version: %w", err)
	}
	return artv.Id, nil
}
