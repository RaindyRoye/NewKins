package bean

import (
	"encoding/json"
	"testing"
	"time"
)

// --- Page and PageGen tests ---

func TestPage_JSONRoundTrip(t *testing.T) {
	original := Page{
		Page:  2,
		Size:  10,
		Total: 42,
		Pages: 5,
		Data:  "some data",
	}
	bts, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("json.Marshal(Page) error: %v", err)
	}
	var decoded Page
	if err := json.Unmarshal(bts, &decoded); err != nil {
		t.Fatalf("json.Unmarshal(Page) error: %v", err)
	}
	if decoded.Page != original.Page || decoded.Size != original.Size ||
		decoded.Total != original.Total || decoded.Pages != original.Pages {
		t.Errorf("Page round-trip mismatch: got %+v, want %+v", decoded, original)
	}
}

func TestPage_ZeroValues(t *testing.T) {
	p := Page{}
	bts, err := json.Marshal(p)
	if err != nil {
		t.Fatalf("json.Marshal(Page{}) error: %v", err)
	}
	var decoded Page
	if err := json.Unmarshal(bts, &decoded); err != nil {
		t.Fatalf("json.Unmarshal error: %v", err)
	}
	if decoded.Page != 0 || decoded.Size != 0 || decoded.Total != 0 || decoded.Pages != 0 {
		t.Errorf("zero-value Page should decode to zero fields, got %+v", decoded)
	}
}

func TestPageGen_Fields(t *testing.T) {
	pg := PageGen{
		SQL:       "SELECT * FROM builds WHERE id = ?",
		Args:      []any{42},
		CountCols: "id",
		FindCols:  "id,name,status",
	}
	if pg.SQL == "" {
		t.Error("PageGen.SQL should not be empty")
	}
	if len(pg.Args) != 1 {
		t.Errorf("PageGen.Args length = %d, want 1", len(pg.Args))
	}
	if pg.CountCols != "id" {
		t.Errorf("PageGen.CountCols = %q, want %q", pg.CountCols, "id")
	}
}

// --- IdsRes tests ---

func TestIdsRes_JSONRoundTrip(t *testing.T) {
	original := IdsRes{Id: "abc-123", Aid: 456}
	bts, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("json.Marshal(IdsRes) error: %v", err)
	}
	var decoded IdsRes
	if err := json.Unmarshal(bts, &decoded); err != nil {
		t.Fatalf("json.Unmarshal(IdsRes) error: %v", err)
	}
	if decoded.Id != original.Id || decoded.Aid != original.Aid {
		t.Errorf("IdsRes round-trip mismatch: got %+v, want %+v", decoded, original)
	}
}

// --- LoginReq tests ---

func TestLoginReq_JSONRoundTrip(t *testing.T) {
	original := LoginReq{Name: "admin", Pass: "secret123"}
	bts, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("json.Marshal(LoginReq) error: %v", err)
	}
	var decoded LoginReq
	if err := json.Unmarshal(bts, &decoded); err != nil {
		t.Fatalf("json.Unmarshal(LoginReq) error: %v", err)
	}
	if decoded.Name != original.Name || decoded.Pass != original.Pass {
		t.Errorf("LoginReq round-trip mismatch: got %+v, want %+v", decoded, original)
	}
}

func TestLoginReq_EmptyFields(t *testing.T) {
	req := LoginReq{}
	bts, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("json.Marshal error: %v", err)
	}
	// Empty fields should produce valid JSON
	var decoded LoginReq
	if err := json.Unmarshal(bts, &decoded); err != nil {
		t.Fatalf("json.Unmarshal error: %v", err)
	}
	if decoded.Name != "" || decoded.Pass != "" {
		t.Errorf("empty LoginReq should decode to empty strings, got %+v", decoded)
	}
}

// --- LoginRes tests ---

func TestLoginRes_JSONRoundTrip(t *testing.T) {
	original := LoginRes{
		Token:         "tok-xyz",
		Id:            "user-1",
		Name:          "admin",
		Nick:          "Admin User",
		Avatar:        "https://example.com/avatar.png",
		LastLoginTime: "2024-01-15T10:30:00Z",
	}
	bts, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("json.Marshal(LoginRes) error: %v", err)
	}
	var decoded LoginRes
	if err := json.Unmarshal(bts, &decoded); err != nil {
		t.Fatalf("json.Unmarshal(LoginRes) error: %v", err)
	}
	if decoded != original {
		t.Errorf("LoginRes round-trip mismatch: got %+v, want %+v", decoded, original)
	}
}

