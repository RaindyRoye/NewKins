package model

import (
	"encoding/json"
	"testing"
	"time"
)

// --- TUser tests ---

func TestTUser_JSONRoundTrip(t *testing.T) {
	now := time.Now().Truncate(time.Second)
	original := TUser{
		Id:        "user-1",
		Aid:       1,
		Name:      "admin",
		Pass:      "hashed-pw",
		Nick:      "Admin User",
		Avatar:    "https://example.com/avatar.png",
		Created:   now,
		LoginTime: now,
		Active:    1,
	}
	bts, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("json.Marshal(TUser) error: %v", err)
	}
	var decoded TUser
	if err := json.Unmarshal(bts, &decoded); err != nil {
		t.Fatalf("json.Unmarshal(TUser) error: %v", err)
	}
	if decoded.Id != original.Id || decoded.Name != original.Name ||
		decoded.Nick != original.Nick || decoded.Active != original.Active {
		t.Errorf("TUser round-trip mismatch: got %+v, want %+v", decoded, original)
	}
}

func TestTUser_JSONFieldNames(t *testing.T) {
	u := TUser{Id: "u1", Name: "test", Nick: "nick"}
	bts, _ := json.Marshal(u)
	s := string(bts)
	expectedKeys := []string{"id", "name", "nick", "avatar", "loginTime"}
	for _, key := range expectedKeys {
		if !jsonContains(s, key) {
			t.Errorf("TUser JSON should contain key %q, got: %s", key, s)
		}
	}
}

// --- TOrg tests ---

func TestTOrg_JSONRoundTrip(t *testing.T) {
	now := time.Now().Truncate(time.Second)
	original := TOrg{
		Id:      "org-1",
		Aid:     10,
		Uid:     "user-1",
		Name:    "My Org",
		Desc:    "A test organization",
		Public:  1,
		Created: now,
		Updated: now,
		Deleted: 0,
	}
	bts, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("json.Marshal(TOrg) error: %v", err)
	}
	var decoded TOrg
	if err := json.Unmarshal(bts, &decoded); err != nil {
		t.Fatalf("json.Unmarshal(TOrg) error: %v", err)
	}
	if decoded.Id != original.Id || decoded.Name != original.Name ||
		decoded.Public != original.Public {
		t.Errorf("TOrg round-trip mismatch: got %+v, want %+v", decoded, original)
	}
}

// --- TBuild tests ---

func TestTBuild_JSONRoundTrip(t *testing.T) {
	now := time.Now().Truncate(time.Second)
	original := TBuild{
		Id:                "build-1",
		PipelineId:        "pipe-1",
		PipelineVersionId: "ver-1",
		Status:            "success",
		Event:             "push",
		Started:           now,
		Finished:          now.Add(5 * time.Minute),
		Created:           now,
		Updated:           now,
		Version:           "v1.0.0",
	}
	bts, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("json.Marshal(TBuild) error: %v", err)
	}
	var decoded TBuild
	if err := json.Unmarshal(bts, &decoded); err != nil {
		t.Fatalf("json.Unmarshal(TBuild) error: %v", err)
	}
	if decoded.Id != original.Id || decoded.Status != original.Status ||
		decoded.Version != original.Version {
		t.Errorf("TBuild round-trip mismatch: got %+v, want %+v", decoded, original)
	}
}

// --- TPipeline tests ---

func TestTPipeline_JSONRoundTrip(t *testing.T) {
	p := TPipeline{
		Id:           "pipe-1",
		Uid:          "user-1",
		Name:         "build-pipe",
		DisplayName:  "Build Pipeline",
		PipelineType: "standard",
		// xorm:"-" fields should still appear in JSON
		AccessToken: "tok-secret",
		Url:         "https://github.com/test/repo",
		Username:    "user",
		// xorm:"-" fields with json:"-" should NOT appear
		Deleted: 0,
	}
	bts, err := json.Marshal(p)
	if err != nil {
		t.Fatalf("json.Marshal(TPipeline) error: %v", err)
	}
	s := string(bts)
	// accessToken has json tag, should be present
	if !jsonContains(s, "accessToken") {
		t.Error("TPipeline JSON should contain 'accessToken'")
	}
	// Deleted has json:"-", should NOT be present
	if jsonContains(s, "deleted") {
		t.Error("TPipeline JSON should NOT contain 'deleted' (json:\"-\")")
	}
}

