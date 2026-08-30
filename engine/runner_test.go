package engine

import "testing"

func TestBaseRunner_ServerInfo(t *testing.T) {
	r := &baseRunner{}
	info, err := r.ServerInfo()
	if err != nil {
		t.Errorf("ServerInfo() error = %v", err)
	}
	if info == nil {
		t.Error("ServerInfo() returned nil")
	}
}

func TestBaseRunner_FindJobId_EmptyParams(t *testing.T) {
	r := &baseRunner{}
	tests := []struct {
		name    string
		buildID string
		stgNm   string
		stpNm   string
	}{
		{"empty buildID", "", "stage", "step"},
		{"empty stage", "build", "", "step"},
		{"empty step", "build", "stage", ""},
		{"all empty", "", "", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			id, ok := r.FindJobId(tt.buildID, tt.stgNm, tt.stpNm)
			if ok {
				t.Errorf("FindJobId() = (%s, true), want false", id)
			}
		})
	}
}

func TestBaseRunner_GetEnv_EmptyParams(t *testing.T) {
	r := &baseRunner{}
	tests := []struct {
		name    string
		buildID string
		jobId   string
		key     string
	}{
		{"empty jobId", "build", "", "key"},
		{"empty key", "build", "job", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			val, ok := r.GetEnv(tt.buildID, tt.jobId, tt.key)
			if ok {
				t.Errorf("GetEnv() = (%s, true), want false", val)
			}
		})
	}
}

func TestBaseRunner_ReadDir_EmptyParams(t *testing.T) {
	r := &baseRunner{}
	_, err := r.ReadDir(1, "", "path")
	if err == nil {
		t.Error("ReadDir() with empty buildID should error")
	}
	_, err = r.ReadDir(1, "build", "")
	if err == nil {
		t.Error("ReadDir() with empty path should error")
	}
}

func TestBaseRunner_ReadFile_EmptyParams(t *testing.T) {
	r := &baseRunner{}
	_, _, err := r.ReadFile(1, "", "path", 0)
	if err == nil {
		t.Error("ReadFile() with empty buildID should error")
	}
	_, _, err = r.ReadFile(1, "build", "", 0)
	if err == nil {
		t.Error("ReadFile() with empty path should error")
	}
}

func TestBaseRunner_StatFile_EmptyParams(t *testing.T) {
	r := &baseRunner{}
	_, err := r.StatFile(1, "build", "", "dir", "path")
	if err == nil {
		t.Error("StatFile() with empty jobId should error")
	}
	_, err = r.StatFile(1, "build", "job", "dir", "")
	if err == nil {
		t.Error("StatFile() with empty path should error")
	}
}

func TestBaseRunner_UploadFile_EmptyParams(t *testing.T) {
	r := &baseRunner{}
	_, err := r.UploadFile(1, "build", "", "dir", "path", 0)
	if err == nil {
		t.Error("UploadFile() with empty jobId should error")
	}
	_, err = r.UploadFile(1, "build", "job", "dir", "", 0)
	if err == nil {
		t.Error("UploadFile() with empty path should error")
	}
}

func TestBaseRunner_FindArtVersionId_EmptyParams(t *testing.T) {
	r := &baseRunner{}
	_, err := r.FindArtVersionId("", "idnt", "name")
	if err == nil {
		t.Error("FindArtVersionId() with empty buildID should error")
	}
}

func TestBaseRunner_NewArtVersionId_EmptyParams(t *testing.T) {
	r := &baseRunner{}
	_, err := r.NewArtVersionId("", "idnt", "name")
	if err == nil {
		t.Error("NewArtVersionId() with empty buildID should error")
	}
}

func TestBaseRunner_GenEnv_NilParams(t *testing.T) {
	r := &baseRunner{}
	err := r.GenEnv("build", "", nil)
	if err == nil {
		t.Error("GenEnv() with empty jobId should error")
	}
}
