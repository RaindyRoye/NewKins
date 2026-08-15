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

// setupParamTestDB creates an isolated in-memory SQLite DB for param tests.
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
	if err := eng.Sync2(&model.TParam{}); err != nil {
		t.Fatalf("failed to sync schema: %v", err)
	}
	return eng
}

// --- FindParamCtx ---

func TestFindParamCtx_Found(t *testing.T) {
	eng := setupParamTestDB(t)
	ctx := context.Background()

	// Insert a param directly
	param := &model.TParam{
		Aid:   1,
		Name:  "test-param",
		Title: "Test Param",
		Data:  `{"key":"value"}`,
		Times: time.Now(),
	}
	if _, err := eng.Insert(param); err != nil {
		t.Fatalf("insert param: %v", err)
	}

	got, ok := FindParamCtx(ctx, "test-param")
	if !ok {
		t.Fatal("FindParamCtx should find the param")
	}
	if got.Name != "test-param" {
		t.Errorf("Name = %q, want %q", got.Name, "test-param")
	}
	if got.Title != "Test Param" {
		t.Errorf("Title = %q, want %q", got.Title, "Test Param")
	}
	if got.Data != `{"key":"value"}` {
		t.Errorf("Data = %q, want %q", got.Data, `{"key":"value"}`)
	}
}

func TestFindParamCtx_NotFound(t *testing.T) {
	setupParamTestDB(t)
	ctx := context.Background()

	got, ok := FindParamCtx(ctx, "nonexistent")
	if ok {
		t.Fatal("FindParamCtx should not find nonexistent param")
	}
	// Even when not found, a zero-value struct is returned
	if got == nil {
		t.Fatal("FindParamCtx should return non-nil struct even when not found")
	}
	if got.Name != "" {
		t.Errorf("Name should be empty for not-found param, got %q", got.Name)
	}
}

// --- SetParamCtx ---

func TestSetParamCtx_CreateNew(t *testing.T) {
	eng := setupParamTestDB(t)
	ctx := context.Background()

	err := SetParamCtx(ctx, "new-param", []byte("hello world"), "My Title")
	if err != nil {
		t.Fatalf("SetParamCtx create: %v", err)
	}

	// Verify the record was inserted
	p := &model.TParam{}
	ok, err := eng.Where("name=?", "new-param").Get(p)
	if err != nil {
		t.Fatalf("query: %v", err)
	}
	if !ok {
		t.Fatal("param should exist after creation")
	}
	if p.Data != "hello world" {
		t.Errorf("Data = %q, want %q", p.Data, "hello world")
	}
	if p.Title != "My Title" {
		t.Errorf("Title = %q, want %q", p.Title, "My Title")
	}
}

func TestSetParamCtx_UpdateExisting(t *testing.T) {
	eng := setupParamTestDB(t)
	ctx := context.Background()

	// Create initial param
	err := SetParamCtx(ctx, "update-me", []byte("initial"), "Initial Title")
	if err != nil {
		t.Fatalf("SetParamCtx create: %v", err)
	}

	// Update it
	err = SetParamCtx(ctx, "update-me", []byte("updated"), "Updated Title")
	if err != nil {
		t.Fatalf("SetParamCtx update: %v", err)
	}

	// Verify only one record exists and it has updated values
	p := &model.TParam{}
	ok, err := eng.Where("name=?", "update-me").Get(p)
	if err != nil {
		t.Fatalf("query: %v", err)
	}
	if !ok {
		t.Fatal("param should exist after update")
	}
	if p.Data != "updated" {
		t.Errorf("Data = %q, want %q", p.Data, "updated")
	}
	if p.Title != "Updated Title" {
		t.Errorf("Title = %q, want %q", p.Title, "Updated Title")
	}

	// Ensure only one record
	count, err := eng.Where("name=?", "update-me").Count(&model.TParam{})
	if err != nil {
		t.Fatalf("count: %v", err)
	}
	if count != 1 {
		t.Errorf("count = %d, want 1 (should update, not insert duplicate)", count)
	}
}

