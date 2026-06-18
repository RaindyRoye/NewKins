package model

import (
	"testing"
)

func TestTTrigger_Param(t *testing.T) {
	tr := &TTrigger{
		Id:    "test-1",
		Name:  "test trigger",
		Param: map[string]any{"key": "value"},
	}
	if tr.Param["key"] != "value" {
		t.Error("Param map not set correctly")
	}
}

func TestTTriggerRun_Fields(t *testing.T) {
	tr := &TTriggerRun{
		Id:                  "run-1",
		Number:              42,
		PipelineName:        "test-pipeline",
		PipelineDisplayName: "Test Pipeline",
		BStatus:             "success",
	}
	if tr.Number != 42 {
		t.Errorf("expected Number 42, got %d", tr.Number)
	}
	if tr.PipelineName != "test-pipeline" {
		t.Errorf("expected PipelineName 'test-pipeline', got %s", tr.PipelineName)
	}
	if tr.BStatus != "success" {
		t.Errorf("expected BStatus 'success', got %s", tr.BStatus)
	}
}

func TestRunBuild_TableName(t *testing.T) {
	rb := RunBuild{}
	if rb.TableName() != "t_build" {
		t.Errorf("expected TableName 't_build', got %s", rb.TableName())
	}
}

func TestRunStage_TableName(t *testing.T) {
	rs := RunStage{}
	if rs.TableName() != "t_stage" {
		t.Errorf("expected TableName 't_stage', got %s", rs.TableName())
	}
}

func TestRunStep_TableName(t *testing.T) {
	rst := RunStep{}
	if rst.TableName() != "t_step" {
		t.Errorf("expected TableName 't_step', got %s", rst.TableName())
	}
}

func TestTOrgInfo_TableName(t *testing.T) {
	oi := TOrgInfo{}
	if oi.TableName() != "t_org" {
		t.Errorf("expected TableName 't_org', got %s", oi.TableName())
	}
}

func TestTPipelineInfo_TableName(t *testing.T) {
	pi := TPipelineInfo{}
	if pi.TableName() != "t_pipeline" {
		t.Errorf("expected TableName 't_pipeline', got %s", pi.TableName())
	}
}

func TestTUserOrgInfo_TableName(t *testing.T) {
	uoi := TUserOrgInfo{}
	if uoi.TableName() != "t_user" {
		t.Errorf("expected TableName 't_user', got %s", uoi.TableName())
	}
}
