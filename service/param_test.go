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

// setupParamTestDB creates an in-memory SQLite database with the TParam table.
func setupParamTestDB(t *testing.T) *xorm.Engine {
	t.Helper()
	db, err := xorm.NewEngine("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("failed to create test database: %v", err)
	}

	err = db.Sync2(new(model.TParam))
	if err != nil {
		t.Fatalf("failed to sync schema: %v", err)
	}

	oldDb := comm.Db
	oldCtx := comm.Ctx
	comm.Db = db
	comm.Ctx = context.Background()

	t.Cleanup(func() {
		comm.Db = oldDb
		comm.Ctx = oldCtx
		_ = db.Close()
	})

	return db
}

func TestFindParam_Found(t *testing.T) {
	db := setupParamTestDB(t)

	param := &model.TParam{
		Name:  "test_key",
		Title: "Test Title",
		Data:  `{"hello":"world"}`,
		Times: time.Now(),
	}
	_, err := db.Insert(param)
	if err != nil {
		t.Fatalf("insert failed: %v", err)
	}

	found, ok := FindParam("test_key")
	if !ok {
		t.Fatal("expected to find existing param")
	}
	if found.Data != `{"hello":"world"}` {
		t.Errorf("data mismatch: got %q", found.Data)
	}
	if found.Title != "Test Title" {
		t.Errorf("title mismatch: got %q", found.Title)
	}
}

func TestFindParam_NotFound(t *testing.T) {
	setupParamTestDB(t)

	_, ok := FindParam("nonexistent_key")
	if ok {
		t.Error("should not find nonexistent param")
	}
}

func TestFindParamCtx_Found(t *testing.T) {
	db := setupParamTestDB(t)
	ctx := context.Background()

	param := &model.TParam{
		Name:  "ctx_key",
		Title: "Ctx Title",
		Data:  "ctx_data",
		Times: time.Now(),
	}
	_, err := db.Insert(param)
	if err != nil {
		t.Fatalf("insert failed: %v", err)
	}

	found, ok := FindParamCtx(ctx, "ctx_key")
	if !ok {
		t.Fatal("expected to find existing param")
	}
	if found.Data != "ctx_data" {
		t.Errorf("data mismatch: got %q", found.Data)
	}
}

func TestFindParamCtx_CancelledContext(t *testing.T) {
	setupParamTestDB(t)

	cancelCtx, cancel := context.WithCancel(context.Background())
	cancel()

	// Should not panic even with cancelled context
	_, _ = FindParamCtx(cancelCtx, "any_key")
}

func TestGetParam_NotFound(t *testing.T) {
	setupParamTestDB(t)

	_, err := GetParam("missing_key")
	if err == nil {
		t.Fatal("expected error for missing param")
	}
	if !errors.Is(err, ErrParamNotFound) {
		t.Errorf("expected ErrParamNotFound, got %v", err)
	}
}

func TestGetParam_Found(t *testing.T) {
	db := setupParamTestDB(t)

	param := &model.TParam{
		Name:  "get_key",
		Data:  "get_value",
		Times: time.Now(),
	}
	_, err := db.Insert(param)
	if err != nil {
		t.Fatalf("insert failed: %v", err)
	}

	value, err := GetParam("get_key")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(value) != "get_value" {
		t.Errorf("value mismatch: got %q, want %q", string(value), "get_value")
	}
}

func TestGetParamCtx_NotFound(t *testing.T) {
	setupParamTestDB(t)
	ctx := context.Background()

	_, err := GetParamCtx(ctx, "missing_key")
	if err == nil {
		t.Fatal("expected error")
	}
	if !errors.Is(err, ErrParamNotFound) {
		t.Errorf("expected ErrParamNotFound, got %v", err)
	}
}

func TestGetParamCtx_Found(t *testing.T) {
	db := setupParamTestDB(t)
	ctx := context.Background()

	param := &model.TParam{
		Name:  "ctx_get_key",
		Data:  "ctx_get_value",
		Times: time.Now(),
	}
	_, err := db.Insert(param)
	if err != nil {
		t.Fatalf("insert failed: %v", err)
	}

	value, err := GetParamCtx(ctx, "ctx_get_key")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(value) != "ctx_get_value" {
		t.Errorf("value mismatch: got %q", string(value))
	}
}

