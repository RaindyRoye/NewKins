package model

import (
	"time"
)

type TPipelineInfo struct {
	Id           string    `xorm:"not null pk VARCHAR(64)" json:"id"`
	Uid          string    `xorm:"VARCHAR(64)" json:"uid"`
	Name         string    `xorm:"VARCHAR(255)" json:"name"`
	DisplayName  string    `xorm:"VARCHAR(255)" json:"displayName"`
	PipelineType string    `xorm:"VARCHAR(255)" json:"pipelineType"`
	YmlContent   string    `xorm:"-" json:"ymlContent"`
	AccessToken  string    `xorm:"-" json:"accessToken"`
	Url          string    `xorm:"-" json:"url"`
	Username     string    `xorm:"-" json:"username"`
	Created      time.Time `xorm:"DATETIME" json:"created"`
}

func (TPipelineInfo) TableName() string {
	return "t_pipeline"
}