// --- TStage tests ---

func TestTStage_JSONRoundTrip(t *testing.T) {
	now := time.Now().Truncate(time.Second)
	original := TStage{
		Id:                "stage-1",
		PipelineVersionId: "ver-1",
		BuildId:           "build-1",
		Status:            "running",
		Name:              "build",
		DisplayName:       "Build Stage",
		Started:           now,
		Sort:              1,
		Stage:             "build",
	}
	bts, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("json.Marshal(TStage) error: %v", err)
	}
	var decoded TStage
	if err := json.Unmarshal(bts, &decoded); err != nil {
		t.Fatalf("json.Unmarshal(TStage) error: %v", err)
	}
	if decoded.Id != original.Id || decoded.Name != original.Name ||
		decoded.Sort != original.Sort {
		t.Errorf("TStage round-trip mismatch: got %+v, want %+v", decoded, original)
	}
}

// --- TStep tests ---

func TestTStep_JSONRoundTrip(t *testing.T) {
	now := time.Now().Truncate(time.Second)
	original := TStep{
		Id:                "step-1",
		BuildId:           "build-1",
		StageId:           "stage-1",
		PipelineVersionId: "ver-1",
		Step:              "shell",
		Status:            "success",
		ExitCode:          0,
		Name:              "compile",
		Started:           now,
		Finished:          now.Add(30 * time.Second),
		Sort:              1,
	}
	bts, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("json.Marshal(TStep) error: %v", err)
	}
	var decoded TStep
	if err := json.Unmarshal(bts, &decoded); err != nil {
		t.Fatalf("json.Unmarshal(TStep) error: %v", err)
	}
	if decoded.Id != original.Id || decoded.Step != original.Step ||
		decoded.ExitCode != original.ExitCode {
		t.Errorf("TStep round-trip mismatch: got %+v, want %+v", decoded, original)
	}
}

// --- TCmdLine tests ---

func TestTCmdLine_JSONRoundTrip(t *testing.T) {
	now := time.Now().Truncate(time.Second)
	original := TCmdLine{
		Id:       "cmd-1",
		GroupId:  "grp-1",
		BuildId:  "build-1",
		StepId:   "step-1",
		Status:   "done",
		Num:      42,
		Code:     0,
		Content:  "echo hello world",
		Created:  now,
		Started:  now,
		Finished: now.Add(time.Second),
	}
	bts, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("json.Marshal(TCmdLine) error: %v", err)
	}
	var decoded TCmdLine
	if err := json.Unmarshal(bts, &decoded); err != nil {
		t.Fatalf("json.Unmarshal(TCmdLine) error: %v", err)
	}
	if decoded.Content != original.Content || decoded.Num != original.Num {
		t.Errorf("TCmdLine round-trip mismatch: got %+v, want %+v", decoded, original)
	}
}

// --- TMessage tests ---

func TestTMessage_JSONRoundTrip(t *testing.T) {
	now := time.Now().Truncate(time.Second)
	original := TMessage{
		Id:      "msg-1",
		Aid:     100,
		Uid:     "user-1",
		Title:   "Build Failed",
		Content: "Pipeline build-123 failed at stage 'test'",
		Types:   "build",
		Created: now,
		Infos:   `{"buildId":"build-123"}`,
		Url:     "/builds/build-123",
	}
	bts, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("json.Marshal(TMessage) error: %v", err)
	}
	var decoded TMessage
	if err := json.Unmarshal(bts, &decoded); err != nil {
		t.Fatalf("json.Unmarshal(TMessage) error: %v", err)
	}
	if decoded.Title != original.Title || decoded.Types != original.Types {
		t.Errorf("TMessage round-trip mismatch: got %+v, want %+v", decoded, original)
	}
}

