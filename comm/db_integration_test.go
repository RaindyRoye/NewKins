package comm

import (
	"context"
	"strings"
	"testing"

	"github.com/gokins/gokins/bean"
	_ "github.com/mattn/go-sqlite3"
	"xorm.io/xorm"
)

// setupTestDB creates an in-memory SQLite database with a test table.
func setupTestDB(t *testing.T) *xorm.Engine {
	t.Helper()
	eng, err := xorm.NewEngine("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("failed to create test database: %v", err)
	}

	// Set the global Db so findCount can use it
	oldDb := Db
	Db = eng
	t.Cleanup(func() {
		Db = oldDb
		_ = eng.Close()
	})

	// Create a simple test table (xorm maps "testItem" -> "test_item")
	_, err = eng.Exec(`
		CREATE TABLE test_item (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL,
			value INTEGER NOT NULL
		)
	`)
	if err != nil {
		t.Fatalf("failed to create test table: %v", err)
	}

	// Insert test data
	for i := 1; i <= 25; i++ {
		_, err = eng.Exec("INSERT INTO test_item (name, value) VALUES (?, ?)", "item", i)
		if err != nil {
			t.Fatalf("failed to insert test data: %v", err)
		}
	}

	return eng
}

type testItem struct {
	Id    int64 `xorm:"pk autoincr"`
	Name  string
	Value int
}

func TestFindPage_BasicPagination(t *testing.T) {
	eng := setupTestDB(t)
	ses := eng.NewSession()
	defer func() { _ = ses.Close() }()

	var items []testItem
	page, err := FindPage(ses.OrderBy("id"), &items, 1, 10)
	if err != nil {
		t.Fatalf("FindPage failed: %v", err)
	}

	if page.Page != 1 {
		t.Errorf("page.Page = %d, want 1", page.Page)
	}
	if page.Size != 10 {
		t.Errorf("page.Size = %d, want 10", page.Size)
	}
	if page.Total != 25 {
		t.Errorf("page.Total = %d, want 25", page.Total)
	}
	if page.Pages != 3 {
		t.Errorf("page.Pages = %d, want 3", page.Pages)
	}
	if len(items) != 10 {
		t.Errorf("len(items) = %d, want 10", len(items))
	}
}

func TestFindPage_SecondPage(t *testing.T) {
	eng := setupTestDB(t)
	ses := eng.NewSession()
	defer func() { _ = ses.Close() }()

	var items []testItem
	page, err := FindPage(ses.OrderBy("id"), &items, 2, 10)
	if err != nil {
		t.Fatalf("FindPage failed: %v", err)
	}

	if page.Page != 2 {
		t.Errorf("page.Page = %d, want 2", page.Page)
	}
	if len(items) != 10 {
		t.Errorf("len(items) = %d, want 10", len(items))
	}
}

func TestFindPage_LastPage(t *testing.T) {
	eng := setupTestDB(t)
	ses := eng.NewSession()
	defer func() { _ = ses.Close() }()

	var items []testItem
	page, err := FindPage(ses.OrderBy("id"), &items, 3, 10)
	if err != nil {
		t.Fatalf("FindPage failed: %v", err)
	}

	if page.Page != 3 {
		t.Errorf("page.Page = %d, want 3", page.Page)
	}
	if len(items) != 5 {
		t.Errorf("len(items) = %d, want 5 (last page)", len(items))
	}
}

func TestFindPage_DefaultPageSize(t *testing.T) {
	eng := setupTestDB(t)
	ses := eng.NewSession()
	defer func() { _ = ses.Close() }()

	var items []testItem
	page, err := FindPage(ses.OrderBy("id"), &items, 1)
	if err != nil {
		t.Fatalf("FindPage failed: %v", err)
	}

	if page.Size != 10 {
		t.Errorf("page.Size = %d, want 10 (default)", page.Size)
	}
	if len(items) != 10 {
		t.Errorf("len(items) = %d, want 10", len(items))
	}
}

func TestFindPage_ZeroPageNumber(t *testing.T) {
	eng := setupTestDB(t)
	ses := eng.NewSession()
	defer func() { _ = ses.Close() }()

	var items []testItem
	page, err := FindPage(ses.OrderBy("id"), &items, 0, 10)
	if err != nil {
		t.Fatalf("FindPage failed: %v", err)
	}

	if page.Page != 1 {
		t.Errorf("page.Page = %d, want 1 (zero should default to 1)", page.Page)
	}
}

