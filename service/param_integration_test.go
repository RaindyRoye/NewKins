package service

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/gokins/gokins/comm"
	"github.com/gokins/gokins/model"
	_ "github.com/mattn/go-sqlite3"
	"xorm.io/xorm"
)

// setupParamTestDB creates an in-memory SQLite database with the TParam table.
func setupParamTestDB(t *testing.T) {
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
		t.Fatalf("failed to sync TParam table: %v", err)
	}
}

func TestFindParamCtx_Found(t *testing.T) {
	setupParamTestDB(t)
	ctx := context.Background()

	// Insert a param
	p := &model.TParam{
		Name:  "test-key",
		Title: "Test Title",
		Data:  `{"foo":"bar"}`,
		Times: time.Now(),
	}
	if _, err := comm.Db.Insert(p); err != nil {
		t.Fatalf("insert param: %v", err)
	}

	// Find it
	result, ok := FindParamCtx(ctx, "test-key")
	if !ok {
		t.Fatal("FindParamCtx should return true for existing key")
	}
	if result.Name != "test-key" {
		t.Errorf("Name = %q, want %q", result.Name, "test-key")
	}
	if result.Title != "Test Title" {
		t.Errorf("Title = %q, want %q", result.Title, "Test Title")
	}
	if result.Data != `{"foo":"bar"}` {
		t.Errorf("Data = %q, want %q", result.Data, `{"foo":"bar"}`)
	}
}

func TestFindParamCtx_NotFound(t *testing.T) {
	setupParamTestDB(t)
	ctx := context.Background()

	result, ok := FindParamCtx(ctx, "nonexistent")
	if ok {
		t.Error("FindParamCtx should return false for nonexistent key")
	}
	// result is still a zero-value struct
	if result.Aid != 0 {
		t.Errorf("result.Aid = %d, want 0", result.Aid)
	}
}

func TestSetParamCtx_Insert(t *testing.T) {
	setupParamTestDB(t)
	ctx := context.Background()

	// Insert new param
	err := SetParamCtx(ctx, "new-key", []byte(`{"x":1}`), "New Title")
	if err != nil {
		t.Fatalf("SetParamCtx (insert): %v", err)
	}

	// Verify it was inserted
	result, ok := FindParamCtx(ctx, "new-key")
	if !ok {
		t.Fatal("param should exist after insert")
	}
	if result.Title != "New Title" {
		t.Errorf("Title = %q, want %q", result.Title, "New Title")
	}
	if result.Data != `{"x":1}` {
		t.Errorf("Data = %q, want %q", result.Data, `{"x":1}`)
	}
}

func TestSetParamCtx_Update(t *testing.T) {
	setupParamTestDB(t)
	ctx := context.Background()

	// Insert first
	err := SetParamCtx(ctx, "update-key", []byte(`{"v":1}`), "Version 1")
	if err != nil {
		t.Fatalf("SetParamCtx (insert): %v", err)
	}

	// Update
	err = SetParamCtx(ctx, "update-key", []byte(`{"v":2}`), "Version 2")
	if err != nil {
		t.Fatalf("SetParamCtx (update): %v", err)
	}

	// Verify update
	result, ok := FindParamCtx(ctx, "update-key")
	if !ok {
		t.Fatal("param should exist after update")
	}
	if result.Title != "Version 2" {
		t.Errorf("Title = %q, want %q", result.Title, "Version 2")
	}
	if result.Data != `{"v":2}` {
		t.Errorf("Data = %q, want %q", result.Data, `{"v":2}`)
	}
}

func TestSetsParamCtx_JSON(t *testing.T) {
	setupParamTestDB(t)
	ctx := context.Background()

	data := map[string]int{"count": 42}
	err := SetsParamCtx(ctx, "json-key", data, "JSON Test")
	if err != nil {
		t.Fatalf("SetsParamCtx: %v", err)
	}

	// Verify it was stored as JSON
	result, ok := FindParamCtx(ctx, "json-key")
	if !ok {
		t.Fatal("param should exist")
	}

	var parsed map[string]int
	if err := json.Unmarshal([]byte(result.Data), &parsed); err != nil {
		t.Fatalf("unmarshal stored data: %v", err)
	}
	if parsed["count"] != 42 {
		t.Errorf("parsed count = %d, want 42", parsed["count"])
	}
}