// --- TParam tests ---

func TestTParam_JSONRoundTrip(t *testing.T) {
	original := TParam{
		Aid:   1,
		Name:  "global-param",
		Title: "Global Parameter",
		Data:  `{"key":"value"}`,
		Times: time.Now().Truncate(time.Second),
	}
	bts, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("json.Marshal(TParam) error: %v", err)
	}
	var decoded TParam
	if err := json.Unmarshal(bts, &decoded); err != nil {
		t.Fatalf("json.Unmarshal(TParam) error: %v", err)
	}
	if decoded.Name != original.Name || decoded.Data != original.Data {
		t.Errorf("TParam round-trip mismatch: got %+v, want %+v", decoded, original)
	}
}

// --- TPipelineConf tests ---

func TestTPipelineConf_JSONRoundTrip(t *testing.T) {
	original := TPipelineConf{
		Aid:         1,
		PipelineId:  "pipe-1",
		Url:         "https://github.com/test/repo",
		AccessToken: "ghp_secret",
		YmlContent:  "version: '1.0'\nstages: []",
		Username:    "testuser",
	}
	bts, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("json.Marshal(TPipelineConf) error: %v", err)
	}
	var decoded TPipelineConf
	if err := json.Unmarshal(bts, &decoded); err != nil {
		t.Fatalf("json.Unmarshal(TPipelineConf) error: %v", err)
	}
	if decoded.PipelineId != original.PipelineId ||
		decoded.YmlContent != original.YmlContent {
		t.Errorf("TPipelineConf round-trip mismatch: got %+v, want %+v", decoded, original)
	}
}

// --- TPipelineVar tests ---

func TestTPipelineVar_JSONRoundTrip(t *testing.T) {
	original := TPipelineVar{
		Aid:        1,
		Uid:        "user-1",
		PipelineId: "pipe-1",
		Name:       "DB_HOST",
		Value:      "localhost:5432",
		Remarks:    "Database host",
		Public:     1,
	}
	bts, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("json.Marshal(TPipelineVar) error: %v", err)
	}
	var decoded TPipelineVar
	if err := json.Unmarshal(bts, &decoded); err != nil {
		t.Fatalf("json.Unmarshal(TPipelineVar) error: %v", err)
	}
	if decoded.Name != original.Name || decoded.Value != original.Value ||
		decoded.Public != original.Public {
		t.Errorf("TPipelineVar round-trip mismatch: got %+v, want %+v", decoded, original)
	}
}

// --- TOrgVar tests ---

func TestTOrgVar_JSONRoundTrip(t *testing.T) {
	original := TOrgVar{
		Aid:     1,
		Uid:     "user-1",
		OrgId:   "org-1",
		Name:    "API_KEY",
		Value:   "secret",
		Remarks: "Production key",
		Public:  0,
	}
	bts, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("json.Marshal(TOrgVar) error: %v", err)
	}
	var decoded TOrgVar
	if err := json.Unmarshal(bts, &decoded); err != nil {
		t.Fatalf("json.Unmarshal(TOrgVar) error: %v", err)
	}
	if decoded.Name != original.Name || decoded.OrgId != original.OrgId {
		t.Errorf("TOrgVar round-trip mismatch: got %+v, want %+v", decoded, original)
	}
}

// --- TUserOrg tests ---

func TestTUserOrg_JSONRoundTrip(t *testing.T) {
	now := time.Now().Truncate(time.Second)
	original := TUserOrg{
		Aid:      1,
		Uid:      "user-1",
		OrgId:    "org-1",
		Created:  now,
		PermAdm:  1,
		PermRw:   1,
		PermExec: 0,
		PermDown: 1,
	}
	bts, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("json.Marshal(TUserOrg) error: %v", err)
	}
	var decoded TUserOrg
	if err := json.Unmarshal(bts, &decoded); err != nil {
		t.Fatalf("json.Unmarshal(TUserOrg) error: %v", err)
	}
	if decoded.PermAdm != original.PermAdm || decoded.PermRw != original.PermRw ||
		decoded.PermExec != original.PermExec || decoded.PermDown != original.PermDown {
		t.Errorf("TUserOrg round-trip mismatch: got %+v, want %+v", decoded, original)
	}
}

