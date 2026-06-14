package server

import (
	"fmt"
	"strings"

	"github.com/gokins/gokins/comm"
	"github.com/sirupsen/logrus"
)

// ensureIndexes creates database indexes for columns that are frequently
// queried but lack indexes in the original migration SQL. This function is
// idempotent — it silently skips indexes that already exist.
//
// The indexes added here target the most common WHERE/JOIN patterns observed
// in the service and engine packages:
//   - Foreign key lookups (build_id, stage_id, pipeline_id, etc.)
//   - Login/auth lookups (user name)
//   - Soft-delete filtering (deleted flag)
//   - Timer trigger scanning (enabled + types)
func ensureIndexes() {
	if comm.Db == nil {
		logrus.Debug("ensureIndexes: database not initialized, skipping")
		return
	}

	type indexDef struct {
		table   string
		name    string
		columns string
	}

	indexes := []indexDef{
		// t_build: queried by pipeline_id and pipeline_version_id
		{"t_build", "idx_build_pipeline_id", "pipeline_id"},
		{"t_build", "idx_build_pipeline_version_id", "pipeline_version_id"},

		// t_stage: queried by build_id
		{"t_stage", "idx_stage_build_id", "build_id"},

		// t_step: queried by build_id and stage_id
		{"t_step", "idx_step_build_id", "build_id"},
		{"t_step", "idx_step_stage_id", "stage_id"},

		// t_cmd_line: queried by build_id and step_id for log streaming
		{"t_cmd_line", "idx_cmdline_build_id", "build_id"},
		{"t_cmd_line", "idx_cmdline_step_id", "step_id"},

		// t_pipeline: queried by uid and deleted flag
		{"t_pipeline", "idx_pipeline_uid", "uid"},
		{"t_pipeline", "idx_pipeline_deleted", "deleted"},

		// t_pipeline_conf: queried by pipeline_id
		{"t_pipeline_conf", "idx_pipeconf_pipeline_id", "pipeline_id"},

		// t_pipeline_version: queried by pipeline_id
		{"t_pipeline_version", "idx_pipever_pipeline_id", "pipeline_id"},

		// t_pipeline_var: queried by pipeline_id
		{"t_pipeline_var", "idx_pipevar_pipeline_id", "pipeline_id"},

		// t_user: queried by name for login
		{"t_user", "idx_user_name", "name"},

		// t_param: queried by name for key-value lookups
		{"t_param", "idx_param_name", "name"},

		// t_trigger: scanned by enabled + types for timer scheduling
		{"t_trigger", "idx_trigger_enabled_types", "enabled, types"},

		// t_artifactory: queried by identifier + org_id
		{"t_artifactory", "idx_artifactory_identifier", "identifier"},

		// t_org_pipe: used in subqueries filtered by pipe_id
		{"t_org_pipe", "idx_orgpipe_pipe_id", "pipe_id"},

		// t_artifact_version: queried by package_id
		{"t_artifact_version", "idx_artver_package_id", "package_id"},
	}

	for _, idx := range indexes {
		if err := createIndexIfNotExists(idx.table, idx.name, idx.columns); err != nil {
			// Index creation failures are non-fatal — the queries still work,
			// just slower. Log at debug level to avoid noise.
			logrus.Debugf("ensureIndexes: skip %s.%s: %v", idx.table, idx.name, err)
		}
	}
	logrus.Debugf("ensureIndexes: checked %d indexes", len(indexes))
}

// createIndexIfNotExists creates an index if it does not already exist.
// For PostgreSQL and SQLite, uses CREATE INDEX IF NOT EXISTS.
// For MySQL, attempts creation and silently ignores "Duplicate key name" errors.
func createIndexIfNotExists(table, indexName, columns string) error {
	var sql string
	if comm.IsMySQL {
		// MySQL doesn't support IF NOT EXISTS for CREATE INDEX.
		// We attempt creation and ignore duplicate-key errors.
		sql = "CREATE INDEX `" + indexName + "` ON `" + table + "` (" + columns + ")"
		_, err := comm.Db.Exec(sql)
		if err != nil {
			msg := err.Error()
			// MySQL error 1061 = Duplicate key name
			if strings.Contains(msg, "Duplicate key name") || strings.Contains(msg, "1061") {
				return nil
			}
			return fmt.Errorf("create mysql index %s on %s: %w", indexName, table, err)
		}
	} else {
		// SQLite and PostgreSQL support IF NOT EXISTS
		sql = "CREATE INDEX IF NOT EXISTS " + indexName + " ON " + table + " (" + columns + ")"
		_, err := comm.Db.Exec(sql)
		if err != nil {
			return fmt.Errorf("create index %s on %s: %w", indexName, table, err)
		}
	}
	return nil
}
