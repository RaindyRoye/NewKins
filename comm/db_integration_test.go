package comm

import (
	"context"
	"errors"
	"testing"

	"github.com/gokins/gokins/bean"
	_ "github.com/mattn/go-sqlite3"
	"xorm.io/builder"
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

// --- findCountCtx and FindPageCtx tests ---

func TestFindCountCtx_Basic(t *testing.T) {
	_ = setupTestDB(t)
	ResetCtx()

	var items []testItem
	count, err := findCountCtx(Ctx, builder.NewCond(), &items)
	if err != nil {
		t.Fatalf("findCountCtx failed: %v", err)
	}
	if count != 25 {
		t.Errorf("count = %d, want 25", count)
	}
}

func TestFindCountCtx_WithCondition(t *testing.T) {
	_ = setupTestDB(t)
	ResetCtx()

	var items []testItem
	count, err := findCountCtx(Ctx, builder.Gt{"value": 10}, &items)
	if err != nil {
		t.Fatalf("findCountCtx failed: %v", err)
	}
	if count != 15 {
		t.Errorf("count = %d, want 15", count)
	}
}

func TestFindCountCtx_NilData(t *testing.T) {
	_, err := findCountCtx(Ctx, builder.NewCond(), nil)
	if err == nil {
		t.Fatal("expected error for nil data, got nil")
	}
	if !errors.Is(err, ErrFindCountInvalidData) {
		t.Errorf("error = %v, want ErrFindCountInvalidData", err)
	}
}

func TestFindCountCtx_NonSlicePointer(t *testing.T) {
	type item struct{ Name string }
	var v item
	_, err := findCountCtx(Ctx, builder.NewCond(), &v)
	if err == nil {
		t.Fatal("expected error for pointer to non-slice, got nil")
	}
	if !errors.Is(err, ErrFindCountNotSlice) {
		t.Errorf("error = %v, want ErrFindCountNotSlice", err)
	}
}

func TestFindCountCtx_CanceledContext(t *testing.T) {
	_ = setupTestDB(t)
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately

	var items []testItem
	_, err := findCountCtx(ctx, builder.NewCond(), &items)
	if err == nil {
		t.Fatal("expected error for canceled context, got nil")
	}
}

func TestFindPageCtx_BasicPagination(t *testing.T) {
	eng := setupTestDB(t)
	ResetCtx()

	ses := eng.NewSession()
	defer func() { _ = ses.Close() }()

	var items []testItem
	page, err := FindPageCtx(Ctx, ses.OrderBy("id"), &items, 1, 10)
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
	ResetCtx()

	ses := eng.NewSession()
	defer func() { _ = ses.Close() }()

	var items []testItem
	page, err := FindPageCtx(Ctx, ses.OrderBy("id"), &items, 2, 10)
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

func TestFindPageCtx_DefaultPageSize(t *testing.T) {
	eng := setupTestDB(t)
	ResetCtx()

	ses := eng.NewSession()
	defer func() { _ = ses.Close() }()

	var items []testItem
	page, err := FindPageCtx(Ctx, ses.OrderBy("id"), &items, 1)
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

func TestFindPageCtx_CustomSize(t *testing.T) {
	eng := setupTestDB(t)
	ResetCtx()

	ses := eng.NewSession()
	defer func() { _ = ses.Close() }()

	var items []testItem
	page, err := FindPageCtx(Ctx, ses.OrderBy("id"), &items, 1, 5)
	if err != nil {
		t.Fatalf("FindPageCtx failed: %v", err)
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

func TestFindPageCtx_EmptyResult(t *testing.T) {
	eng := setupTestDB(t)
	ResetCtx()

	ses := eng.NewSession()
	defer func() { _ = ses.Close() }()

	var items []testItem
	page, err := FindPageCtx(Ctx, ses.Where("value > ?", 100).OrderBy("id"), &items, 1, 10)
	if err != nil {
		t.Fatalf("FindPageCtx failed: %v", err)
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

func TestFindPageCtx_CanceledContext(t *testing.T) {
	eng := setupTestDB(t)
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately

	ses := eng.NewSession()
	defer func() { _ = ses.Close() }()

	var items []testItem
	_, err := FindPageCtx(ctx, ses.OrderBy("id"), &items, 1, 10)
	if err == nil {
		t.Fatal("expected error for canceled context, got nil")
	}
}

func TestFindPagesCtx_WithRealDB(t *testing.T) {
	_ = setupTestDB(t)
	ResetCtx()

	gen := &bean.PageGen{
		SQL:      "SELECT {{select}} FROM test_item\nORDER BY id",
		FindCols: "*",
	}

	var items []testItem
	page, err := FindPagesCtx(Ctx, gen, &items, 1, 10)
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

func TestFindPagesCtx_WithCountCols(t *testing.T) {
	_ = setupTestDB(t)
	ResetCtx()

	gen := &bean.PageGen{
		SQL:       "SELECT {{select}} FROM test_item\nORDER BY id",
		FindCols:  "*",
		CountCols: "DISTINCT name",
	}

	var items []testItem
	page, err := FindPagesCtx(Ctx, gen, &items, 1, 10)
	if err != nil {
		t.Fatalf("FindPagesCtx failed: %v", err)
	}

	if page.Total != 1 {
		t.Errorf("page.Total = %d, want 1 (all items have same name)", page.Total)
	}
	if len(items) != 10 {
		t.Errorf("len(items) = %d, want 10", len(items))
	}
}

func TestFindPagesCtx_WithStartOffset(t *testing.T) {
	_ = setupTestDB(t)
	ResetCtx()

	gen := &bean.PageGen{
		SQL:      "SELECT {{select}} FROM test_item\nORDER BY id",
		FindCols: "*",
	}

	var items []testItem
	page, err := FindPagesCtx(Ctx, gen, &items, 2, 10)
	if err != nil {
		t.Fatalf("FindPagesCtx failed: %v", err)
	}

	if page.Page != 2 {
		t.Errorf("page.Page = %d, want 2", page.Page)
	}
	if len(items) != 10 {
		t.Errorf("len(items) = %d, want 10", len(items))
	}
}

func TestFindPagesCtx_WithLimitPlaceholder(t *testing.T) {
	_ = setupTestDB(t)
	ResetCtx()

	gen := &bean.PageGen{
		SQL:      "SELECT {{select}} FROM test_item\nORDER BY id\n{{limit}}",
		FindCols: "*",
	}

	var items []testItem
	_, err := FindPagesCtx(Ctx, gen, &items, 1, 5)
	if err != nil {
		t.Fatalf("FindPagesCtx failed: %v", err)
	}

	if len(items) != 5 {
		t.Errorf("len(items) = %d, want 5", len(items))
	}
}

func TestFindPagesCtx_MissingOrderBy(t *testing.T) {
	gen := &bean.PageGen{
		SQL:      "SELECT {{select}} FROM test_item WHERE value > 5",
		FindCols: "*",
	}

	var items []testItem
	_, err := FindPagesCtx(Ctx, gen, &items, 1)
	if err == nil {
		t.Fatal("expected error for SQL without ORDER BY clause, got nil")
	}
	if !errors.Is(err, ErrFindPagesMissingOrderBy) {
		t.Errorf("error = %v, want ErrFindPagesMissingOrderBy", err)
	}
}
