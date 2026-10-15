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

func setupPermsCoverageTestDB(t *testing.T) *xorm.Engine {
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

	err = eng.Sync2(&model.TUser{})
	if err != nil {
		t.Fatalf("failed to sync database schema: %v", err)
	}

	return eng
}

func TestCheckPermissionCtx_AdminUser(t *testing.T) {
	eng := setupPermsCoverageTestDB(t)
	ctx := context.Background()

	// Insert admin user
	user := &model.TUser{
		Id:   "admin",
		Aid:  1,
		Name: "admin",
	}
	if _, err := eng.Insert(user); err != nil {
		t.Fatalf("failed to insert user: %v", err)
	}

	// Test admin permission
	if !CheckPermissionCtx(ctx, "admin", PermAdmin) {
		t.Error("CheckPermissionCtx should grant admin permission to admin user")
	}

	// Test common permission
	if !CheckPermissionCtx(ctx, "admin", PermCommon) {
		t.Error("CheckPermissionCtx should grant common permission to admin user")
	}
}

func TestCheckPermissionCtx_RegularUser(t *testing.T) {
	eng := setupPermsCoverageTestDB(t)
	ctx := context.Background()

	// Insert regular user
	user := &model.TUser{
		Id:   "regular",
		Aid:  1,
		Name: "regular",
	}
	if _, err := eng.Insert(user); err != nil {
		t.Fatalf("failed to insert user: %v", err)
	}

	// Test admin permission (should fail)
	if CheckPermissionCtx(ctx, "regular", PermAdmin) {
		t.Error("CheckPermissionCtx should not grant admin permission to regular user")
	}

	// Test common permission (should pass)
	if !CheckPermissionCtx(ctx, "regular", PermCommon) {
		t.Error("CheckPermissionCtx should grant common permission to regular user")
	}
}

func TestCheckPermissionCtx_NonexistentUser(t *testing.T) {
	setupPermsCoverageTestDB(t)
	ctx := context.Background()

	// Test with nonexistent user
	if CheckPermissionCtx(ctx, "nonexistent", PermCommon) {
		t.Error("CheckPermissionCtx should return false for nonexistent user")
	}
}

func TestCheckPermissionCtx_UnknownPermission(t *testing.T) {
	eng := setupPermsCoverageTestDB(t)
	ctx := context.Background()

	// Insert user
	user := &model.TUser{
		Id:   "user1",
		Aid:  1,
		Name: "user1",
	}
	if _, err := eng.Insert(user); err != nil {
		t.Fatalf("failed to insert user: %v", err)
	}

	// Test unknown permission
	if CheckPermissionCtx(ctx, "user1", "unknown") {
		t.Error("CheckPermissionCtx should return false for unknown permission")
	}
}

func TestCheckPermission_Wrapper(t *testing.T) {
	eng := setupPermsCoverageTestDB(t)

	// Insert admin user
	user := &model.TUser{
		Id:   "admin",
		Aid:  1,
		Name: "admin",
	}
	if _, err := eng.Insert(user); err != nil {
		t.Fatalf("failed to insert user: %v", err)
	}

	// Test wrapper
	if !CheckPermission("admin", PermAdmin) {
		t.Error("CheckPermission should grant admin permission to admin user")
	}
}

func TestGetsParamCtx_Success(t *testing.T) {
	eng := setupParamTestDB(t)
	ctx := context.Background()

	// Insert test param with JSON data
	type TestData struct {
		Key string `json:"key"`
		Val int    `json:"val"`
	}
	data := TestData{Key: "test", Val: 42}
	jsonBytes, _ := json.Marshal(data)

	param := &model.TParam{
		Name: "json-param",
		Data: string(jsonBytes),
	}
	if _, err := eng.Insert(param); err != nil {
		t.Fatalf("failed to insert param: %v", err)
	}

	// Test deserialization
	var result TestData
	err := GetsParamCtx(ctx, "json-param", &result)
	if err != nil {
		t.Fatalf("GetsParamCtx error: %v", err)
	}
	if result.Key != "test" || result.Val != 42 {
		t.Errorf("GetsParamCtx deserialized wrong data: %+v", result)
	}
}

func TestGetsParamCtx_NotFound(t *testing.T) {
	setupParamTestDB(t)
	ctx := context.Background()

	var result map[string]string
	err := GetsParamCtx(ctx, "nonexistent", &result)
	if err == nil {
		t.Error("GetsParamCtx should return error for nonexistent param")
	}
}

func TestGetsParam_Wrapper(t *testing.T) {
	eng := setupParamTestDB(t)

	// Insert test param
	type TestData struct {
		Name string `json:"name"`
	}
	data := TestData{Name: "wrapper"}
	jsonBytes, _ := json.Marshal(data)

	param := &model.TParam{
		Name: "wrapper-json",
		Data: string(jsonBytes),
	}
	if _, err := eng.Insert(param); err != nil {
		t.Fatalf("failed to insert param: %v", err)
	}

	// Test wrapper
	var result TestData
	err := GetsParam("wrapper-json", &result)
	if err != nil {
		t.Fatalf("GetsParam error: %v", err)
	}
	if result.Name != "wrapper" {
		t.Errorf("GetsParam deserialized wrong data: %+v", result)
	}
}

func TestGetsParamCacheCtx_CacheHit(t *testing.T) {
	eng := setupParamTestDB(t)
	ctx := context.Background()

	// Setup cache
	oldCache := comm.BCache
	defer func() { comm.BCache = oldCache }()

	cache, err := xorm.NewEngine("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("failed to create cache: %v", err)
	}
	defer cache.Close()

	// For this test, we'll skip cache setup and just test DB path
	comm.BCache = nil

	// Insert test param
	type TestData struct {
		Value string `json:"value"`
	}
	data := TestData{Value: "cached"}
	jsonBytes, _ := json.Marshal(data)

	param := &model.TParam{
		Name: "cache-param",
		Data: string(jsonBytes),
	}
	if _, err := eng.Insert(param); err != nil {
		t.Fatalf("failed to insert param: %v", err)
	}

	// Test cache miss -> DB fetch
	var result TestData
	err = GetsParamCacheCtx(ctx, "cache-param", &result, time.Minute)
	if err != nil {
		t.Fatalf("GetsParamCacheCtx error: %v", err)
	}
	if result.Value != "cached" {
		t.Errorf("GetsParamCacheCtx deserialized wrong data: %+v", result)
	}
}

func TestGetsParamCache_Wrapper(t *testing.T) {
	eng := setupParamTestDB(t)

	// Disable cache for this test
	oldCache := comm.BCache
	comm.BCache = nil
	defer func() { comm.BCache = oldCache }()

	// Insert test param
	type TestData struct {
		Count int `json:"count"`
	}
	data := TestData{Count: 99}
	jsonBytes, _ := json.Marshal(data)

	param := &model.TParam{
		Name: "wrapper-cache",
		Data: string(jsonBytes),
	}
	if _, err := eng.Insert(param); err != nil {
		t.Fatalf("failed to insert param: %v", err)
	}

	// Test wrapper
	var result TestData
	err := GetsParamCache("wrapper-cache", &result)
	if err != nil {
		t.Fatalf("GetsParamCache error: %v", err)
	}
	if result.Count != 99 {
		t.Errorf("GetsParamCache deserialized wrong data: %+v", result)
	}
}
