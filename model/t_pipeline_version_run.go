package model

type RunPipelineVersion struct {
	Id                  string `xorm:"not null pk VARCHAR(64)" json:"id"`
	Number              int64  `xorm:"comment('构建次数') BIGINT(20)" json:"number"`
	PipelineName        string `xorm:"VARCHAR(255)" json:"pipelineName"`
	PipelineDisplayName string `xorm:"VARCHAR(255)" json:"pipelineDisplayName"`
	//Build
	Status string `xorm:"comment('构建状态') VARCHAR(100)" json:"status"`
}
