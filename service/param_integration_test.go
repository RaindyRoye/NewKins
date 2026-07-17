package service

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/gokins/gokins/comm"
	"github.com/gokins/gokins/model"
	_ "github.com/mattn/go-sqlite3"
	"xorm.io/xorm"
)

func setupParamTestDB(t *testing.T) {
	t.Helper()
	origDb := comm.Db
	origCtx := comm.Ctx
	t.Cleanup(func() {
		comm.Db = origDb
		comm.Ctx = origCtx
	})

	db, err := xorm.NewEngine("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("failed to init test DB: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	_, err = db.Exec(`CREATE TABLE t_param (
		aid INTEGER PRIMARY KEY AUTOINCREMENT,
		name VARCHAR(255) NOT NULL,
		title VARCHAR(255),
		data TEXT,
		times DATETIME
	)`)
	if err != nil {
		t.Fatalf("failed to create t_param table: %v", err)
	}

	comm.Db = db
	comm.Ctx = context.Background()
}

func TestFindParamCtx_Found(t *testing.T) {
	setupParamTestDB(t)
	// Insert a param
	_, err := comm.Db.Insert(&model.TParam{
		Name:  "test-key",
		Title: "Test Title",
		Data:  `{"hello":"world"}`,
		Times: time.Now(),
	})
	if err != nil {
		t.Fatalf("insert param: %v", err)
	}

	p, ok := FindParamCtx(context.Background(), "test-key")
	if !ok {
		t.Fatal("FindParamCtx should find the param")
	}
	if p.Name != "test-key" {
		t.Errorf("param Name = %q, want %q", p.Name, "test-key")
	}
	if p.Title != "Test Title" {
		t.Errorf("param Title = %q, want %q", p.Title, "Test Title")
	}
}

func TestFindParamCtx_NotFound(t *testing.T) {
	setupParamTestDB(t)
	_, ok := FindParamCtx(context.Background(), "nonexistent")
	if ok {
		t.Error("FindParamCtx should not find nonexistent param")
	}
}

func TestSetParamCtx_Create(t *testing.T) {
	setupParamTestDB(t)
	err := SetParamCtx(context.Background(), "new-key", []byte("some data"), "New Title")
	if err != nil {
		t.Fatalf("SetParamCtx create: %v", err)
	}

	p, ok := FindParamCtx(context.Background(), "new-key")
	if !ok {
		t.Fatal("param should exist after create")
	}
	if p.Data != "some data" {
		t.Errorf("param Data = %q, want %q", p.Data, "some data")
	}
	if p.Title != "New Title" {
		t.Errorf("param Title = %q, want %q", p.Title, "New Title")
	}
}

func TestSetParamCtx_Update(t *testing.T) {
	setupParamTestDB(t)
	// Create first
	err := SetParamCtx(context.Background(), "upd-key", []byte("initial"), "Initial")
	if err != nil {
		t.Fatalf("SetParamCtx create: %v", err)
	}
	// Update
	err = SetParamCtx(context.Background(), "upd-key", []byte("updated"), "Updated")
	if err != nil {
		t.Fatalf("SetParamCtx update: %v", err)
	}

	p, ok := FindParamCtx(context.Background(), "upd-key")
	if !ok {
		t.Fatal("param should exist after update")
	}
	if p.Data != "updated" {
		t.Errorf("param Data = %q, want %q", p.Data, "updated")
	}
	if p.Title != "Updated" {
		t.Errorf("param Title = %q, want %q", p.Title, "Updated")
	}
}

func TestSetsParamCtx_Success(t *testing.T) {
	setupParamTestDB(t)
	data := map[string]string{"key": "value"}
	err := SetsParamCtx(context.Background(), "json-key", data, "JSON")
	if err != nil {
		t.Fatalf("SetsParamCtx: %v", err)
	}

	p, ok := FindParamCtx(context.Background(), "json-key")
	if !ok {
		t.Fatal("param should exist")
	}
	var result map[string]string
	if err := json.Unmarshal([]byte(p.Data), &result); err != nil {
		t.Fatalf("unmarshal param data: %v", err)
	}
	if result["key"] != "value" {
		t.Errorf("result[key] = %q, want %q", result["key"], "value")
	}
}

func TestGetParamCtx_Found(t *testing.T) {
	setupParamTestDB(t)
	err := SetParamCtx(context.Background(), "raw-key", []byte("raw data"))
	if err != nil {
		t.Fatalf("SetParamCtx: %v", err)
	}

	data, err := GetParamCtx(context.Background(), "raw-key")
	if err != nil {
		t.Fatalf("GetParamCtx: %v", err)
	}
	if string(data) != "raw data" {
		t.Errorf("GetParamCtx data = %q, want %q", string(data), "raw data")
	}
}

func TestGetParamCtx_NotFound(t *testing.T) {
	setupParamTestDB(t)
	_, err := GetParamCtx(context.Background(), "missing")
	if err == nil {
		t.Fatal("GetParamCtx should return error for missing param")
	}
}

func TestGetsParamCtx_Success(t *testing.T) {
	setupParamTestDB(t)
	type myData struct {
		Foo string `json:"foo"`
		Bar int    `json:"bar"`
	}
	original := myData{Foo: "hello", Bar: 42}
	err := SetsParamCtx(context.Background(), "struct-key", original)
	if err != nil {
		t.Fatalf("SetsParamCtx: %v", err)
	}

	var loaded myData
	err = GetsParamCtx(context.Background(), "struct-key", &loaded)
	if err != nil {
		t.Fatalf("GetsParamCtx: %v", err)
	}
	if loaded.Foo != "hello" || loaded.Bar != 42 {
		t.Errorf("GetsParamCtx loaded = %+v, want %+v", loaded, original)
	}
}

func TestGetsParamCtx_NotFound(t *testing.T) {
	setupParamTestDB(t)
	var data map[string]string
	err := GetsParamCtx(context.Background(), "nope", &data)
	if err == nil {
		t.Fatal("GetsParamCtx should return error for missing param")
	}
}