func TestLoginRes_JSONFieldNames(t *testing.T) {
	res := LoginRes{
		Token:         "tok",
		Id:            "id1",
		Name:          "n",
		Nick:          "nick",
		Avatar:        "av",
		LastLoginTime: "2024-01-01",
	}
	bts, err := json.Marshal(res)
	if err != nil {
		t.Fatalf("json.Marshal error: %v", err)
	}
	s := string(bts)
	expectedKeys := []string{"token", "id", "name", "nick", "avatar", "lastLoginTime"}
	for _, key := range expectedKeys {
		if !contains(s, `"`+key+`"`) {
			t.Errorf("LoginRes JSON should contain key %q, got: %s", key, s)
		}
	}
}

// --- PipelineShow tests ---

func TestPipelineShow_JSONRoundTrip(t *testing.T) {
	original := PipelineShow{
		Id:           "pipe-1",
		Uid:          "user-1",
		Name:         "build-pipeline",
		DisplayName:  "Build Pipeline",
		PipelineType: "standard",
		YmlContent:   "version: 1.0",
		Url:          "https://github.com/test/repo",
		Username:     "user",
		AccessToken:  "tok-123",
	}
	bts, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("json.Marshal(PipelineShow) error: %v", err)
	}
	var decoded PipelineShow
	if err := json.Unmarshal(bts, &decoded); err != nil {
		t.Fatalf("json.Unmarshal(PipelineShow) error: %v", err)
	}
	if decoded != original {
		t.Errorf("PipelineShow round-trip mismatch: got %+v, want %+v", decoded, original)
	}
}

// --- OrgVar tests ---

func TestOrgVar_JSONRoundTrip(t *testing.T) {
	original := OrgVar{
		Aid:     1,
		OrgId:   "org-1",
		Name:    "API_KEY",
		Value:   "secret-value",
		Remarks: "Production API key",
		Public:  false,
	}
	bts, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("json.Marshal(OrgVar) error: %v", err)
	}
	var decoded OrgVar
	if err := json.Unmarshal(bts, &decoded); err != nil {
		t.Fatalf("json.Unmarshal(OrgVar) error: %v", err)
	}
	if decoded != original {
		t.Errorf("OrgVar round-trip mismatch: got %+v, want %+v", decoded, original)
	}
}

func TestOrgVar_PublicFlag(t *testing.T) {
	tests := []struct {
		name   string
		public bool
	}{
		{"public true", true},
		{"public false", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := OrgVar{Name: "VAR", Value: "val", Public: tt.public}
			bts, _ := json.Marshal(v)
			var decoded OrgVar
			_ = json.Unmarshal(bts, &decoded)
			if decoded.Public != tt.public {
				t.Errorf("OrgVar.Public = %v, want %v", decoded.Public, tt.public)
			}
		})
	}
}

// --- PipelineVar tests ---

func TestPipelineVar_JSONRoundTrip(t *testing.T) {
	original := PipelineVar{
		Aid:        10,
		PipelineId: "pipe-42",
		Name:       "DEPLOY_ENV",
		Value:      "production",
		Remarks:    "Deployment environment",
		Public:     true,
	}
	bts, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("json.Marshal(PipelineVar) error: %v", err)
	}
	var decoded PipelineVar
	if err := json.Unmarshal(bts, &decoded); err != nil {
		t.Fatalf("json.Unmarshal(PipelineVar) error: %v", err)
	}
	if decoded != original {
		t.Errorf("PipelineVar round-trip mismatch: got %+v, want %+v", decoded, original)
	}
}

// --- LogOutJson tests ---

func TestLogOutJson_JSONRoundTrip(t *testing.T) {
	now := time.Now().Truncate(time.Second)
	original := LogOutJson{
		Id:      "log-1",
		Content: "build started",
		Times:   now,
		Errs:    false,
	}
	bts, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("json.Marshal(LogOutJson) error: %v", err)
	}
	var decoded LogOutJson
	if err := json.Unmarshal(bts, &decoded); err != nil {
		t.Fatalf("json.Unmarshal(LogOutJson) error: %v", err)
	}
	if decoded.Id != original.Id || decoded.Content != original.Content || decoded.Errs != original.Errs {
		t.Errorf("LogOutJson round-trip mismatch: got %+v, want %+v", decoded, original)
	}
	if !decoded.Times.Equal(original.Times) {
		t.Errorf("LogOutJson.Times mismatch: got %v, want %v", decoded.Times, original.Times)
	}
}

func TestLogOutJson_ErrorFlag(t *testing.T) {
	tests := []struct {
		name string
		errs bool
	}{
		{"no error", false},
		{"has error", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := LogOutJson{Id: "log", Content: "msg", Errs: tt.errs}
			bts, _ := json.Marshal(v)
			var decoded LogOutJson
			_ = json.Unmarshal(bts, &decoded)
			if decoded.Errs != tt.errs {
				t.Errorf("LogOutJson.Errs = %v, want %v", decoded.Errs, tt.errs)
			}
		})
	}
}

