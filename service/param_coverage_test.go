package service

import (
	"context"
	"testing"

	"github.com/gokins/gokins/comm"
	"github.com/gokins/gokins/model"
	_ "github.com/mattn/go-sqlite3"
	"xorm.io/xorm"
)

func setupParamTestDB(t *testing.T) *xorm.Engine {
	t.Helper()
	eng, err := xorm.NewEngine("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("failed to create test database: %v", err)
	}

	oldDb := comm.Db
	comm.Db = eng
	t.Cleanup(func() {
		comm.Db = oldDb
		_ = eng.Close()
	})

	err = eng.Sync2(&model.TParam{})
	if err != nil {
		t.Fatalf("failed to sync database schema: %v", err)
	}

	return eng
}

func TestFindParamCtx_Found(t *testing.T) {
	eng := setupParamTestDB(t)
	ctx := context.Background()

	// Insert test param
	param := &model.TParam{
		Name:  "test-param",
		Data:  "test-data",
		Title: "Test Param",
	}
	if _, err := eng.Insert(param); err != nil {
		t.Fatalf("failed to insert param: %v", err)
	}

	// Test lookup
	result, ok := FindParamCtx(ctx, "test-param")
	if !ok {
		t.Error("FindParamCtx should find param")
	}
	if result.Name != "test-param" {
		t.Errorf("FindParamCtx found wrong param: name = %q, want %q", result.Name, "test-param")
	}
	if result.Data != "test-data" {
		t.Errorf("FindParamCtx found wrong data: data = %q, want %q", result.Data, "test-data")
	}
}

func TestFindParamCtx_NotFound(t *testing.T) {
	setupParamTestDB(t)
	ctx := context.Background()

	_, ok := FindParamCtx(ctx, "nonexistent")
	if ok {
		t.Error("FindParamCtx should return false for nonexistent param")
	}
}

func TestFindParam_Wrapper(t *testing.T) {
	eng := setupParamTestDB(t)

	// Insert test param
	param := &model.TParam{
		Name:  "wrapper-param",
		Data:  "wrapper-data",
	}
	if _, err := eng.Insert(param); err != nil {
		t.Fatalf("failed to insert param: %v", err)
	}

	// Test wrapper
	result, ok := FindParam("wrapper-param")
	if !ok {
		t.Error("FindParam should find param")
	}
	if result.Name != "wrapper-param" {
		t.Errorf("FindParam found wrong param: name = %q, want %q", result.Name, "wrapper-param")
	}
}

func TestSetParamCtx_Create(t *testing.T) {
	setupParamTestDB(t)
	ctx := context.Background()

	// Create new param
	err := SetParamCtx(ctx, "new-param", []byte("new-data"), "New Param")
	if err != nil {
		t.Fatalf("SetParamCtx error: %v", err)
	}

	// Verify it was created
	result, ok := FindParamCtx(ctx, "new-param")
	if !ok {
		t.Fatal("SetParamCtx should create param")
	}
	if result.Data != "new-data" {
		t.Errorf("SetParamCtx created wrong data: data = %q, want %q", result.Data, "new-data")
	}
	if result.Title != "New Param" {
		t.Errorf("SetParamCtx created wrong title: title = %q, want %q", result.Title, "New Param")
	}
}

func TestSetParamCtx_Update(t *testing.T) {
	eng := setupParamTestDB(t)
	ctx := context.Background()

	// Insert initial param
	param := &model.TParam{
		Name:  "update-param",
		Data:  "old-data",
		Title: "Old Title",
	}
	if _, err := eng.Insert(param); err != nil {
		t.Fatalf("failed to insert param: %v", err)
	}

	// Update param
	err := SetParamCtx(ctx, "update-param", []byte("updated-data"), "Updated Title")
	if err != nil {
		t.Fatalf("SetParamCtx error: %v", err)
	}

	// Verify it was updated
	result, ok := FindParamCtx(ctx, "update-param")
	if !ok {
		t.Fatal("SetParamCtx should update param")
	}
	if result.Data != "updated-data" {
		t.Errorf("SetParamCtx updated wrong data: data = %q, want %q", result.Data, "updated-data")
	}
	if result.Title != "Updated Title" {
		t.Errorf("SetParamCtx updated wrong title: title = %q, want %q", result.Title, "Updated Title")
	}
}

func TestSetParam_Wrapper(t *testing.T) {
	setupParamTestDB(t)

	// Test wrapper
	err := SetParam("wrapper-set", []byte("wrapper-data"))
	if err != nil {
		t.Fatalf("SetParam error: %v", err)
	}

	// Verify
	result, ok := FindParam("wrapper-set")
	if !ok {
		t.Fatal("SetParam should create param")
	}
	if result.Data != "wrapper-data" {
		t.Errorf("SetParam created wrong data: data = %q, want %q", result.Data, "wrapper-data")
	}
}

func TestGetParamCtx_Found(t *testing.T) {
	eng := setupParamTestDB(t)
	ctx := context.Background()

	// Insert test param
	param := &model.TParam{
		Name:  "get-param",
		Data:  "get-data",
	}
	if _, err := eng.Insert(param); err != nil {
		t.Fatalf("failed to insert param: %v", err)
	}

	// Test lookup
	data, err := GetParamCtx(ctx, "get-param")
	if err != nil {
		t.Fatalf("GetParamCtx error: %v", err)
	}
	if string(data) != "get-data" {
		t.Errorf("GetParamCtx returned wrong data: %q, want %q", string(data), "get-data")
	}
}

func TestGetParamCtx_NotFound(t *testing.T) {
	setupParamTestDB(t)
	ctx := context.Background()

	_, err := GetParamCtx(ctx, "nonexistent")
	if err == nil {
		t.Error("GetParamCtx should return error for nonexistent param")
	}
	if err != ErrParamNotFound {
		t.Errorf("GetParamCtx returned wrong error: %v, want %v", err, ErrParamNotFound)
	}
}

func TestGetParam_Wrapper(t *testing.T) {
	eng := setupParamTestDB(t)

	// Insert test param
	param := &model.TParam{
		Name:  "wrapper-get",
		Data:  "wrapper-get-data",
	}
	if _, err := eng.Insert(param); err != nil {
		t.Fatalf("failed to insert param: %v", err)
	}

	// Test wrapper
	data, err := GetParam("wrapper-get")
	if err != nil {
		t.Fatalf("GetParam error: %v", err)
	}
	if string(data) != "wrapper-get-data" {
		t.Errorf("GetParam returned wrong data: %q, want %q", string(data), "wrapper-get-data")
	}
}
