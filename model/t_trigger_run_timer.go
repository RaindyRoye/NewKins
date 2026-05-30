package model

import (
	"time"
)

type TimerTriggerRun struct {
	Id         string `xorm:"not null pk VARCHAR(64)" json:"id"`
	Uid        string `xorm:"index VARCHAR(64)" json:"uid"`
	PipelineId string `xorm:"VARCHAR(64)" json:"pipelineId"`
	Types      string `xorm:"VARCHAR(50)" json:"types"`
	Name       string `xorm:"VARCHAR(100)" json:"name"`
	Params     string `xorm:"JSON" json:"params"`
	Enabled    int    `xorm:"INT(1)" json:"enabled"`
	// run
	RunCreated time.Time `xorm:"DATETIME" json:"created"`
}
