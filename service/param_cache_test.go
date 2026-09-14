package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/gokins/gokins/comm"
	"github.com/gokins/gokins/model"
	_ "github.com/mattn/go-sqlite3"
	"xorm.io/xorm"
)

func setupParamCacheTestDb(t *testing.T) *xorm.Engine {
	t.Helper()
	origDb := comm.Db
	t.Cleanup(func() { comm.Db = origDb })

	db, err := xorm.NewEngine("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("create sqlite engine: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	comm.Db = db

	_, err = db.Exec(`CREATE TABLE t_param (
		aid INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT,
		name VARCHAR(100),
		title VARCHAR(255),
		data TEXT,
		times DATETIME
	)`)
	if err != nil {
		t.Fatalf("create param table: %v", err)
	}

	return db
}

func TestGetsParamCacheCtx_DataNil(t *testing.T) {
	setupParamCacheTestDb(t)
	err := GetsParamCacheCtx(context.TODO(), "somekey", nil)
	if !errors.Is(err, ErrParamDataNil) {
		t.Errorf("expected wrapped ErrParamDataNil, got %v", err)
	}
}

func TestGetsParamCacheCtx_ParamNotFound(t *testing.T) {
	setupParamCacheTestDb(t)
	var data map[string]string
	err := GetsParamCacheCtx(context.TODO(), "nonexistent", &data)
	if err == nil {
		t.Error("expected error for nonexistent param, got nil")
	}
}

func TestGetsParamCacheCtx_Success(t *testing.T) {
	db := setupParamCacheTestDb(t)
	// Create a param with JSON data
	if _, err := db.Insert(&model.TParam{
		Name:  "testkey",
		Title: "Test Param",
		Data:  `{"foo":"bar","num":42}`,
		Times: time.Now(),
	}); err != nil {
		t.Fatalf("insert param: %v", err)
	}

	var data map[string]interface{}
	err := GetsParamCacheCtx(context.TODO(), "testkey", &data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if data["foo"] != "bar" {
		t.Errorf("expected foo=bar, got %v", data["foo"])
	}
	if data["num"].(float64) != 42 {
		t.Errorf("expected num=42, got %v", data["num"])
	}
}

func TestGetsParamCache_GlobalWrapper(t *testing.T) {
	db := setupParamCacheTestDb(t)
	// Initialize global context
	comm.ResetCtx()
	if _, err := db.Insert(&model.TParam{
		Name:  "globalkey",
		Title: "Global",
		Data:  `{"value":"test"}`,
		Times: time.Now(),
	}); err != nil {
		t.Fatalf("insert param: %v", err)
	}

	var data map[string]string
	err := GetsParamCache("globalkey", &data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if data["value"] != "test" {
		t.Errorf("expected value=test, got %s", data["value"])
	}
}
