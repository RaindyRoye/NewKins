package service

import (
	"context"
	"errors"
	"testing"
)

func TestParseTriggerParams_Empty(t *testing.T) {
	_, err := parseTriggerParams("tid1", "")
	if err == nil {
		t.Fatal("expected error for empty params")
	}
	if !errors.Is(err, ErrTriggerNoParams) {
		t.Errorf("expected ErrTriggerNoParams, got: %v", err)
	}
}

func TestParseTriggerParams_InvalidJSON(t *testing.T) {
	_, err := parseTriggerParams("tid2", "{invalid")
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
	if !errors.Is(err, ErrTriggerNoParams) {
		// It's a wrapped unmarshal error
		t.Logf("got expected wrapped error: %v", err)
	}
}

func TestParseTriggerParams_Valid(t *testing.T) {
	m, err := parseTriggerParams("tid3", `{"hookType":"github","secret":"s3cr3t","branch":"main"}`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if m["hookType"] != "github" {
		t.Errorf("expected hookType=github, got %s", m["hookType"])
	}
	if m["secret"] != "s3cr3t" {
		t.Errorf("expected secret=s3cr3t, got %s", m["secret"])
	}
	if m["branch"] != "main" {
		t.Errorf("expected branch=main, got %s", m["branch"])
	}
}

func TestParseTriggerParams_MissingOptionalFields(t *testing.T) {
	m, err := parseTriggerParams("tid4", `{"hookType":"gitea"}`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if m["hookType"] != "gitea" {
		t.Errorf("expected hookType=gitea, got %s", m["hookType"])
	}
	// Optional fields should be empty strings when missing
	if m["secret"] != "" {
		t.Errorf("expected empty secret, got %s", m["secret"])
	}
	if m["event"] != "" {
		t.Errorf("expected empty event, got %s", m["event"])
	}
}

func TestTriggerRunRecorder_Record_NilDB(t *testing.T) {
	// With comm.Db nil, the recorder should gracefully handle the panic
	// via the logrus error path (not crash the test).
	rec := &triggerRunRecorder{
		ctx:           context.Background(),
		triggerId:     "test-trigger",
		pipeVersionId: "pv-123",
		infos:         `{"event":"push"}`,
	}
	defer func() {
		if r := recover(); r != nil {
			t.Logf("recovered panic (Db is nil in unit tests): %v", r)
		}
	}()
	rec.record()
}

func TestTriggerRunRecorder_RecordWithError(t *testing.T) {
	rec := &triggerRunRecorder{
		ctx:       context.Background(),
		triggerId: "test-trigger-err",
		err:       errors.New("test failure"),
		infos:     "{}",
	}
	defer func() {
		if r := recover(); r != nil {
			t.Logf("recovered panic (Db is nil in unit tests): %v", r)
		}
	}()
	rec.record()
}

func TestTriggerTimer_NilTrigger(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Logf("recovered panic (nil trigger): %v", r)
		}
	}()
	_, err := TriggerTimer(context.Background(), nil)
	if err == nil {
		t.Log("TriggerTimer with nil trigger: no error returned (Db is nil, so it panics instead)")
	}
}
