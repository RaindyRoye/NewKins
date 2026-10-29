package route

import (
	"net/http"
	"testing"

	"github.com/gokins/gokins/bean"
	"github.com/gokins/gokins/comm"
	"github.com/gokins/gokins/model"
	hbtp "github.com/mgr9525/HyperByte-Transfer-Protocol"
)

// Valid pipeline YAML for testing
const validPipelineYAML = `version: 1.0
stages:
  - name: build
    steps:
      - step: shell
        name: echo
        commands: echo "Hello"
`

// --- Pipeline.new tests ---

func TestPipelineNew_InvalidParams(t *testing.T) {
	setupPipelineTestDB(t)
	c, w := makePipelineTestContext(t, nil)
	ctrl := PipelineController{}

	// Empty name
	npipe := &bean.NewPipeline{Name: "", Content: validPipelineYAML}
	ctrl.new(c, npipe)
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for empty name, got %d: %s", w.Code, w.Body.String())
	}

	// Empty content
	c, w = makePipelineTestContext(t, nil)
	npipe = &bean.NewPipeline{Name: "test", Content: ""}
	ctrl.new(c, npipe)
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for empty content, got %d: %s", w.Code, w.Body.String())
	}
}

func TestPipelineNew_InvalidYAML(t *testing.T) {
	setupPipelineTestDB(t)
	c, w := makePipelineTestContext(t, nil)
	ctrl := PipelineController{}

	npipe := &bean.NewPipeline{Name: "test", Content: "invalid: yaml: [[["}
	ctrl.new(c, npipe)
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for invalid YAML, got %d: %s", w.Code, w.Body.String())
	}
}

func TestPipelineNew_EmptyStages(t *testing.T) {
	setupPipelineTestDB(t)
	c, w := makePipelineTestContext(t, nil)
	ctrl := PipelineController{}

	// Valid YAML but empty stages
	npipe := &bean.NewPipeline{Name: "test", Content: "version: 1.0\n"}
	ctrl.new(c, npipe)
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for empty stages, got %d: %s", w.Code, w.Body.String())
	}
}

// --- Pipeline.save tests ---

func TestPipelineSave_EmptyPipelineId(t *testing.T) {
	setupPipelineTestDB(t)
	ctrl := PipelineController{}
	m := &hbtp.Map{}
	m.Set("pipelineId", "")
	c, w := makePipelineTestContext(t, m)
	ctrl.save(c, m)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for empty pipelineId, got %d", w.Code)
	}
}

func TestPipelineSave_NonexistentPipeline(t *testing.T) {
	setupPipelineTestDB(t)
	ctrl := PipelineController{}
	m := &hbtp.Map{}
	m.Set("pipelineId", "nonexistent-id")
	m.Set("content", validPipelineYAML)
	c, w := makePipelineTestContext(t, m)
	ctrl.save(c, m)

	// Should get 403/405 since user doesn't have write permission for nonexistent pipeline
	if w.Code != http.StatusMethodNotAllowed && w.Code != http.StatusNotFound {
		t.Errorf("expected 405 or 404 for nonexistent pipeline, got %d: %s", w.Code, w.Body.String())
	}
}

func TestPipelineSave_Success(t *testing.T) {
	setupPipelineTestDB(t)

	// Insert a test pipeline owned by test-user
	_, err := comm.Db.Exec(`INSERT INTO t_pipeline (id, uid, name, display_name, deleted) 
		VALUES ('save-pipe-id', 'test-user', 'save-test', 'Save Test', 0)`)
	if err != nil {
		t.Fatalf("insert pipeline: %v", err)
	}
	_, err = comm.Db.Exec(`INSERT INTO t_pipeline_conf (pipeline_id, yml_content, url) 
		VALUES ('save-pipe-id', 'old content', 'http://old')`)
	if err != nil {
		t.Fatalf("insert pipeline_conf: %v", err)
	}

	ctrl := PipelineController{}
	m := &hbtp.Map{}
	m.Set("pipelineId", "save-pipe-id")
	m.Set("name", "updated-name")
	m.Set("displayName", "Updated Name")
	m.Set("content", validPipelineYAML)
	m.Set("url", "http://new-url")
	m.Set("username", "newuser")
	m.Set("accessToken", "newtoken")
	c, w := makePipelineTestContext(t, m)
	ctrl.save(c, m)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200 for successful save, got %d: %s", w.Code, w.Body.String())
	}

	// Verify pipeline was updated
	var pipe model.TPipeline
	has, err := comm.Db.Where("id = ?", "save-pipe-id").Get(&pipe)
	if err != nil || !has {
		t.Fatalf("query pipeline after save: %v, has=%v", err, has)
	}
	if pipe.Name != "updated-name" {
		t.Errorf("expected name 'updated-name', got '%s'", pipe.Name)
	}
}