func TestSetParamCtx_NoTitle(t *testing.T) {
	eng := setupParamTestDB(t)
	ctx := context.Background()

	err := SetParamCtx(ctx, "no-title", []byte("data"))
	if err != nil {
		t.Fatalf("SetParamCtx no title: %v", err)
	}

	p := &model.TParam{}
	ok, err := eng.Where("name=?", "no-title").Get(p)
	if err != nil {
		t.Fatalf("query: %v", err)
	}
	if !ok {
		t.Fatal("param should exist")
	}
	if p.Title != "" {
		t.Errorf("Title = %q, want empty string", p.Title)
	}
}

// --- SetsParamCtx (JSON serialization) ---

func TestSetsParamCtx_RoundTrip(t *testing.T) {
	setupParamTestDB(t)
	ctx := context.Background()

	type testData struct {
		Name  string `json:"name"`
		Value int    `json:"value"`
	}
	input := testData{Name: "test", Value: 42}

	err := SetsParamCtx(ctx, "json-param", input, "JSON Test")
	if err != nil {
		t.Fatalf("SetsParamCtx: %v", err)
	}

	// Verify raw data is valid JSON
	got, ok := FindParamCtx(ctx, "json-param")
	if !ok {
		t.Fatal("param should exist")
	}

	var decoded testData
	if err := json.Unmarshal([]byte(got.Data), &decoded); err != nil {
		t.Fatalf("decode JSON: %v", err)
	}
	if decoded.Name != "test" || decoded.Value != 42 {
		t.Errorf("decoded = %+v, want {test 42}", decoded)
	}
}

func TestSetsParamCtx_Map(t *testing.T) {
	setupParamTestDB(t)
	ctx := context.Background()

	input := map[string]any{
		"key1": "value1",
		"key2": 123,
	}
	err := SetsParamCtx(ctx, "map-param", input)
	if err != nil {
		t.Fatalf("SetsParamCtx map: %v", err)
	}

	got, ok := FindParamCtx(ctx, "map-param")
	if !ok {
		t.Fatal("param should exist")
	}
	var decoded map[string]any
	if err := json.Unmarshal([]byte(got.Data), &decoded); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if decoded["key1"] != "value1" {
		t.Errorf("key1 = %v, want value1", decoded["key1"])
	}
}

// --- GetParamCtx ---

func TestGetParamCtx_Found(t *testing.T) {
	setupParamTestDB(t)
	ctx := context.Background()

	if err := SetParamCtx(ctx, "raw-param", []byte("raw-bytes")); err != nil {
		t.Fatalf("SetParamCtx: %v", err)
	}

	data, err := GetParamCtx(ctx, "raw-param")
	if err != nil {
		t.Fatalf("GetParamCtx: %v", err)
	}
	if string(data) != "raw-bytes" {
		t.Errorf("data = %q, want %q", string(data), "raw-bytes")
	}
}

func TestGetParamCtx_NotFound(t *testing.T) {
	setupParamTestDB(t)
	ctx := context.Background()

	_, err := GetParamCtx(ctx, "missing")
	if err == nil {
		t.Fatal("GetParamCtx(missing) should return error")
	}
	if !errors.Is(err, ErrParamNotFound) {
		t.Errorf("GetParamCtx(missing) = %v, want ErrParamNotFound", err)
	}
}

// --- GetsParamCtx (JSON deserialization) ---

func TestGetsParamCtx_RoundTrip(t *testing.T) {
	setupParamTestDB(t)
	ctx := context.Background()

	type config struct {
		Host string `json:"host"`
		Port int    `json:"port"`
	}
	original := config{Host: "localhost", Port: 8080}
	if err := SetsParamCtx(ctx, "config", original); err != nil {
		t.Fatalf("SetsParamCtx: %v", err)
	}

	var loaded config
	if err := GetsParamCtx(ctx, "config", &loaded); err != nil {
		t.Fatalf("GetsParamCtx: %v", err)
	}
	if loaded.Host != "localhost" || loaded.Port != 8080 {
		t.Errorf("loaded = %+v, want %+v", loaded, original)
	}
}

func TestGetsParamCtx_NotFound(t *testing.T) {
	setupParamTestDB(t)
	ctx := context.Background()

	var data map[string]string
	err := GetsParamCtx(ctx, "nonexistent", &data)
	if err == nil {
		t.Fatal("GetsParamCtx(nonexistent) should return error")
	}
}

