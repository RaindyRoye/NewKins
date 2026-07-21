package service

import (
	"testing"

	"github.com/gokins/core/runtime"
	"github.com/gokins/gokins/comm"
)

func TestReplace(t *testing.T) {
	vars := map[string]*runtime.Variables{
		"USER":  {Value: "alice"},
		"BRANCH": {Value: "main"},
		"SECRET": {Value: "s3cret", Secret: true},
	}

	tests := []struct {
		name     string
		input    string
		mustShow bool
		wantStr  string
		wantSec  bool
	}{
		{"empty", "", false, "", false},
		{"no vars", "hello world", false, "hello world", false},
		{"single var", "user: ${{USER}}", false, "user: alice", false},
		{"multiple vars", "${{USER}} on ${{BRANCH}}", false, "alice on main", false},
		{"secret masked", "token: ${{SECRET}}", false, "token: " + comm.MaskedValue, true},
		{"secret shown", "token: ${{SECRET}}", true, "token: s3cret", true},
		{"undefined var", "val: ${{UNDEFINED}}", false, "val: ", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotStr, gotSec := replace(tt.input, vars, tt.mustShow)
			if gotStr != tt.wantStr {
				t.Errorf("replace(%q) = %q, want %q", tt.input, gotStr, tt.wantStr)
			}
			if gotSec != tt.wantSec {
				t.Errorf("replace(%q) secret = %v, want %v", tt.input, gotSec, tt.wantSec)
			}
		})
	}
}