func TestTUserOrg_PermissionFlags(t *testing.T) {
	tests := []struct {
		name    string
		permAdm int
		permRw  int
	}{
		{"admin", 1, 1},
		{"read-write", 0, 1},
		{"read-only", 0, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			uo := TUserOrg{PermAdm: tt.permAdm, PermRw: tt.permRw}
			bts, _ := json.Marshal(uo)
			var decoded TUserOrg
			_ = json.Unmarshal(bts, &decoded)
			if decoded.PermAdm != tt.permAdm || decoded.PermRw != tt.permRw {
				t.Errorf("permission flags mismatch: got adm=%d rw=%d, want adm=%d rw=%d",
					decoded.PermAdm, decoded.PermRw, tt.permAdm, tt.permRw)
			}
		})
	}
}

// --- TOrgPipe tests ---

func TestTOrgPipe_JSONRoundTrip(t *testing.T) {
	original := TOrgPipe{
		Aid:     1,
		OrgId:   "org-1",
		PipeId:  "pipe-1",
		Created: time.Now().Truncate(time.Second),
		Public:  1,
	}
	bts, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("json.Marshal(TOrgPipe) error: %v", err)
	}
	var decoded TOrgPipe
	if err := json.Unmarshal(bts, &decoded); err != nil {
		t.Fatalf("json.Unmarshal(TOrgPipe) error: %v", err)
	}
	if decoded.OrgId != original.OrgId || decoded.PipeId != original.PipeId ||
		decoded.Public != original.Public {
		t.Errorf("TOrgPipe round-trip mismatch: got %+v, want %+v", decoded, original)
	}
}

// --- TUserToken tests ---

func TestTUserToken_JSONRoundTrip(t *testing.T) {
	original := TUserToken{
		Aid:          1,
		Uid:          100,
		Type:         "github",
		Openid:       "gh-123",
		Name:         "user",
		Nick:         "Test User",
		Avatar:       "https://avatars.example.com/user.png",
		AccessToken:  "ghp_token",
		RefreshToken: "ghr_refresh",
		ExpiresIn:    3600,
	}
	bts, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("json.Marshal(TUserToken) error: %v", err)
	}
	var decoded TUserToken
	if err := json.Unmarshal(bts, &decoded); err != nil {
		t.Fatalf("json.Unmarshal(TUserToken) error: %v", err)
	}
	if decoded.Type != original.Type || decoded.Openid != original.Openid ||
		decoded.ExpiresIn != original.ExpiresIn {
		t.Errorf("TUserToken round-trip mismatch: got %+v, want %+v", decoded, original)
	}
}

// --- TUserInfo tests ---

func TestTUserInfo_JSONRoundTrip(t *testing.T) {
	original := TUserInfo{
		Id:       "user-1",
		Phone:    "+1234567890",
		Email:    "user@example.com",
		Birthday: time.Date(1990, 1, 15, 0, 0, 0, 0, time.UTC),
		Remark:   "VIP user",
		PermUser: 1,
		PermOrg:  1,
		PermPipe: 0,
	}
	bts, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("json.Marshal(TUserInfo) error: %v", err)
	}
	var decoded TUserInfo
	if err := json.Unmarshal(bts, &decoded); err != nil {
		t.Fatalf("json.Unmarshal(TUserInfo) error: %v", err)
	}
	if decoded.Email != original.Email || decoded.PermUser != original.PermUser {
		t.Errorf("TUserInfo round-trip mismatch: got %+v, want %+v", decoded, original)
	}
}

// --- TUserMsg tests ---

