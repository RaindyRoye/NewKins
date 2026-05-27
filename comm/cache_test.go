package comm

import (
	"bytes"
	"testing"
	"time"

	hbtp "github.com/mgr9525/HyperByte-Transfer-Protocol"
)

func buildCacheEntry(expiry time.Time, data []byte) []byte {
	tms := []byte(expiry.Format(time.RFC3339Nano))
	buf := &bytes.Buffer{}
	buf.Write(hbtp.BigIntToByte(int64(len(tms)), 4))
	buf.Write(tms)
	buf.Write(data)
	return buf.Bytes()
}

func TestParseCacheData_ValidNotExpired(t *testing.T) {
	data := []byte("hello world")
	// Set expiry 1 hour in the future
	entry := buildCacheEntry(time.Now().Add(time.Hour), data)

	result := parseCacheData(entry)
	if result == nil {
		t.Fatal("expected non-nil result for non-expired entry")
	}
	if !bytes.Equal(result, data) {
		t.Errorf("got %q, want %q", result, data)
	}
}

func TestParseCacheData_Expired(t *testing.T) {
	data := []byte("hello world")
	// Set expiry 1 hour in the past
	entry := buildCacheEntry(time.Now().Add(-time.Hour), data)

	result := parseCacheData(entry)
	if result != nil {
		t.Errorf("expected nil result for expired entry, got %q", result)
	}
}

func TestParseCacheData_NilInput(t *testing.T) {
	result := parseCacheData(nil)
	if result != nil {
		t.Errorf("expected nil result for nil input, got %v", result)
	}
}

func TestParseCacheData_EmptyInput(t *testing.T) {
	result := parseCacheData([]byte{})
	if result != nil {
		t.Errorf("expected nil result for empty input, got %v", result)
	}
}

func TestParseCacheData_InvalidTimestamp(t *testing.T) {
	// Create entry with invalid timestamp format
	invalidTs := []byte("not-a-timestamp")
	buf := &bytes.Buffer{}
	buf.Write(hbtp.BigIntToByte(int64(len(invalidTs)), 4))
	buf.Write(invalidTs)
	buf.Write([]byte("data"))

	result := parseCacheData(buf.Bytes())
	if result != nil {
		t.Errorf("expected nil result for invalid timestamp, got %v", result)
	}
}

func TestParseCacheData_EmptyData(t *testing.T) {
	// Entry with valid timestamp but no data payload
	entry := buildCacheEntry(time.Now().Add(time.Hour), []byte{})

	result := parseCacheData(entry)
	if result == nil {
		t.Fatal("expected non-nil result for non-expired entry with empty data")
	}
	if len(result) != 0 {
		t.Errorf("expected empty data, got %q", result)
	}
}

func TestParseCacheData_ExpiringSoon(t *testing.T) {
	data := []byte("soon-to-expire")
	// Set expiry 1 second in the future
	entry := buildCacheEntry(time.Now().Add(time.Second), data)

	result := parseCacheData(entry)
	if result == nil {
		t.Fatal("expected non-nil result for entry that hasn't expired yet")
	}
	if !bytes.Equal(result, data) {
		t.Errorf("got %q, want %q", result, data)
	}
}

func TestParseCacheData_JustExpired(t *testing.T) {
	data := []byte("just-expired")
	// Set expiry 1 second in the past
	entry := buildCacheEntry(time.Now().Add(-time.Second), data)

	result := parseCacheData(entry)
	if result != nil {
		t.Errorf("expected nil for just-expired entry, got %q", result)
	}
}

func TestParseCacheData_BinaryData(t *testing.T) {
	// Test with binary data containing null bytes
	data := []byte{0x00, 0x01, 0xFF, 0xFE, 0x00, 0x42}
	entry := buildCacheEntry(time.Now().Add(time.Hour), data)

	result := parseCacheData(entry)
	if result == nil {
		t.Fatal("expected non-nil result for binary data")
	}
	if !bytes.Equal(result, data) {
		t.Errorf("binary data mismatch: got %v, want %v", result, data)
	}
}