func TestGetParamCtx_Found(t *testing.T) {
	setupParamTestDB(t)
	ctx := context.Background()

	// Insert a param
	p := &model.TParam{
		Name:  "get-key",
		Data:  "raw-data-content",
		Times: time.Now(),
	}
	if _, err := comm.Db.Insert(p); err != nil {
		t.Fatalf("insert param: %v", err)
	}

	bts, err := GetParamCtx(ctx, "get-key")
	if err != nil {
		t.Fatalf("GetParamCtx: %v", err)
	}
	if string(bts) != "raw-data-content" {
		t.Errorf("data = %q, want %q", string(bts), "raw-data-content")
	}
}

func TestGetParamCtx_NotFound(t *testing.T) {
	setupParamTestDB(t)
	ctx := context.Background()

	_, err := GetParamCtx(ctx, "missing-key")
	if err == nil {
		t.Fatal("GetParamCtx should return error for missing key")
	}
	if !errors.Is(err, ErrParamNotFound) {
		t.Errorf("error = %v, want ErrParamNotFound", err)
	}
}

func TestGetsParamCtx_Deserialize(t *testing.T) {
	setupParamTestDB(t)
	ctx := context.Background()

	type TestData struct {
		Name  string `json:"name"`
		Value int    `json:"value"`
	}

	// Insert JSON data
	original := TestData{Name: "test", Value: 99}
	err := SetsParamCtx(ctx, "deserialize-key", original, "Deserialize Test")
	if err != nil {
		t.Fatalf("SetsParamCtx: %v", err)
	}

	// Deserialize
	var result TestData
	err = GetsParamCtx(ctx, "deserialize-key", &result)
	if err != nil {
		t.Fatalf("GetsParamCtx: %v", err)
	}
	if result.Name != "test" {
		t.Errorf("Name = %q, want %q", result.Name, "test")
	}
	if result.Value != 99 {
		t.Errorf("Value = %d, want 99", result.Value)
	}
}

func TestGetsParamCtx_NotFound(t *testing.T) {
	setupParamTestDB(t)
	ctx := context.Background()

	var data map[string]string
	err := GetsParamCtx(ctx, "nonexistent", &data)
	if err == nil {
		t.Fatal("GetsParamCtx should return error for missing key")
	}
	if !errors.Is(err, ErrParamNotFound) {
		t.Errorf("error = %v, want ErrParamNotFound", err)
	}
}

func TestGetsParamCacheCtx_WithDB(t *testing.T) {
	setupParamTestDB(t)
	ctx := context.Background()

	// Cache is nil, so GetsParamCacheCtx should fall through to DB
	comm.BCache = nil

	type CacheData struct {
		Items []string `json:"items"`
	}

	// Insert data
	original := CacheData{Items: []string{"a", "b", "c"}}
	err := SetsParamCtx(ctx, "cache-key", original, "Cache Test")
	if err != nil {
		t.Fatalf("SetsParamCtx: %v", err)
	}

	// Get via cache path (will fall through to DB since cache is nil)
	var result CacheData
	err = GetsParamCacheCtx(ctx, "cache-key", &result)
	// With nil BCache, CacheGets returns ErrCacheNotInit, then GetsParamCtx succeeds
	if err != nil {
		t.Fatalf("GetsParamCacheCtx: %v", err)
	}
	if len(result.Items) != 3 {
		t.Errorf("Items count = %d, want 3", len(result.Items))
	}
	if result.Items[0] != "a" {
		t.Errorf("Items[0] = %q, want %q", result.Items[0], "a")
	}
}

func TestSetParamCtx_WithoutTitle(t *testing.T) {
	setupParamTestDB(t)
	ctx := context.Background()

	// Insert without title
	err := SetParamCtx(ctx, "no-title-key", []byte("some-data"))
	if err != nil {
		t.Fatalf("SetParamCtx without title: %v", err)
	}

	result, ok := FindParamCtx(ctx, "no-title-key")
	if !ok {
		t.Fatal("param should exist")
	}
	if result.Title != "" {
		t.Errorf("Title = %q, want empty", result.Title)
	}
}
