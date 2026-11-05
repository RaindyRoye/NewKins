package route

import (
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/gokins/core/utils"
	"github.com/gokins/gokins/bean"
	"github.com/gokins/gokins/comm"
	"github.com/gokins/gokins/model"
	_ "github.com/mattn/go-sqlite3"
	hbtp "github.com/mgr9525/HyperByte-Transfer-Protocol"
)

func TestPipelineController_delete_EmptyId(t *testing.T) {
	setupPipelineNewTestDB(t)
	adminUser := &model.TUser{Id: "admin", Name: "admin", Active: 1}

	m := &hbtp.Map{}
	m.Set("id", "")

	c, w := makePipelineGinContext(t, m, adminUser)
	ctrl := PipelineController{}
	ctrl.delete(c, m)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status code = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestPipelineController_delete_Success(t *testing.T) {
	db := setupPipelineNewTestDB(t)
	adminUser := &model.TUser{Id: "admin", Name: "admin", Active: 1}

	// Create a pipeline
	pipeline := &model.TPipeline{
		Id:          utils.NewXid(),
		Uid:         "admin",
		Name:        "test-pipeline",
		DisplayName: "Test Pipeline",
	}
	_, err := db.InsertOne(pipeline)
	if err != nil {
		t.Fatalf("insert pipeline: %v", err)
	}

	m := &hbtp.Map{}
	m.Set("id", pipeline.Id)

	c, w := makePipelineGinContext(t, m, adminUser)
	ctrl := PipelineController{}
	ctrl.delete(c, m)

	if w.Code != http.StatusOK {
		t.Errorf("status code = %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	// Verify pipeline is marked as deleted
	var updated model.TPipeline
	ok, err := comm.Db.Where("id=?", pipeline.Id).Get(&updated)
	if err != nil {
		t.Fatalf("query pipeline: %v", err)
	}
	if !ok {
		t.Fatal("pipeline not found")
	}
	if updated.Deleted != 1 {
		t.Errorf("pipeline deleted = %d, want 1", updated.Deleted)
	}
}

func TestPipelineController_info_Success(t *testing.T) {
	db := setupPipelineNewTestDB(t)
	adminUser := &model.TUser{Id: "admin", Name: "admin", Active: 1}

	// Create a pipeline with conf
	pipeline := &model.TPipeline{
		Id:          utils.NewXid(),
		Uid:         "admin",
		Name:        "test-pipeline",
		DisplayName: "Test Pipeline",
	}
	_, err := db.InsertOne(pipeline)
	if err != nil {
		t.Fatalf("insert pipeline: %v", err)
	}

	conf := &model.TPipelineConf{
		PipelineId:  pipeline.Id,
		YmlContent:  "stages: []",
		Url:         "https://github.com/test/repo",
		Username:    "testuser",
		AccessToken: "token123",
	}
	_, err = db.InsertOne(conf)
	if err != nil {
		t.Fatalf("insert pipeline conf: %v", err)
	}

	m := &hbtp.Map{}
	m.Set("id", pipeline.Id)

	c, w := makePipelineGinContext(t, m, adminUser)
	ctrl := PipelineController{}
	ctrl.info(c, m)

	if w.Code != http.StatusOK {
		t.Errorf("status code = %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if resp["pipe"] == nil {
		t.Error("expected 'pipe' in response")
	}
	if resp["perm"] == nil {
		t.Error("expected 'perm' in response")
	}
}

func TestPipelineController_varDel_InvalidAid(t *testing.T) {
	setupPipelineNewTestDB(t)
	adminUser := &model.TUser{Id: "admin", Name: "admin", Active: 1}

	m := &hbtp.Map{}
	m.Set("aid", int64(0))

	c, w := makePipelineGinContext(t, m, adminUser)
	ctrl := PipelineController{}
	ctrl.varDel(c, m)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status code = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestPipelineController_varDel_Success(t *testing.T) {
	db := setupPipelineNewTestDB(t)
	adminUser := &model.TUser{Id: "admin", Name: "admin", Active: 1}

	// Create a pipeline
	pipeline := &model.TPipeline{
		Id:   utils.NewXid(),
		Uid:  "admin",
		Name: "test-pipeline",
	}
	_, err := db.InsertOne(pipeline)
	if err != nil {
		t.Fatalf("insert pipeline: %v", err)
	}

	// Create a variable
	v := &model.TPipelineVar{
		Uid:        "admin",
		PipelineId: pipeline.Id,
		Name:       "TEST_VAR",
		Value:      "test_value",
		Public:     1,
	}
	_, err = db.InsertOne(v)
	if err != nil {
		t.Fatalf("insert variable: %v", err)
	}

	m := &hbtp.Map{}
	m.Set("aid", v.Aid)

	c, w := makePipelineGinContext(t, m, adminUser)
	ctrl := PipelineController{}
	ctrl.varDel(c, m)

	if w.Code != http.StatusOK {
		t.Errorf("status code = %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	// Verify variable was deleted
	var count int64
	count, err = comm.Db.Where("aid=?", v.Aid).Count(&model.TPipelineVar{})
	if err != nil {
		t.Fatalf("count variables: %v", err)
	}
	if count != 0 {
		t.Errorf("expected 0 variables after delete, got %d", count)
	}
}

func TestPipelineController_searchSha_EmptyId(t *testing.T) {
	setupPipelineNewTestDB(t)
	adminUser := &model.TUser{Id: "admin", Name: "admin", Active: 1}

	m := &hbtp.Map{}
	m.Set("id", "")

	c, w := makePipelineGinContext(t, m, adminUser)
	ctrl := PipelineController{}
	ctrl.searchSha(c, m)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status code = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestPipelineController_searchSha_WithShas(t *testing.T) {
	db := setupPipelineNewTestDB(t)
	adminUser := &model.TUser{Id: "admin", Name: "admin", Active: 1}

	// Create a pipeline
	pipeline := &model.TPipeline{
		Id:   utils.NewXid(),
		Uid:  "admin",
		Name: "test-pipeline",
	}
	_, err := db.InsertOne(pipeline)
	if err != nil {
		t.Fatalf("insert pipeline: %v", err)
	}

	// Create pipeline versions with different SHAs
	shas := []string{"abc123", "def456", "abc789"}
	for i, sha := range shas {
		pv := &model.TPipelineVersion{
			Id:                  utils.NewXid(),
			Uid:                 "admin",
			Number:              int64(i + 1),
			Sha:                 sha,
			PipelineName:        "test-pipeline",
			PipelineDisplayName: "Test Pipeline",
			PipelineId:          pipeline.Id,
			Created:             time.Now(),
		}
		_, err := db.InsertOne(pv)
		if err != nil {
			t.Fatalf("insert pipeline version: %v", err)
		}
	}

	m := &hbtp.Map{}
	m.Set("id", pipeline.Id)
	m.Set("q", "abc")

	c, w := makePipelineGinContext(t, m, adminUser)
	ctrl := PipelineController{}
	ctrl.searchSha(c, m)

	if w.Code != http.StatusOK {
		t.Errorf("status code = %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	var resp []map[string]string
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}

	// Should find 2 SHAs matching "abc"
	if len(resp) != 2 {
		t.Errorf("expected 2 matching SHAs, got %d", len(resp))
	}
}

func TestPipelineController_copy_EmptyPipelineId(t *testing.T) {
	setupPipelineNewTestDB(t)
	adminUser := &model.TUser{Id: "admin", Name: "admin", Active: 1}

	m := &hbtp.Map{}
	m.Set("pipelineId", "")

	c, w := makePipelineGinContext(t, m, adminUser)
	ctrl := PipelineController{}
	ctrl.copy(c, m)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status code = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestPipelineController_copy_Success(t *testing.T) {
	db := setupPipelineNewTestDB(t)
	adminUser := &model.TUser{Id: "admin", Name: "admin", Active: 1}

	// Create a pipeline with conf
	pipeline := &model.TPipeline{
		Id:          utils.NewXid(),
		Uid:         "admin",
		Name:        "original-pipeline",
		DisplayName: "Original Pipeline",
	}
	_, err := db.InsertOne(pipeline)
	if err != nil {
		t.Fatalf("insert pipeline: %v", err)
	}

	conf := &model.TPipelineConf{
		PipelineId:  pipeline.Id,
		YmlContent:  "stages: []",
		Url:         "https://github.com/test/repo",
		Username:    "testuser",
		AccessToken: "token123",
	}
	_, err = db.InsertOne(conf)
	if err != nil {
		t.Fatalf("insert pipeline conf: %v", err)
	}

	m := &hbtp.Map{}
	m.Set("pipelineId", pipeline.Id)

	c, w := makePipelineGinContext(t, m, adminUser)
	ctrl := PipelineController{}
	ctrl.copy(c, m)

	if w.Code != http.StatusOK {
		t.Errorf("status code = %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}

	newId, ok := resp["id"].(string)
	if !ok || newId == "" {
		t.Fatal("expected new pipeline id in response")
	}
	if newId == pipeline.Id {
		t.Error("new pipeline should have different id")
	}

	// Verify new pipeline was created
	var newPipeline model.TPipeline
	ok2, err := comm.Db.Where("id=?", newId).Get(&newPipeline)
	if err != nil {
		t.Fatalf("query new pipeline: %v", err)
	}
	if !ok2 {
		t.Fatal("new pipeline not found")
	}
	if newPipeline.Name != "original-pipeline_copy" {
		t.Errorf("new pipeline name = %q, want %q", newPipeline.Name, "original-pipeline_copy")
	}

	// Verify new pipeline conf was created
	var newConf model.TPipelineConf
	ok3, err := comm.Db.Where("pipeline_id=?", newId).Get(&newConf)
	if err != nil {
		t.Fatalf("query new pipeline conf: %v", err)
	}
	if !ok3 {
		t.Fatal("new pipeline conf not found")
	}
	if newConf.Url != "https://github.com/test/repo" {
		t.Errorf("new conf url = %q, want %q", newConf.Url, "https://github.com/test/repo")
	}
}

func TestPipelineController_pipelineVersion_EmptyId(t *testing.T) {
	setupPipelineNewTestDB(t)
	adminUser := &model.TUser{Id: "admin", Name: "admin", Active: 1}

	m := &hbtp.Map{}
	m.Set("id", "")

	c, w := makePipelineGinContext(t, m, adminUser)
	ctrl := PipelineController{}
	ctrl.pipelineVersion(c, m)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status code = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestPipelineController_pipelineVersion_NotFound(t *testing.T) {
	setupPipelineNewTestDB(t)
	adminUser := &model.TUser{Id: "admin", Name: "admin", Active: 1}

	m := &hbtp.Map{}
	m.Set("id", "nonexistent")

	c, w := makePipelineGinContext(t, m, adminUser)
	ctrl := PipelineController{}
	ctrl.pipelineVersion(c, m)

	if w.Code != http.StatusNotFound {
		t.Errorf("status code = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestPipelineController_rebuild_EmptyPipelineVersionId(t *testing.T) {
	setupPipelineNewTestDB(t)
	adminUser := &model.TUser{Id: "admin", Name: "admin", Active: 1}

	m := &hbtp.Map{}
	m.Set("pipelineVersionId", "")

	c, w := makePipelineGinContext(t, m, adminUser)
	ctrl := PipelineController{}
	ctrl.rebuild(c, m)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status code = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestPipelineController_rebuild_NotFound(t *testing.T) {
	setupPipelineNewTestDB(t)
	adminUser := &model.TUser{Id: "admin", Name: "admin", Active: 1}

	m := &hbtp.Map{}
	m.Set("pipelineVersionId", "nonexistent")

	c, w := makePipelineGinContext(t, m, adminUser)
	ctrl := PipelineController{}
	ctrl.rebuild(c, m)

	if w.Code != http.StatusNotFound {
		t.Errorf("status code = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestPipelineController_orgPipelines_EmptyOrgId(t *testing.T) {
	setupPipelineNewTestDB(t)
	adminUser := &model.TUser{Id: "admin", Name: "admin", Active: 1}

	m := &hbtp.Map{}
	m.Set("orgId", "")

	c, w := makePipelineGinContext(t, m, adminUser)
	ctrl := PipelineController{}
	ctrl.orgPipelines(c, m)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status code = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestPipelineController_getPipelines_AsAdmin(t *testing.T) {
	db := setupPipelineNewTestDB(t)
	adminUser := &model.TUser{Id: "admin", Name: "admin", Active: 1}

	// Create some pipelines
	for i := 0; i < 3; i++ {
		pipeline := &model.TPipeline{
			Id:          utils.NewXid(),
			Uid:         "admin",
			Name:        "pipeline-" + string(rune('0'+i)),
			DisplayName: "Pipeline " + string(rune('0'+i)),
		}
		_, err := db.InsertOne(pipeline)
		if err != nil {
			t.Fatalf("insert pipeline: %v", err)
		}
	}

	m := &hbtp.Map{}
	m.Set("q", "")
	m.Set("page", int64(1))

	c, w := makePipelineGinContext(t, m, adminUser)
	ctrl := PipelineController{}
	ctrl.getPipelines(c, m)

	if w.Code != http.StatusOK {
		t.Errorf("status code = %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if resp["data"] == nil {
		t.Error("expected 'data' in response")
	}
}

func TestPipelineController_getPipelines_WithSearch(t *testing.T) {
	db := setupPipelineNewTestDB(t)
	adminUser := &model.TUser{Id: "admin", Name: "admin", Active: 1}

	// Create pipelines with different names
	names := []string{"alpha-pipeline", "beta-pipeline", "gamma-pipeline"}
	for _, name := range names {
		pipeline := &model.TPipeline{
			Id:          utils.NewXid(),
			Uid:         "admin",
			Name:        name,
			DisplayName: name,
		}
		_, err := db.InsertOne(pipeline)
		if err != nil {
			t.Fatalf("insert pipeline: %v", err)
		}
	}

	m := &hbtp.Map{}
	m.Set("q", "alpha")
	m.Set("page", int64(1))

	c, w := makePipelineGinContext(t, m, adminUser)
	ctrl := PipelineController{}
	ctrl.getPipelines(c, m)

	if w.Code != http.StatusOK {
		t.Errorf("status code = %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}
}

func TestPipelineController_vars_WithSearch(t *testing.T) {
	db := setupPipelineNewTestDB(t)
	adminUser := &model.TUser{Id: "admin", Name: "admin", Active: 1}

	// Create a pipeline
	pipeline := &model.TPipeline{
		Id:   utils.NewXid(),
		Uid:  "admin",
		Name: "test-pipeline",
	}
	_, err := db.InsertOne(pipeline)
	if err != nil {
		t.Fatalf("insert pipeline: %v", err)
	}

	// Create variables with different names
	varNames := []string{"DATABASE_URL", "API_KEY", "SECRET_TOKEN"}
	for _, name := range varNames {
		v := &model.TPipelineVar{
			Uid:        "admin",
			PipelineId: pipeline.Id,
			Name:       name,
			Value:      "value-" + name,
			Public:     0,
		}
		_, err := db.InsertOne(v)
		if err != nil {
			t.Fatalf("insert variable: %v", err)
		}
	}

	m := &hbtp.Map{}
	m.Set("pipelineId", pipeline.Id)
	m.Set("q", "API")
	m.Set("page", int64(1))

	c, w := makePipelineGinContext(t, m, adminUser)
	ctrl := PipelineController{}
	ctrl.vars(c, m)

	if w.Code != http.StatusOK {
		t.Errorf("status code = %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}
}

func TestPipelineController_varSave_UpdateExisting(t *testing.T) {
	db := setupPipelineNewTestDB(t)
	adminUser := &model.TUser{Id: "admin", Name: "admin", Active: 1}

	// Create a pipeline
	pipeline := &model.TPipeline{
		Id:   utils.NewXid(),
		Uid:  "admin",
		Name: "test-pipeline",
	}
	_, err := db.InsertOne(pipeline)
	if err != nil {
		t.Fatalf("insert pipeline: %v", err)
	}

	// Create an existing variable
	existingVar := &model.TPipelineVar{
		Uid:        "admin",
		PipelineId: pipeline.Id,
		Name:       "EXISTING_VAR",
		Value:      "old_value",
		Remarks:    "old remark",
		Public:     0,
	}
	_, err = db.InsertOne(existingVar)
	if err != nil {
		t.Fatalf("insert existing variable: %v", err)
	}

	// Update with aid specified
	pv := &bean.PipelineVar{
		Aid:        existingVar.Aid,
		Name:       "EXISTING_VAR",
		Value:      "new_value",
		Remarks:    "new remark",
		PipelineId: pipeline.Id,
		Public:     true,
	}

	c, w := makePipelineGinContext(t, pv, adminUser)
	ctrl := PipelineController{}
	ctrl.varSave(c, pv)

	if w.Code != http.StatusOK {
		t.Errorf("status code = %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	// Verify variable was updated
	var updated model.TPipelineVar
	ok, err := comm.Db.Where("aid=?", existingVar.Aid).Get(&updated)
	if err != nil {
		t.Fatalf("query updated variable: %v", err)
	}
	if !ok {
		t.Fatal("variable not found")
	}
	if updated.Value != "new_value" {
		t.Errorf("variable value = %q, want %q", updated.Value, "new_value")
	}
	if updated.Remarks != "new remark" {
		t.Errorf("variable remarks = %q, want %q", updated.Remarks, "new remark")
	}
	if updated.Public != 1 {
		t.Errorf("variable public = %d, want 1", updated.Public)
	}
}

func TestPipelineController_run_EmptyPipelineId(t *testing.T) {
	setupPipelineNewTestDB(t)
	adminUser := &model.TUser{Id: "admin", Name: "admin", Active: 1}

	m := &hbtp.Map{}
	m.Set("pipelineId", "")

	c, w := makePipelineGinContext(t, m, adminUser)
	ctrl := PipelineController{}
	ctrl.run(c, m)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status code = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestPipelineController_run_PipelineNotFound(t *testing.T) {
	setupPipelineNewTestDB(t)
	adminUser := &model.TUser{Id: "admin", Name: "admin", Active: 1}

	m := &hbtp.Map{}
	m.Set("pipelineId", "nonexistent")
	m.Set("sha", "abc123")

	c, w := makePipelineGinContext(t, m, adminUser)
	ctrl := PipelineController{}
	ctrl.run(c, m)

	if w.Code != http.StatusNotFound {
		t.Errorf("status code = %d, want %d", w.Code, http.StatusNotFound)
	}
}