func TestGetsParamCtx_InvalidJSON(t *testing.T) {
	setupParamTestDB(t)
	ctx := context.Background()

	// Store invalid JSON
	if err := SetParamCtx(ctx, "bad-json", []byte("not-json{{")); err != nil {
		t.Fatalf("SetParamCtx: %v", err)
	}

	var data map[string]string
	err := GetsParamCtx(ctx, "bad-json", &data)
	if err == nil {
		t.Fatal("GetsParamCtx(invalid JSON) should return error")
	}
}

func TestGetsParamCtx_IntegNilData(t *testing.T) {
	setupParamTestDB(t)
	ctx := context.Background()

	err := GetsParamCtx(ctx, "key", nil)
	if err == nil {
		t.Fatal("GetsParamCtx(nil data) should return error")
	}
	if !errors.Is(err, ErrParamDataNil) {
		t.Errorf("GetsParamCtx(nil data) = %v, want ErrParamDataNil", err)
	}
}

// --- Multiple params CRUD ---

func TestParamCtx_MultipleParams(t *testing.T) {
	setupParamTestDB(t)
	ctx := context.Background()

	params := map[string]string{
		"param-a": "data-a",
		"param-b": "data-b",
		"param-c": "data-c",
	}
	for name, data := range params {
		if err := SetParamCtx(ctx, name, []byte(data)); err != nil {
			t.Fatalf("SetParamCtx(%s): %v", name, err)
		}
	}

	// Verify each one
	for name, want := range params {
		got, ok := FindParamCtx(ctx, name)
		if !ok {
			t.Errorf("FindParamCtx(%s) not found", name)
			continue
		}
		if got.Data != want {
			t.Errorf("FindParamCtx(%s).Data = %q, want %q", name, got.Data, want)
		}
	}

	// GetParam for each
	for name, want := range params {
		data, err := GetParamCtx(ctx, name)
		if err != nil {
			t.Errorf("GetParamCtx(%s): %v", name, err)
			continue
		}
		if string(data) != want {
			t.Errorf("GetParamCtx(%s) = %q, want %q", name, string(data), want)
		}
	}
}

// --- Global-context wrappers ---

func TestFindParam_GlobalContext(t *testing.T) {
	setupParamTestDB(t)

	// comm.Ctx is initialized in init(), so it's available
	if err := SetParamCtx(comm.Ctx, "global-param", []byte("global-data")); err != nil {
		t.Fatalf("SetParamCtx: %v", err)
	}

	got, ok := FindParam("global-param")
	if !ok {
		t.Fatal("FindParam should find the param")
	}
	if got.Data != "global-data" {
		t.Errorf("Data = %q, want %q", got.Data, "global-data")
	}
}

func TestSetParam_GlobalContext(t *testing.T) {
	setupParamTestDB(t)

	if err := SetParam("global-set", []byte("value"), "title"); err != nil {
		t.Fatalf("SetParam: %v", err)
	}

	data, err := GetParam("global-set")
	if err != nil {
		t.Fatalf("GetParam: %v", err)
	}
	if string(data) != "value" {
		t.Errorf("data = %q, want %q", string(data), "value")
	}
}

func TestSetsParam_GlobalContext(t *testing.T) {
	setupParamTestDB(t)

	input := map[string]int{"count": 5}
	if err := SetsParam("global-json", input); err != nil {
		t.Fatalf("SetsParam: %v", err)
	}

	var loaded map[string]int
	if err := GetsParam("global-json", &loaded); err != nil {
		t.Fatalf("GetsParam: %v", err)
	}
	if loaded["count"] != 5 {
		t.Errorf("count = %d, want 5", loaded["count"])
	}
}

func TestGetParam_GlobalContext_NotFound(t *testing.T) {
	setupParamTestDB(t)

	_, err := GetParam("does-not-exist")
	if !errors.Is(err, ErrParamNotFound) {
		t.Errorf("GetParam = %v, want ErrParamNotFound", err)
	}
}

// --- Canceled context ---

func TestSetParamCtx_CancelledContext(t *testing.T) {
	setupParamTestDB(t)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	// SQLite may still succeed with a canceled context since it's in-memory,
	// but we at least verify no panic occurs
	_ = SetParamCtx(ctx, "canceled", []byte("data"))
}