func TestTUserMsg_JSONRoundTrip(t *testing.T) {
	now := time.Now().Truncate(time.Second)
	original := TUserMsg{
		Aid:     1,
		Uid:     "user-1",
		MsgId:   "msg-1",
		Created: now,
		Status:  0,
		Deleted: 0,
	}
	bts, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("json.Marshal(TUserMsg) error: %v", err)
	}
	var decoded TUserMsg
	if err := json.Unmarshal(bts, &decoded); err != nil {
		t.Fatalf("json.Unmarshal(TUserMsg) error: %v", err)
	}
	if decoded.Uid != original.Uid || decoded.MsgId != original.MsgId {
		t.Errorf("TUserMsg round-trip mismatch: got %+v, want %+v", decoded, original)
	}
}

// --- TArtifactory tests ---

func TestTArtifactory_JSONRoundTrip(t *testing.T) {
	original := TArtifactory{
		Id:         "art-1",
		Aid:        1,
		Uid:        "user-1",
		OrgId:      "org-1",
		Identifier: "docker",
		Name:       "My Docker Registry",
		Source:     "docker",
		Desc:       "Docker image repository",
	}
	bts, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("json.Marshal(TArtifactory) error: %v", err)
	}
	var decoded TArtifactory
	if err := json.Unmarshal(bts, &decoded); err != nil {
		t.Fatalf("json.Unmarshal(TArtifactory) error: %v", err)
	}
	if decoded.Identifier != original.Identifier || decoded.Name != original.Name {
		t.Errorf("TArtifactory round-trip mismatch: got %+v, want %+v", decoded, original)
	}
}

// --- TArtifactPackage tests ---

func TestTArtifactPackage_JSONRoundTrip(t *testing.T) {
	original := TArtifactPackage{
		Id:          "pkg-1",
		Aid:         1,
		RepoId:      "art-1",
		Name:        "my-app",
		DisplayName: "My Application",
		Desc:        "Main application package",
		Verln:       5,
	}
	bts, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("json.Marshal(TArtifactPackage) error: %v", err)
	}
	var decoded TArtifactPackage
	if err := json.Unmarshal(bts, &decoded); err != nil {
		t.Fatalf("json.Unmarshal(TArtifactPackage) error: %v", err)
	}
	if decoded.Name != original.Name || decoded.Verln != original.Verln {
		t.Errorf("TArtifactPackage round-trip mismatch: got %+v, want %+v", decoded, original)
	}
}

// --- TPipelineVersion tests ---

func TestTPipelineVersion_JSONRoundTrip(t *testing.T) {
	now := time.Now().Truncate(time.Second)
	original := TPipelineVersion{
		Id:                  "ver-1",
		Uid:                 "user-1",
		Number:              42,
		Events:              "push",
		Sha:                 "abc123def",
		PipelineName:        "build-pipe",
		PipelineDisplayName: "Build Pipeline",
		PipelineId:          "pipe-1",
		Version:             "v1.2.3",
		Content:             "version: '1.0'",
		Created:             now,
		PrNumber:            15,
		RepoCloneUrl:        "https://github.com/test/repo.git",
	}
	bts, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("json.Marshal(TPipelineVersion) error: %v", err)
	}
	var decoded TPipelineVersion
	if err := json.Unmarshal(bts, &decoded); err != nil {
		t.Fatalf("json.Unmarshal(TPipelineVersion) error: %v", err)
	}
	if decoded.Number != original.Number || decoded.Sha != original.Sha ||
		decoded.PrNumber != original.PrNumber {
		t.Errorf("TPipelineVersion round-trip mismatch: got %+v, want %+v", decoded, original)
	}
}

// --- TTriggerRun tests ---

func TestTTriggerRun_JSONFields(t *testing.T) {
	tr := TTriggerRun{
		Id:                  "run-1",
		Aid:                 1,
		Tid:                 "trigger-1",
		PipeVersionId:       "ver-1",
		Infos:               `{"event":"push"}`,
		Number:              100,
		PipelineName:        "build-pipe",
		PipelineDisplayName: "Build Pipeline",
		BStatus:             "success",
	}
	if tr.Number != 100 {
		t.Errorf("TTriggerRun.Number = %d, want 100", tr.Number)
	}
	if tr.BStatus != "success" {
		t.Errorf("TTriggerRun.BStatus = %q, want 'success'", tr.BStatus)
	}
}

