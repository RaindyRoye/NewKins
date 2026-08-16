package model

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/gokins/core/common"
	"github.com/gokins/gokins/comm"
	hbtp "github.com/mgr9525/HyperByte-Transfer-Protocol"
)

type TArtifactVersion struct {
	Id        string    `xorm:"not null pk VARCHAR(64)" json:"id"`
	Aid       int64     `xorm:"not null pk autoincr BIGINT(20)" json:"aid"`
	RepoId    string    `xorm:"index(rpnm) VARCHAR(64)" json:"repoId"`
	PackageId string    `xorm:"index(idx_artver_package_id) VARCHAR(64)" json:"packageId"`
	Name      string    `xorm:"index(rpnm) VARCHAR(100)" json:"name"`
	Version   string    `xorm:"VARCHAR(100)" json:"version"`
	Sha       string    `xorm:"VARCHAR(100)" json:"sha"`
	Desc      string    `xorm:"VARCHAR(500)" json:"desc"`
	Preview   int       `xorm:"INT(1)" json:"preview"`
	Created   time.Time `xorm:"DATETIME" json:"created"`
	Updated   time.Time `xorm:"DATETIME" json:"updated"`

	Files []hbtp.Map `xorm:"-" json:"files"`
}

func (c *TArtifactVersion) ReadFiles() error {
	dir := filepath.Join(comm.WorkPath, common.PathArtifacts, c.Id)
	fls, err := c.readDir(dir)
	c.Files = fls
	if err != nil {
		return fmt.Errorf("read artifact files %q: %w", c.Id, err)
	}
	return nil
}
func (c *TArtifactVersion) readDir(pth string) ([]hbtp.Map, error) {
	var rts []hbtp.Map
	fls, err := os.ReadDir(pth)
	if err != nil {
		return nil, fmt.Errorf("read artifact dir %s: %w", pth, err)
	}
	for _, v := range fls {
		if v.IsDir() {
			fls, err := c.readDir(filepath.Join(pth, v.Name()))
			if err != nil {
				return nil, fmt.Errorf("read artifact subdir %q: %w", v.Name(), err)
			}
			var chd []hbtp.Map
			chd = append(chd, fls...)
			rts = append(rts, hbtp.Map{
				"name":  v.Name(),
				"dir":   true,
				"size":  0,
				"child": chd,
			})
		} else {
			info, _ := v.Info()
			rts = append(rts, hbtp.Map{
				"name": v.Name(),
				"dir":  false,
				"size": info.Size(),
			})
		}
	}
	return rts, nil
}
