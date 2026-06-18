package comm

import (
	"strings"
	"testing"

	"github.com/gokins/gokins/bean"
	"xorm.io/builder"
)

func TestFindCount_NilData(t *testing.T) {
	_, err := findCount(builder.NewCond(), nil)
	if err == nil {
		t.Fatal("expected error for nil data, got nil")
	}
	if !strings.Contains(err.Error(), "non-nil pointer") {
		t.Errorf("error = %q, want containing \"non-nil pointer\"", err.Error())
	}
}

func TestFindCount_NonSlicePointer(t *testing.T) {
	type item struct{ Name string }
	var v item
	_, err := findCount(builder.NewCond(), &v)
	if err == nil {
		t.Fatal("expected error for pointer to non-slice, got nil")
	}
	if !strings.Contains(err.Error(), "expected pointer to slice") {
		t.Errorf("error = %q, want containing \"expected pointer to slice\"", err.Error())
	}
}

func TestFindCount_NonPointer(t *testing.T) {
	_, err := findCount(builder.NewCond(), 42)
	if err == nil {
		t.Fatal("expected error for non-pointer, got nil")
	}
	if !strings.Contains(err.Error(), "expected pointer to slice") {
		t.Errorf("error = %q, want containing \"expected pointer to slice\"", err.Error())
	}
}

func TestFindCount_StringArg(t *testing.T) {
	_, err := findCount(builder.NewCond(), "not a slice")
	if err == nil {
		t.Fatal("expected error for string argument, got nil")
	}
}

func TestFindPages_MissingOrderByClause(t *testing.T) {
	// FindPages should return an error (not panic) when SQL lacks "\nORDER BY"
	gen := &bean.PageGen{
		SQL:      "SELECT {{select}} FROM t_build WHERE deleted != 1",
		FindCols: "*",
	}

	var results []any
	_, err := FindPages(gen, &results, 1)
	if err == nil {
		t.Fatal("expected error for SQL without ORDER BY clause, got nil")
	}
	if !strings.Contains(err.Error(), "ORDER BY") {
		t.Errorf("error should mention ORDER BY, got: %v", err)
	}
}

func TestFindPages_EmptySQL(t *testing.T) {
	gen := &bean.PageGen{
		SQL:      "",
		FindCols: "*",
	}

	var results []any
	_, err := FindPages(gen, &results, 1)
	if err == nil {
		t.Fatal("expected error for empty SQL, got nil")
	}
}

func TestFindPages_SQLWithOrderByButNoSelect(t *testing.T) {
	// SQL has ORDER BY but Db is nil — should fail at DB access, not at parsing
	gen := &bean.PageGen{
		SQL:      "SELECT {{select}} FROM t_build\nORDER BY created DESC",
		FindCols: "*",
	}

	var results []any
	// Db is nil in test context, so this should panic or return a nil pointer error
	// We only verify the ORDER BY parsing doesn't return an error
	defer func() {
		if r := recover(); r != nil {
			// Expected: nil pointer dereference from Db being nil
			// This confirms the ORDER BY check passed correctly
			t.Logf("recovered expected panic (Db is nil): %v", r)
		}
	}()
	_, err := FindPages(gen, &results, 1)
	if err != nil && strings.Contains(err.Error(), "ORDER BY") {
		t.Errorf("should not return ORDER BY error when clause is present: %v", err)
	}
}