// --- Pipeline.pipelineVersions tests ---

func TestPipelineVersions_EmptyPipelineId(t *testing.T) {
	setupPipelineTestDB(t)
	ctrl := PipelineController{}
	m := &hbtp.Map{}
	m.Set("pipelineId", "")
	m.Set("page", int64(1))
	c, w := makePipelineTestContext(t, m)
	ctrl.pipelineVersions(c, m)

	// Empty pipelineId triggers the admin/user branch
	if w.Code != http.StatusOK {
		t.Errorf("expected 200 for empty pipelineId (admin fallback), got %d: %s", w.Code, w.Body.String())
	}
}

func TestPipelineVersions_NonexistentPipeline(t *testing.T) {
	setupPipelineTestDB(t)
	ctrl := PipelineController{}
	m := &hbtp.Map{}
	m.Set("pipelineId", "nonexistent-id")
	m.Set("page", int64(1))
	c, w := makePipelineTestContext(t, m)
	ctrl.pipelineVersions(c, m)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404 for nonexistent pipeline, got %d: %s", w.Code, w.Body.String())
	}
}

func TestPipelineVersions_Success(t *testing.T) {
	setupPipelineTestDB(t)

	// Insert a test pipeline and versions
	_, err := comm.Db.Exec(`INSERT INTO t_pipeline (id, uid, name, display_name, deleted) 
		VALUES ('versions-pipe-id', 'test-user', 'versions-test', 'Versions Test', 0)`)
	if err != nil {
		t.Fatalf("insert pipeline: %v", err)
	}
	for i := 1; i <= 3; i++ {
		_, err = comm.Db.Exec(`INSERT INTO t_pipeline_version (id, pipeline_id, number, deleted) 
			VALUES (?, 'versions-pipe-id', ?, 0)`, "ver-"+string(rune('a'+i)), i)
		if err != nil {
			t.Fatalf("insert pipeline_version: %v", err)
		}
	}

	ctrl := PipelineController{}
	m := &hbtp.Map{}
	m.Set("pipelineId", "versions-pipe-id")
	m.Set("page", int64(1))
	c, w := makePipelineTestContext(t, m)
	ctrl.pipelineVersions(c, m)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200 for valid request, got %d: %s", w.Code, w.Body.String())
	}
}

// --- Pipeline.varSave tests ---

func TestPipelineVarSave_EmptyPipelineId(t *testing.T) {
	setupPipelineTestDB(t)
	ctrl := PipelineController{}
	pv := &bean.PipelineVar{
		Name:  "VAR1",
		Value: "value1",
	}
	c, w := makePipelineTestContext(t, pv)
	ctrl.varSave(c, pv)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for empty pipelineId, got %d: %s", w.Code, w.Body.String())
	}
}

func TestPipelineVarSave_NonexistentPipeline(t *testing.T) {
	setupPipelineTestDB(t)
	ctrl := PipelineController{}
	pv := &bean.PipelineVar{
		PipelineId: "nonexistent-id",
		Name:       "VAR1",
		Value:      "value1",
	}
	c, w := makePipelineTestContext(t, pv)
	ctrl.varSave(c, pv)

	// Should return permission error or not found
	if w.Code != http.StatusMethodNotAllowed && w.Code != http.StatusNotFound {
		t.Errorf("expected 405 or 404 for nonexistent pipeline, got %d: %s", w.Code, w.Body.String())
	}
}
