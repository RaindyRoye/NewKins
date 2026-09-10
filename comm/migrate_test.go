package comm

import (
	"errors"
	"testing"
)

func TestAsset_ValidNames(t *testing.T) {
	assets := []string{
		"mysql/000001_gokins.down.sql",
		"mysql/000001_gokins.up.sql",
		"mysql/000002_gokins.down.sql",
		"mysql/000002_gokins.up.sql",
		"postgres/000001_gokins.down.sql",
		"postgres/000001_gokins.up.sql",
		"sqlite/000001_gokins.down.sql",
		"sqlite/000001_gokins.up.sql",
	}

	for _, name := range assets {
		t.Run(name, func(t *testing.T) {
			data, err := Asset(name)
			if err != nil {
				t.Fatalf("Asset(%q) returned unexpected error: %v", name, err)
			}
			if len(data) == 0 {
				t.Fatalf("Asset(%q) returned empty data", name)
			}
		})
	}
}

func TestAsset_NotFound(t *testing.T) {
	_, err := Asset("nonexistent/file.sql")
	if err == nil {
		t.Fatal("Asset(\"nonexistent/file.sql\") expected error, got nil")
	}
}

func TestAsset_NotFound_WrapsSentinel(t *testing.T) {
	_, err := Asset("nonexistent/file.sql")
	if err == nil {
		t.Fatal("Asset(\"nonexistent/file.sql\") expected error, got nil")
	}
	if !errors.Is(err, ErrAssetNotFound) {
		t.Errorf("Asset not-found error should wrap ErrAssetNotFound, got: %v", err)
	}
	if err.Error() == "" {
		t.Error("error message should not be empty")
	}
}

func TestAsset_BackslashNormalization(t *testing.T) {
	// Test that backslashes are normalized to forward slashes
	data, err := Asset("mysql\\000001_gokins.down.sql")
	if err != nil {
		t.Fatalf("Asset with backslashes returned unexpected error: %v", err)
	}
	if len(data) == 0 {
		t.Fatal("Asset with backslashes returned empty data")
	}
}

func TestAssetNames(t *testing.T) {
	names := AssetNames()
	if len(names) != 8 {
		t.Fatalf("AssetNames() returned %d names, expected 8", len(names))
	}

	// Verify all expected names are present
	expected := map[string]bool{
		"mysql/000001_gokins.down.sql":    false,
		"mysql/000001_gokins.up.sql":      false,
		"mysql/000002_gokins.down.sql":    false,
		"mysql/000002_gokins.up.sql":      false,
		"postgres/000001_gokins.down.sql": false,
		"postgres/000001_gokins.up.sql":   false,
		"sqlite/000001_gokins.down.sql":   false,
		"sqlite/000001_gokins.up.sql":     false,
	}

	for _, name := range names {
		if _, ok := expected[name]; ok {
			expected[name] = true
		} else {
			t.Errorf("AssetNames() returned unexpected name: %s", name)
		}
	}

	for name, found := range expected {
		if !found {
			t.Errorf("AssetNames() missing expected name: %s", name)
		}
	}
}

func TestAssetDir_Root(t *testing.T) {
	dirs, err := AssetDir("")
	if err != nil {
		t.Fatalf("AssetDir(\"\") returned unexpected error: %v", err)
	}
	if len(dirs) != 3 {
		t.Fatalf("AssetDir(\"\") returned %d dirs, expected 3 (mysql, postgres, sqlite)", len(dirs))
	}
}

func TestAssetDir_SubDir(t *testing.T) {
	dirs, err := AssetDir("mysql")
	if err != nil {
		t.Fatalf("AssetDir(\"mysql\") returned unexpected error: %v", err)
	}
	if len(dirs) != 4 {
		t.Fatalf("AssetDir(\"mysql\") returned %d files, expected 4", len(dirs))
	}
}

func TestAssetDir_NotFound(t *testing.T) {
	_, err := AssetDir("nonexistent")
	if err == nil {
		t.Fatal("AssetDir(\"nonexistent\") expected error, got nil")
	}
}

func TestAssetDir_NotFound_WrapsSentinel(t *testing.T) {
	tests := []struct {
		name  string
		asset string
	}{
		{"nonexistent directory", "nonexistent"},
		{"nonexistent file (leaf node)", "mysql/000001_gokins.up.sql"},
		{"deep path nonexistent", "nonexistent/path"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := AssetDir(tt.asset)
			if err == nil {
				t.Fatalf("AssetDir(%q) expected error, got nil", tt.asset)
			}
			if !errors.Is(err, ErrAssetNotFound) {
				t.Errorf("AssetDir error should wrap ErrAssetNotFound, got: %v", err)
			}
		})
	}
}

func TestBindataRead_InvalidData(t *testing.T) {
	// Test that bindata_read returns a wrapped error for invalid gzip data
	_, err := bindata_read([]byte("not gzip data"), "test")
	if err == nil {
		t.Fatal("bindata_read with invalid data expected error, got nil")
	}
	// Verify error is wrapped (can be unwrapped)
	unwrapped := errors.Unwrap(err)
	if unwrapped == nil {
		t.Fatal("bindata_read error should be wrapped to allow unwrapping")
	}
}

func TestErrAssetNotFound_IsSentinel(t *testing.T) {
	// Verify ErrAssetNotFound is a proper sentinel error
	if ErrAssetNotFound == nil {
		t.Fatal("ErrAssetNotFound should not be nil")
	}
	if ErrAssetNotFound.Error() == "" {
		t.Error("ErrAssetNotFound message should not be empty")
	}
}