func TestFindPage_CustomSize(t *testing.T) {
	eng := setupTestDB(t)
	ses := eng.NewSession()
	defer func() { _ = ses.Close() }()

	var items []testItem
	page, err := FindPage(ses.OrderBy("id"), &items, 1, 5)
	if err != nil {
		t.Fatalf("FindPage failed: %v", err)
	}

	if page.Size != 5 {
		t.Errorf("page.Size = %d, want 5", page.Size)
	}
	if page.Pages != 5 {
		t.Errorf("page.Pages = %d, want 5", page.Pages)
	}
	if len(items) != 5 {
		t.Errorf("len(items) = %d, want 5", len(items))
	}
}

func TestFindPage_EmptyResult(t *testing.T) {
	eng := setupTestDB(t)
	ses := eng.NewSession()
	defer func() { _ = ses.Close() }()

	var items []testItem
	page, err := FindPage(ses.Where("value > ?", 100).OrderBy("id"), &items, 1, 10)
	if err != nil {
		t.Fatalf("FindPage failed: %v", err)
	}

	if page.Total != 0 {
		t.Errorf("page.Total = %d, want 0", page.Total)
	}
	if page.Pages != 0 {
		t.Errorf("page.Pages = %d, want 0", page.Pages)
	}
	if len(items) != 0 {
		t.Errorf("len(items) = %d, want 0", len(items))
	}
}

func TestFindPages_WithRealDB(t *testing.T) {
	_ = setupTestDB(t) // sets global Db via setupTestDB
	// Reset context in case a previous test canceled it
	ResetCtx()
	oldCtx := Ctx
	t.Cleanup(func() {
		// Restore original context
		Ctx = oldCtx
	})

	gen := &bean.PageGen{
		SQL:      "SELECT {{select}} FROM test_item\nORDER BY id",
		FindCols: "*",
	}

	var items []testItem
	page, err := FindPages(gen, &items, 1, 10)
	if err != nil {
		t.Fatalf("FindPages failed: %v", err)
	}

	if page.Total != 25 {
		t.Errorf("page.Total = %d, want 25", page.Total)
	}
	if len(items) != 10 {
		t.Errorf("len(items) = %d, want 10", len(items))
	}
}

// --- Context-aware pagination tests ---

func TestFindPageCtx_BasicPagination(t *testing.T) {
	eng := setupTestDB(t)
	ses := eng.NewSession()
	defer func() { _ = ses.Close() }()

	ctx := context.Background()
	var items []testItem
	page, err := FindPageCtx(ctx, ses.OrderBy("id"), &items, 1, 10)
	if err != nil {
		t.Fatalf("FindPageCtx failed: %v", err)
	}

	if page.Page != 1 {
		t.Errorf("page.Page = %d, want 1", page.Page)
	}
	if page.Size != 10 {
		t.Errorf("page.Size = %d, want 10", page.Size)
	}
	if page.Total != 25 {
		t.Errorf("page.Total = %d, want 25", page.Total)
	}
	if page.Pages != 3 {
		t.Errorf("page.Pages = %d, want 3", page.Pages)
	}
	if len(items) != 10 {
		t.Errorf("len(items) = %d, want 10", len(items))
	}
}

func TestFindPageCtx_SecondPage(t *testing.T) {
	eng := setupTestDB(t)
	ses := eng.NewSession()
	defer func() { _ = ses.Close() }()

	ctx := context.Background()
	var items []testItem
	page, err := FindPageCtx(ctx, ses.OrderBy("id"), &items, 2, 10)
	if err != nil {
		t.Fatalf("FindPageCtx failed: %v", err)
	}

	if page.Page != 2 {
		t.Errorf("page.Page = %d, want 2", page.Page)
	}
	if len(items) != 10 {
		t.Errorf("len(items) = %d, want 10", len(items))
	}
}

func TestFindPageCtx_LastPage(t *testing.T) {
	eng := setupTestDB(t)
	ses := eng.NewSession()
	defer func() { _ = ses.Close() }()

	ctx := context.Background()
	var items []testItem
	page, err := FindPageCtx(ctx, ses.OrderBy("id"), &items, 3, 10)
	if err != nil {
		t.Fatalf("FindPageCtx failed: %v", err)
	}

	if page.Page != 3 {
		t.Errorf("page.Page = %d, want 3", page.Page)
	}
	if len(items) != 5 {
		t.Errorf("len(items) = %d, want 5 (last page)", len(items))
	}
}