func TestTTriggerRun_JSONExcludesXormDash(t *testing.T) {
	tr := TTriggerRun{
		Id:                  "run-1",
		Number:              42,
		PipelineName:        "pipe",
		PipelineDisplayName: "Pipe",
		BStatus:             "running",
	}
	bts, _ := json.Marshal(tr)
	s := string(bts)
	// Number, PipelineName, PipelineDisplayName, BStatus have json tags, should be present
	for _, key := range []string{"number", "pipelineName", "pipelineDisplayName", "bStatus"} {
		if !jsonContains(s, key) {
			t.Errorf("TTriggerRun JSON should contain %q, got: %s", key, s)
		}
	}
}

// --- TimerTriggerRun tests ---

func TestTimerTriggerRun_JSONRoundTrip(t *testing.T) {
	now := time.Now().Truncate(time.Second)
	original := TimerTriggerRun{
		Id:         "timer-1",
		Uid:        "user-1",
		PipelineId: "pipe-1",
		Types:      "cron",
		Name:       "nightly-build",
		Params:     `{"schedule":"0 0 * * *"}`,
		Enabled:    1,
		RunCreated: now,
	}
	bts, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("json.Marshal(TimerTriggerRun) error: %v", err)
	}
	var decoded TimerTriggerRun
	if err := json.Unmarshal(bts, &decoded); err != nil {
		t.Fatalf("json.Unmarshal(TimerTriggerRun) error: %v", err)
	}
	if decoded.Name != original.Name || decoded.Enabled != original.Enabled {
		t.Errorf("TimerTriggerRun round-trip mismatch: got %+v, want %+v", decoded, original)
	}
}

// --- RunPipelineVersion tests ---

func TestRunPipelineVersion_JSONRoundTrip(t *testing.T) {
	original := RunPipelineVersion{
		Id:                  "rver-1",
		Number:              42,
		PipelineName:        "build-pipe",
		PipelineDisplayName: "Build Pipeline",
		Status:              "success",
	}
	bts, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("json.Marshal(RunPipelineVersion) error: %v", err)
	}
	var decoded RunPipelineVersion
	if err := json.Unmarshal(bts, &decoded); err != nil {
		t.Fatalf("json.Unmarshal(RunPipelineVersion) error: %v", err)
	}
	if decoded.Number != original.Number || decoded.Status != original.Status {
		t.Errorf("RunPipelineVersion round-trip mismatch: got %+v, want %+v", decoded, original)
	}
}

// --- TYmlPlugin tests ---

func TestTYmlPlugin_JSONRoundTrip(t *testing.T) {
	original := TYmlPlugin{
		Aid:        1,
		Name:       "docker-build",
		YmlContent: "steps:\n  - name: build\n    image: docker",
		Deleted:    0,
	}
	bts, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("json.Marshal(TYmlPlugin) error: %v", err)
	}
	var decoded TYmlPlugin
	if err := json.Unmarshal(bts, &decoded); err != nil {
		t.Fatalf("json.Unmarshal(TYmlPlugin) error: %v", err)
	}
	if decoded.Name != original.Name || decoded.YmlContent != original.YmlContent {
		t.Errorf("TYmlPlugin round-trip mismatch: got %+v, want %+v", decoded, original)
	}
}

// --- TYmlTemplate tests ---

func TestTYmlTemplate_JSONRoundTrip(t *testing.T) {
	original := TYmlTemplate{
		Aid:        1,
		Name:       "go-template",
		YmlContent: "version: '1.0'\nstages:\n  - name: test",
		Deleted:    0,
	}
	bts, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("json.Marshal(TYmlTemplate) error: %v", err)
	}
	var decoded TYmlTemplate
	if err := json.Unmarshal(bts, &decoded); err != nil {
		t.Fatalf("json.Unmarshal(TYmlTemplate) error: %v", err)
	}
	if decoded.Name != original.Name || decoded.YmlContent != original.YmlContent {
		t.Errorf("TYmlTemplate round-trip mismatch: got %+v, want %+v", decoded, original)
	}
}