func TestSetParam_NewKey(t *testing.T) {
	setupParamTestDB(t)

	err := SetParam("new_key", []byte("new_value"))
	if err != nil {
		t.Fatalf("SetParam failed: %v", err)
	}

	value, err := GetParam("new_key")
	if err != nil {
		t.Fatalf("GetParam failed: %v", err)
	}
	if string(value) != "new_value" {
		t.Errorf("value mismatch: got %q, want %q", string(value), "new_value")
	}
}

func TestSetParam_UpdateExisting(t *testing.T) {
	setupParamTestDB(t)

	err := SetParam("update_key", []byte("original"))
	if err != nil {
		t.Fatalf("initial SetParam failed: %v", err)
	}

	err = SetParam("update_key", []byte("updated"))
	if err != nil {
		t.Fatalf("update SetParam failed: %v", err)
	}

	value, err := GetParam("update_key")
	if err != nil {
		t.Fatalf("GetParam failed: %v", err)
	}
	if string(value) != "updated" {
		t.Errorf("value mismatch: got %q, want %q", string(value), "updated")
	}
}

func TestSetParam_WithTitle(t *testing.T) {
	setupParamTestDB(t)

	err := SetParam("titled_key", []byte("titled_data"), "My Title")
	if err != nil {
		t.Fatalf("SetParam with title failed: %v", err)
	}

	found, ok := FindParam("titled_key")
	if !ok {
		t.Fatal("expected to find param")
	}
	if found.Title != "My Title" {
		t.Errorf("title mismatch: got %q, want %q", found.Title, "My Title")
	}
}

func TestSetParamCtx_NewKey(t *testing.T) {
	setupParamTestDB(t)
	ctx := context.Background()

	err := SetParamCtx(ctx, "ctx_new_key", []byte("ctx_new_value"))
	if err != nil {
		t.Fatalf("SetParamCtx failed: %v", err)
	}

	value, err := GetParamCtx(ctx, "ctx_new_key")
	if err != nil {
		t.Fatalf("GetParamCtx failed: %v", err)
	}
	if string(value) != "ctx_new_value" {
		t.Errorf("value mismatch: got %q", string(value))
	}
}

func TestSetParamCtx_UpdateExisting(t *testing.T) {
	setupParamTestDB(t)
	ctx := context.Background()

	err := SetParamCtx(ctx, "ctx_update_key", []byte("original"))
	if err != nil {
		t.Fatalf("initial SetParamCtx failed: %v", err)
	}

	err = SetParamCtx(ctx, "ctx_update_key", []byte("updated"))
	if err != nil {
		t.Fatalf("update SetParamCtx failed: %v", err)
	}

	value, err := GetParamCtx(ctx, "ctx_update_key")
	if err != nil {
		t.Fatalf("GetParamCtx failed: %v", err)
	}
	if string(value) != "updated" {
		t.Errorf("value mismatch")
	}
}

func TestSetParamCtx_WithTitle(t *testing.T) {
	setupParamTestDB(t)
	ctx := context.Background()

	err := SetParamCtx(ctx, "ctx_titled_key", []byte("data"), "Ctx Title")
	if err != nil {
		t.Fatalf("SetParamCtx with title failed: %v", err)
	}

	found, ok := FindParamCtx(ctx, "ctx_titled_key")
	if !ok {
		t.Fatal("expected to find param")
	}
	if found.Title != "Ctx Title" {
		t.Errorf("title mismatch: got %q", found.Title)
	}
}

func TestSetsParam_NilData(t *testing.T) {
	setupParamTestDB(t)

	err := SetsParam("test-key", nil)
	if err == nil {
		t.Fatal("SetsParam(nil data) should return error")
	}
	if !errors.Is(err, ErrParamDataNil) {
		t.Errorf("expected ErrParamDataNil, got %v", err)
	}
}

func TestSetsParam_Struct(t *testing.T) {
	setupParamTestDB(t)

	type Config struct {
		Host string `json:"host"`
		Port int    `json:"port"`
	}

	err := SetsParam("config_key", Config{Host: "localhost", Port: 8080})
	if err != nil {
		t.Fatalf("SetsParam failed: %v", err)
	}

	var cfg Config
	err = GetsParam("config_key", &cfg)
	if err != nil {
		t.Fatalf("GetsParam failed: %v", err)
	}
	if cfg.Host != "localhost" || cfg.Port != 8080 {
		t.Errorf("struct mismatch: got %+v", cfg)
	}
}

