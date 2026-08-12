package comm

import (
	"errors"
	"testing"
)

func TestAssetNotFoundSentinel(t *testing.T) {
	// Test that Asset returns ErrAssetNotFound for non-existent assets
	_, err := Asset("nonexistent/file.sql")
	if err == nil {
		t.Fatal("Asset() expected error for non-existent file, got nil")
	}
	if !errors.Is(err, ErrAssetNotFound) {
		t.Errorf("Asset() error = %v, want errors.Is(err, ErrAssetNotFound)", err)
	}
}

func TestAssetDirNotFoundSentinel(t *testing.T) {
	// Test that AssetDir returns ErrAssetNotFound for non-existent directories
	_, err := AssetDir("nonexistent-dir")
	if err == nil {
		t.Fatal("AssetDir() expected error for non-existent directory, got nil")
	}
	if !errors.Is(err, ErrAssetNotFound) {
		t.Errorf("AssetDir() error = %v, want errors.Is(err, ErrAssetNotFound)", err)
	}
}

func TestBindataReadDecompressionLimit(t *testing.T) {
	// This test verifies that bindata_read wraps errors with ErrDecompressionLimit
	// when decompressed size exceeds the limit. We can't easily trigger this
	// without a large compressed asset, but we can verify the sentinel exists.
	if ErrDecompressionLimit == nil {
		t.Fatal("ErrDecompressionLimit should not be nil")
	}
	if ErrDecompressionLimit.Error() != "decompressed size exceeds limit" {
		t.Errorf("ErrDecompressionLimit message = %q, want %q",
			ErrDecompressionLimit.Error(), "decompressed size exceeds limit")
	}
}

func TestInvalidDataTypeSentinel(t *testing.T) {
	// Verify the sentinel error exists and has correct message
	if ErrInvalidDataType == nil {
		t.Fatal("ErrInvalidDataType should not be nil")
	}
	if ErrInvalidDataType.Error() != "invalid data type" {
		t.Errorf("ErrInvalidDataType message = %q, want %q",
			ErrInvalidDataType.Error(), "invalid data type")
	}
}

func TestPageGenMissingOrderBySentinel(t *testing.T) {
	// Verify the sentinel error exists and has correct message
	if ErrPageGenMissingOrderBy == nil {
		t.Fatal("ErrPageGenMissingOrderBy should not be nil")
	}
	if ErrPageGenMissingOrderBy.Error() != "SQL missing ORDER BY clause" {
		t.Errorf("ErrPageGenMissingOrderBy message = %q, want %q",
			ErrPageGenMissingOrderBy.Error(), "SQL missing ORDER BY clause")
	}
}