// --- LogOutJsonRes tests ---

func TestLogOutJsonRes_JSONRoundTrip(t *testing.T) {
	now := time.Now().Truncate(time.Second)
	original := LogOutJsonRes{
		Id:      "log-2",
		Content: "step completed",
		Times:   now,
		Errs:    false,
		Offset:  1024,
	}
	bts, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("json.Marshal(LogOutJsonRes) error: %v", err)
	}
	var decoded LogOutJsonRes
	if err := json.Unmarshal(bts, &decoded); err != nil {
		t.Fatalf("json.Unmarshal(LogOutJsonRes) error: %v", err)
	}
	if decoded.Id != original.Id || decoded.Content != original.Content ||
		decoded.Errs != original.Errs || decoded.Offset != original.Offset {
		t.Errorf("LogOutJsonRes round-trip mismatch: got %+v, want %+v", decoded, original)
	}
}

func TestLogOutJsonRes_OffsetField(t *testing.T) {
	res := LogOutJsonRes{Offset: 512}
	bts, _ := json.Marshal(res)
	var decoded map[string]any
	_ = json.Unmarshal(bts, &decoded)
	off, ok := decoded["offset"]
	if !ok {
		t.Fatal("LogOutJsonRes JSON should contain 'offset' key")
	}
	// JSON numbers decode as float64
	if off.(float64) != 512 {
		t.Errorf("offset = %v, want 512", off)
	}
}

// --- NewPipelineVar tests ---

func TestNewPipelineVar_JSONRoundTrip(t *testing.T) {
	original := NewPipelineVar{
		Name:    "DB_HOST",
		Value:   "localhost:5432",
		Remarks: "Database host",
		Public:  true,
	}
	bts, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("json.Marshal(NewPipelineVar) error: %v", err)
	}
	var decoded NewPipelineVar
	if err := json.Unmarshal(bts, &decoded); err != nil {
		t.Fatalf("json.Unmarshal(NewPipelineVar) error: %v", err)
	}
	if decoded != original {
		t.Errorf("NewPipelineVar round-trip mismatch: got %+v, want %+v", decoded, original)
	}
}

// --- NewPipeline with Vars tests ---

func TestNewPipeline_WithVars(t *testing.T) {
	p := &NewPipeline{
		Name:    "deploy",
		Content: "stages: []",
		Vars: []*NewPipelineVar{
			{Name: "ENV", Value: "prod", Public: false},
			{Name: "REGION", Value: "us-east-1", Public: true},
		},
	}
	if !p.Check() {
		t.Error("NewPipeline with Name and Content should pass Check()")
	}
	if len(p.Vars) != 2 {
		t.Errorf("expected 2 vars, got %d", len(p.Vars))
	}
	bts, err := json.Marshal(p)
	if err != nil {
		t.Fatalf("json.Marshal error: %v", err)
	}
	var decoded NewPipeline
	if err := json.Unmarshal(bts, &decoded); err != nil {
		t.Fatalf("json.Unmarshal error: %v", err)
	}
	if len(decoded.Vars) != 2 {
		t.Errorf("decoded vars length = %d, want 2", len(decoded.Vars))
	}
	if decoded.Vars[0].Name != "ENV" || decoded.Vars[1].Name != "REGION" {
		t.Errorf("vars names mismatch: got %q and %q", decoded.Vars[0].Name, decoded.Vars[1].Name)
	}
}

// --- TriggerParam additional edge case tests ---

func TestTriggerParam_AllFieldsSet(t *testing.T) {
	tp := TriggerParam{
		Id:         "trigger-1",
		PipelineId: "pipe-1",
		Types:      "webhook",
		Name:       "push-trigger",
		Desc:       "Triggers on push events",
		Params:     `{"events":["push"]}`,
		Enabled:    true,
	}
	if err := tp.Check(); err != nil {
		t.Errorf("fully-populated TriggerParam.Check() should succeed, got: %v", err)
	}
}

func TestTriggerParam_EnabledField(t *testing.T) {
	tests := []struct {
		name    string
		enabled bool
	}{
		{"enabled", true},
		{"disabled", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tp := TriggerParam{
				PipelineId: "p1",
				Types:      "webhook",
				Name:       "trig",
				Params:     `{}`,
				Enabled:    tt.enabled,
			}
			bts, _ := json.Marshal(tp)
			var decoded TriggerParam
			_ = json.Unmarshal(bts, &decoded)
			if decoded.Enabled != tt.enabled {
				t.Errorf("TriggerParam.Enabled = %v, want %v", decoded.Enabled, tt.enabled)
			}
		})
	}
}

// helper
func contains(s, substr string) bool {
	return len(s) >= len(substr) && searchSubstring(s, substr)
}

func searchSubstring(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