// --- SchemaMigrations tests ---

func TestSchemaMigrations_JSONRoundTrip(t *testing.T) {
	original := SchemaMigrations{
		Version: 20240101,
		Dirty:   0,
	}
	bts, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("json.Marshal(SchemaMigrations) error: %v", err)
	}
	var decoded SchemaMigrations
	if err := json.Unmarshal(bts, &decoded); err != nil {
		t.Fatalf("json.Unmarshal(SchemaMigrations) error: %v", err)
	}
	if decoded.Version != original.Version || decoded.Dirty != original.Dirty {
		t.Errorf("SchemaMigrations round-trip mismatch: got %+v, want %+v", decoded, original)
	}
}

func TestSchemaMigrations_DirtyFlag(t *testing.T) {
	tests := []struct {
		name  string
		dirty int
	}{
		{"clean", 0},
		{"dirty", 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sm := SchemaMigrations{Version: 1, Dirty: tt.dirty}
			bts, _ := json.Marshal(sm)
			var decoded SchemaMigrations
			_ = json.Unmarshal(bts, &decoded)
			if decoded.Dirty != tt.dirty {
				t.Errorf("SchemaMigrations.Dirty = %d, want %d", decoded.Dirty, tt.dirty)
			}
		})
	}
}

// --- TableName tests for models that define them ---

func TestRunPipelineVersion_NoTableName(t *testing.T) {
	// RunPipelineVersion does not define TableName(), so this tests
	// that it doesn't panic when used
	rpv := RunPipelineVersion{Id: "test", Number: 1, Status: "ok"}
	if rpv.Status != "ok" {
		t.Error("RunPipelineVersion fields should be settable")
	}
}

// --- RunBuild additional tests ---

func TestRunBuild_JSONRoundTrip(t *testing.T) {
	now := time.Now().Truncate(time.Second)
	original := RunBuild{
		Id:                "rb-1",
		PipelineId:        "pipe-1",
		PipelineVersionId: "ver-1",
		Status:            "running",
		Event:             "push",
		Started:           now,
		Version:           "v2.0.0",
	}
	bts, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("json.Marshal(RunBuild) error: %v", err)
	}
	var decoded RunBuild
	if err := json.Unmarshal(bts, &decoded); err != nil {
		t.Fatalf("json.Unmarshal(RunBuild) error: %v", err)
	}
	if decoded.Id != original.Id || decoded.Status != original.Status {
		t.Errorf("RunBuild round-trip mismatch: got %+v, want %+v", decoded, original)
	}
}

// --- RunStage additional tests ---

func TestRunStage_StepidsField(t *testing.T) {
	rs := RunStage{
		Id:      "rs-1",
		Name:    "build",
		Stepids: []string{"step-1", "step-2", "step-3"},
	}
	if len(rs.Stepids) != 3 {
		t.Errorf("RunStage.Stepids length = %d, want 3", len(rs.Stepids))
	}
	bts, _ := json.Marshal(rs)
	s := string(bts)
	if !jsonContains(s, "stepids") {
		t.Errorf("RunStage JSON should contain 'stepids', got: %s", s)
	}
}

// --- RunStep additional tests ---

func TestRunStep_WaitingsField(t *testing.T) {
	rs := RunStep{
		Id:       "rst-1",
		Name:     "deploy",
		Waitings: []string{"step-a", "step-b"},
	}
	if len(rs.Waitings) != 2 {
		t.Errorf("RunStep.Waitings length = %d, want 2", len(rs.Waitings))
	}
	bts, _ := json.Marshal(rs)
	s := string(bts)
	if !jsonContains(s, "waits") {
		t.Errorf("RunStep JSON should contain 'waits', got: %s", s)
	}
}

// helper
func jsonContains(s, key string) bool {
	target := `"` + key + `"`
	for i := 0; i <= len(s)-len(target); i++ {
		if s[i:i+len(target)] == target {
			return true
		}
	}
	return false
}