func TestSetsParam_Map(t *testing.T) {
	setupParamTestDB(t)

	err := SetsParam("map_key", map[string]int{"a": 1, "b": 2})
	if err != nil {
		t.Fatalf("SetsParam map failed: %v", err)
	}

	var result map[string]int
	err = GetsParam("map_key", &result)
	if err != nil {
		t.Fatalf("GetsParam map failed: %v", err)
	}
	if result["a"] != 1 || result["b"] != 2 {
		t.Errorf("map mismatch: got %+v", result)
	}
}

func TestSetsParam_Slice(t *testing.T) {
	setupParamTestDB(t)

	err := SetsParam("array_key", []string{"x", "y", "z"})
	if err != nil {
		t.Fatalf("SetsParam slice failed: %v", err)
	}

	var result []string
	err = GetsParam("array_key", &result)
	if err != nil {
		t.Fatalf("GetsParam slice failed: %v", err)
	}
	if len(result) != 3 || result[0] != "x" || result[1] != "y" || result[2] != "z" {
		t.Errorf("slice mismatch: got %+v", result)
	}
}

func TestSetsParamCtx_NilData(t *testing.T) {
	setupParamTestDB(t)
	ctx := context.Background()

	err := SetsParamCtx(ctx, "test-key", nil)
	if err == nil {
		t.Fatal("SetsParamCtx(nil data) should return error")
	}
	if !errors.Is(err, ErrParamDataNil) {
		t.Errorf("expected ErrParamDataNil, got %v", err)
	}
}

func TestSetsParamCtx_Struct(t *testing.T) {
	setupParamTestDB(t)
	ctx := context.Background()

	type TestData struct {
		ID   int    `json:"id"`
		Text string `json:"text"`
	}

	err := SetsParamCtx(ctx, "ctx_json_key", TestData{ID: 1, Text: "hello"})
	if err != nil {
		t.Fatalf("SetsParamCtx failed: %v", err)
	}

	var result TestData
	err = GetsParamCtx(ctx, "ctx_json_key", &result)
	if err != nil {
		t.Fatalf("GetsParamCtx failed: %v", err)
	}
	if result.ID != 1 || result.Text != "hello" {
		t.Errorf("struct mismatch: got %+v", result)
	}
}

func TestGetsParam_NilData(t *testing.T) {
	setupParamTestDB(t)

	err := GetsParam("test-key", nil)
	if err == nil {
		t.Fatal("GetsParam(nil data) should return error")
	}
	if !errors.Is(err, ErrParamDataNil) {
		t.Errorf("expected ErrParamDataNil, got %v", err)
	}
}

func TestGetsParam_NotFound(t *testing.T) {
	setupParamTestDB(t)

	var result map[string]string
	err := GetsParam("missing_key", &result)
	if err == nil {
		t.Fatal("expected error for missing param")
	}
}

func TestGetsParamCtx_NilData(t *testing.T) {
	setupParamTestDB(t)
	ctx := context.Background()

	err := GetsParamCtx(ctx, "test-key", nil)
	if err == nil {
		t.Fatal("GetsParamCtx(nil data) should return error")
	}
	if !errors.Is(err, ErrParamDataNil) {
		t.Errorf("expected ErrParamDataNil, got %v", err)
	}
}

func TestSetsParam_Nested(t *testing.T) {
	setupParamTestDB(t)

	type Nested struct {
		Inner struct {
			Value string `json:"value"`
		} `json:"inner"`
	}

	var data Nested
	data.Inner.Value = "nested_test"

	err := SetsParam("nested_key", data)
	if err != nil {
		t.Fatalf("SetsParam nested failed: %v", err)
	}

	var result Nested
	err = GetsParam("nested_key", &result)
	if err != nil {
		t.Fatalf("GetsParam nested failed: %v", err)
	}
	if result.Inner.Value != "nested_test" {
		t.Errorf("nested value mismatch")
	}
}

func TestSetsParam_Update(t *testing.T) {
	setupParamTestDB(t)

	type Data struct {
		V int `json:"v"`
	}

	err := SetsParam("update_json", Data{V: 1})
	if err != nil {
		t.Fatalf("initial SetsParam failed: %v", err)
	}

	err = SetsParam("update_json", Data{V: 2})
	if err != nil {
		t.Fatalf("update SetsParam failed: %v", err)
	}

	var result Data
	err = GetsParam("update_json", &result)
	if err != nil {
		t.Fatalf("GetsParam failed: %v", err)
	}
	if result.V != 2 {
		t.Errorf("value mismatch: got %d, want 2", result.V)
	}
}

