package model

import (
	"time"
)

type TPipeline struct {
	Id           string    `xorm:"not null pk VARCHAR(64)" json:"id"`
	Uid          string    `xorm:"index(idx_pipeline_uid) VARCHAR(64)" json:"uid"`
	Name         string    `xorm:"VARCHAR(255)" json:"name"`
	DisplayName  string    `xorm:"VARCHAR(255)" json:"displayName"`
	PipelineType string    `xorm:"VARCHAR(255)" json:"pipelineType"`
	AccessToken  string    `xorm:"-" json:"accessToken"`
	Url          string    `xorm:"-" json:"url"`
	Username     string    `xorm:"-" json:"username"`
	Deleted      int       `xorm:"index(idx_pipeline_deleted) default 0 INT(1)" json:"-"`
	DeletedTime  time.Time `xorm:"DATETIME" json:"-"`
	Created      time.Time `xorm:"DATETIME" json:"-"`

	Nick    string    `xorm:"-" json:"nick"`
	Avat    string    `xorm:"-" json:"avat"`
	Buildln int64     `xorm:"-" json:"buildln"`
	Build   *RunBuild `xorm:"-" json:"build"`
}