func TestFindPageCtx_CanceledContext(t *testing.T) {
	eng := setupTestDB(t)
	ses := eng.NewSession()
	defer func() { _ = ses.Close() }()

	// Create an already-canceled context
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	var items []testItem
	_, err := FindPageCtx(ctx, ses.OrderBy("id"), &items, 1, 10)
	if err == nil {
		t.Fatal("expected error with canceled context, got nil")
	}
}

func TestFindPageCtx_NilData(t *testing.T) {
	eng := setupTestDB(t)
	ses := eng.NewSession()
	defer func() { _ = ses.Close() }()

	ctx := context.Background()
	_, err := FindPageCtx(ctx, ses.OrderBy("id"), nil, 1, 10)
	if err == nil {
		t.Fatal("expected error for nil data, got nil")
	}
}

func TestFindPageCtx_NonSliceData(t *testing.T) {
	eng := setupTestDB(t)
	ses := eng.NewSession()
	defer func() { _ = ses.Close() }()

	ctx := context.Background()
	var v testItem
	_, err := FindPageCtx(ctx, ses.OrderBy("id"), &v, 1, 10)
	if err == nil {
		t.Fatal("expected error for non-slice data, got nil")
	}
	if !strings.Contains(err.Error(), "expected pointer to slice") {
		t.Errorf("error = %q, want containing \"expected pointer to slice\"", err.Error())
	}
}

func TestFindPageCtx_EmptyResult(t *testing.T) {
	eng := setupTestDB(t)
	ses := eng.NewSession()
	defer func() { _ = ses.Close() }()

	ctx := context.Background()
	var items []testItem
	page, err := FindPageCtx(ctx, ses.Where("value > ?", 100).OrderBy("id"), &items, 1, 10)
	if err != nil {
		t.Fatalf("FindPageCtx failed: %v", err)
	}

	if page.Total != 0 {
		t.Errorf("page.Total = %d, want 0", page.Total)
	}
	if len(items) != 0 {
		t.Errorf("len(items) = %d, want 0", len(items))
	}
}

func TestFindPageCtx_DefaultPageSize(t *testing.T) {
	eng := setupTestDB(t)
	ses := eng.NewSession()
	defer func() { _ = ses.Close() }()

	ctx := context.Background()
	var items []testItem
	page, err := FindPageCtx(ctx, ses.OrderBy("id"), &items, 1)
	if err != nil {
		t.Fatalf("FindPageCtx failed: %v", err)
	}

	if page.Size != 10 {
		t.Errorf("page.Size = %d, want 10 (default)", page.Size)
	}
	if len(items) != 10 {
		t.Errorf("len(items) = %d, want 10", len(items))
	}
}

func TestFindPagesCtx_WithRealDB(t *testing.T) {
	_ = setupTestDB(t)
	ResetCtx()
	oldCtx := Ctx
	t.Cleanup(func() {
		Ctx = oldCtx
	})

	gen := &bean.PageGen{
		SQL:      "SELECT {{select}} FROM test_item\nORDER BY id",
		FindCols: "*",
	}

	ctx := context.Background()
	var items []testItem
	page, err := FindPagesCtx(ctx, gen, &items, 1, 10)
	if err != nil {
		t.Fatalf("FindPagesCtx failed: %v", err)
	}

	if page.Total != 25 {
		t.Errorf("page.Total = %d, want 25", page.Total)
	}
	if len(items) != 10 {
		t.Errorf("len(items) = %d, want 10", len(items))
	}
}

func TestFindPagesCtx_SecondPage(t *testing.T) {
	_ = setupTestDB(t)
	ResetCtx()
	oldCtx := Ctx
	t.Cleanup(func() {
		Ctx = oldCtx
	})

	gen := &bean.PageGen{
		SQL:      "SELECT {{select}} FROM test_item\nORDER BY id",
		FindCols: "*",
	}

	ctx := context.Background()
	var items []testItem
	page, err := FindPagesCtx(ctx, gen, &items, 2, 10)
	if err != nil {
		t.Fatalf("FindPagesCtx failed: %v", err)
	}

	if page.Page != 2 {
		t.Errorf("page.Page = %d, want 2", page.Page)
	}
	if page.Total != 25 {
		t.Errorf("page.Total = %d, want 25", page.Total)
	}
	if len(items) != 10 {
		t.Errorf("len(items) = %d, want 10", len(items))
	}
}