func TestGetsParamCacheCtx_NoCache(t *testing.T) {
	setupParamTestDB(t)
	ctx := context.Background()

	// BCache is nil, so CacheGets returns ErrCacheNotInit
	// Should fall through to DB query
	type CacheData struct {
		Value string `json:"value"`
	}

	err := SetsParamCtx(ctx, "cache_key", CacheData{Value: "cached"})
	if err != nil {
		t.Fatalf("SetsParamCtx failed: %v", err)
	}

	var result CacheData
	err = GetsParamCacheCtx(ctx, "cache_key", &result)
	if err != nil {
		t.Fatalf("GetsParamCacheCtx failed: %v", err)
	}
	if result.Value != "cached" {
		t.Errorf("cached value mismatch: got %q", result.Value)
	}
}

func TestGetsParamCacheCtx_NotFound(t *testing.T) {
	setupParamTestDB(t)
	ctx := context.Background()

	var result map[string]string
	err := GetsParamCacheCtx(ctx, "missing_cache_key", &result)
	if err == nil {
		t.Fatal("expected error for missing param")
	}
}

func TestGetsParamCache_NoCache(t *testing.T) {
	setupParamTestDB(t)

	type CacheData struct {
		Value string `json:"value"`
	}

	err := SetsParam("cache_no_cache_key", CacheData{Value: "from_db"})
	if err != nil {
		t.Fatalf("SetsParam failed: %v", err)
	}

	var result CacheData
	err = GetsParamCache("cache_no_cache_key", &result)
	if err != nil {
		t.Fatalf("GetsParamCache failed: %v", err)
	}
	if result.Value != "from_db" {
		t.Errorf("value mismatch: got %q", result.Value)
	}
}

func TestGetsParamCache_NotFound(t *testing.T) {
	setupParamTestDB(t)

	var result map[string]string
	err := GetsParamCache("missing_key", &result)
	if err == nil {
		t.Fatal("expected error for missing param")
	}
}

func TestSetParam_Roundtrip(t *testing.T) {
	setupParamTestDB(t)

	// Set → Get roundtrip with binary-like data
	data := []byte(`{"key":"value with \"quotes\" and unicode: 你好世界"}`)
	err := SetParam("roundtrip_key", data)
	if err != nil {
		t.Fatalf("SetParam failed: %v", err)
	}

	got, err := GetParam("roundtrip_key")
	if err != nil {
		t.Fatalf("GetParam failed: %v", err)
	}
	if string(got) != string(data) {
		t.Errorf("roundtrip mismatch: got %q", string(got))
	}
}

func TestSetParam_EmptyData(t *testing.T) {
	setupParamTestDB(t)

	err := SetParam("empty_key", []byte(""))
	if err != nil {
		t.Fatalf("SetParam with empty data failed: %v", err)
	}

	got, err := GetParam("empty_key")
	if err != nil {
		t.Fatalf("GetParam failed: %v", err)
	}
	if string(got) != "" {
		t.Errorf("expected empty string, got %q", string(got))
	}
}

func TestGetsParamCtx_CancelledContext(t *testing.T) {
	setupParamTestDB(t)

	cancelCtx, cancel := context.WithCancel(context.Background())
	cancel()

	var data map[string]string
	// Should not panic, just return error
	_ = GetsParamCtx(cancelCtx, "any_key", &data)
}

func TestGetParamCtx_CancelledContext(t *testing.T) {
	setupParamTestDB(t)

	cancelCtx, cancel := context.WithCancel(context.Background())
	cancel()

	// Should not panic, just return error
	_, _ = GetParamCtx(cancelCtx, "any_key")
}

func TestSetParam_WithTitleUpdate(t *testing.T) {
	setupParamTestDB(t)

	// Create with title
	err := SetParam("title_update", []byte("data1"), "Title1")
	if err != nil {
		t.Fatalf("SetParam failed: %v", err)
	}

	found, ok := FindParam("title_update")
	if !ok {
		t.Fatal("expected to find param")
	}
	if found.Title != "Title1" {
		t.Errorf("initial title mismatch: got %q", found.Title)
	}

	// Update with new title
	err = SetParam("title_update", []byte("data2"), "Title2")
	if err != nil {
		t.Fatalf("SetParam update failed: %v", err)
	}

	found, ok = FindParam("title_update")
	if !ok {
		t.Fatal("expected to find param after update")
	}
	if found.Title != "Title2" {
		t.Errorf("updated title mismatch: got %q", found.Title)
	}
	if found.Data != "data2" {
		t.Errorf("updated data mismatch: got %q", found.Data)
	}
}