func TestFindPagesCtx_CustomSize(t *testing.T) {
	_ = setupTestDB(t)
	ResetCtx()
	oldCtx := Ctx
	t.Cleanup(func() {
		Ctx = oldCtx
	})

	gen := &bean.PageGen{
		SQL:      "SELECT {{select}} FROM test_item\nORDER BY id",
		FindCols: "*",
	}

	ctx := context.Background()
	var items []testItem
	page, err := FindPagesCtx(ctx, gen, &items, 1, 5)
	if err != nil {
		t.Fatalf("FindPagesCtx failed: %v", err)
	}

	if page.Size != 5 {
		t.Errorf("page.Size = %d, want 5", page.Size)
	}
	if page.Pages != 5 {
		t.Errorf("page.Pages = %d, want 5", page.Pages)
	}
	if len(items) != 5 {
		t.Errorf("len(items) = %d, want 5", len(items))
	}
}

func TestFindPagesCtx_MissingOrderBy(t *testing.T) {
	_ = setupTestDB(t)
	ResetCtx()
	oldCtx := Ctx
	t.Cleanup(func() {
		Ctx = oldCtx
	})

	gen := &bean.PageGen{
		SQL:      "SELECT {{select}} FROM test_item WHERE 1=1",
		FindCols: "*",
	}

	ctx := context.Background()
	var items []testItem
	_, err := FindPagesCtx(ctx, gen, &items, 1, 10)
	if err == nil {
		t.Fatal("expected error for SQL without ORDER BY clause, got nil")
	}
	if !strings.Contains(err.Error(), "ORDER BY") {
		t.Errorf("error should mention ORDER BY, got: %v", err)
	}
}

func TestFindPagesCtx_CanceledContext(t *testing.T) {
	_ = setupTestDB(t)
	ResetCtx()
	oldCtx := Ctx
	t.Cleanup(func() {
		Ctx = oldCtx
	})

	gen := &bean.PageGen{
		SQL:      "SELECT {{select}} FROM test_item\nORDER BY id",
		FindCols: "*",
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	var items []testItem
	_, err := FindPagesCtx(ctx, gen, &items, 1, 10)
	if err == nil {
		t.Fatal("expected error with canceled context, got nil")
	}
}

func TestFindPagesCtx_CountCols(t *testing.T) {
	_ = setupTestDB(t)
	ResetCtx()
	oldCtx := Ctx
	t.Cleanup(func() {
		Ctx = oldCtx
	})

	gen := &bean.PageGen{
		SQL:       "SELECT {{select}} FROM test_item\nORDER BY id",
		FindCols:  "id, name",
		CountCols: "id",
	}

	ctx := context.Background()
	var items []testItem
	page, err := FindPagesCtx(ctx, gen, &items, 1, 10)
	if err != nil {
		t.Fatalf("FindPagesCtx failed: %v", err)
	}

	if page.Total != 25 {
		t.Errorf("page.Total = %d, want 25", page.Total)
	}
	if len(items) != 10 {
		t.Errorf("len(items) = %d, want 10", len(items))
	}
}

func TestFindPagesCtx_WithLimitPlaceholder(t *testing.T) {
	_ = setupTestDB(t)
	ResetCtx()
	oldCtx := Ctx
	t.Cleanup(func() {
		Ctx = oldCtx
	})

	gen := &bean.PageGen{
		SQL:      "SELECT {{select}} FROM test_item\nORDER BY id {{limit}}",
		FindCols: "*",
	}

	ctx := context.Background()
	var items []testItem
	page, err := FindPagesCtx(ctx, gen, &items, 1, 10)
	if err != nil {
		t.Fatalf("FindPagesCtx failed: %v", err)
	}

	if page.Total != 25 {
		t.Errorf("page.Total = %d, want 25", page.Total)
	}
	if len(items) != 10 {
		t.Errorf("len(items) = %d, want 10", len(items))
	}
}
