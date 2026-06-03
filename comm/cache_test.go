package comm

import (
	"bytes"
	"errors"
	"os"
	"testing"
	"time"

	hbtp "github.com/mgr9525/HyperByte-Transfer-Protocol"
	bolt "go.etcd.io/bbolt"
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

// --- Integration tests with real bbolt DB ---

// setupTestCache creates a temporary bbolt DB for testing and sets BCache.
// Returns a cleanup function that must be deferred.
func setupTestCache(t *testing.T) func() {
	t.Helper()
	tmpFile, err := os.CreateTemp("", "cache_test_*.db")
	if err != nil {
		t.Fatalf("failed to create temp db: %v", err)
	}
	_ = tmpFile.Close()

	db, err := bolt.Open(tmpFile.Name(), 0600, &bolt.Options{Timeout: 1 * time.Second})
	if err != nil {
		_ = os.Remove(tmpFile.Name())
		t.Fatalf("failed to open bbolt: %v", err)
	}

	oldCache := BCache
	BCache = db
	mainCacheClearTime = time.Now()

	return func() {
		_ = db.Close()
		BCache = oldCache
		_ = os.Remove(tmpFile.Name())
	}
}

func TestCacheSetAndGet(t *testing.T) {
	cleanup := setupTestCache(t)
	defer cleanup()

	// Set a value
	if err := CacheSet("key1", []byte("value1"), time.Hour); err != nil {
		t.Fatalf("CacheSet failed: %v", err)
	}

	// Get the value back
	got, err := CacheGet("key1")
	if err != nil {
		t.Fatalf("CacheGet failed: %v", err)
	}
	if !bytes.Equal(got, []byte("value1")) {
		t.Errorf("got %q, want %q", got, "value1")
	}
}

func TestCacheGet_NotFound(t *testing.T) {
	cleanup := setupTestCache(t)
	defer cleanup()

	_, err := CacheGet("nonexistent")
	if err != ErrKeyNotFound {
		t.Errorf("expected ErrKeyNotFound, got: %v", err)
	}
}

func TestCacheGet_ExpiredKeyCleanup(t *testing.T) {
	cleanup := setupTestCache(t)
	defer cleanup()

	// Set a key with very short TTL (already expired)
	expired := time.Now().Add(-time.Hour)
	entry := buildCacheEntry(expired, []byte("expired-data"))

	// Manually insert expired entry into bbolt
	if err := BCache.Update(func(tx *bolt.Tx) error {
		bk, err := tx.CreateBucketIfNotExists(mainCacheBucket)
		if err != nil {
			return err
		}
		return bk.Put([]byte("expired-key"), entry)
	}); err != nil {
		t.Fatalf("setup failed: %v", err)
	}

	// CacheGet should return ErrKeyTimeout
	_, err := CacheGet("expired-key")
	if err != ErrKeyTimeout {
		t.Errorf("expected ErrKeyTimeout, got: %v", err)
	}

	// Give the async cleanup goroutine time to run
	time.Sleep(100 * time.Millisecond)

	// Verify the key was actually deleted in the write transaction
	var keyExists bool
	_ = BCache.View(func(tx *bolt.Tx) error {
		bk := tx.Bucket(mainCacheBucket)
		if bk != nil {
			keyExists = bk.Get([]byte("expired-key")) != nil
		}
		return nil
	})
	if keyExists {
		t.Error("expired key should have been deleted by background cleanup")
	}
}

func TestCacheSetsAndGets(t *testing.T) {
	cleanup := setupTestCache(t)
	defer cleanup()

	type testData struct {
		Name  string `json:"name"`
		Value int    `json:"value"`
	}

	// Set structured data
	if err := CacheSets("struct-key", testData{Name: "test", Value: 42}, time.Hour); err != nil {
		t.Fatalf("CacheSets failed: %v", err)
	}

	// Get structured data back
	var got testData
	if err := CacheGets("struct-key", &got); err != nil {
		t.Fatalf("CacheGets failed: %v", err)
	}
	if got.Name != "test" || got.Value != 42 {
		t.Errorf("got %+v, want {test 42}", got)
	}
}

func TestCacheSet_DeleteWithNilData(t *testing.T) {
	cleanup := setupTestCache(t)
	defer cleanup()

	// Set a value first
	if err := CacheSet("del-key", []byte("data"), time.Hour); err != nil {
		t.Fatalf("CacheSet failed: %v", err)
	}

	// Delete by passing nil data
	if err := CacheSet("del-key", nil); err != nil {
		t.Fatalf("CacheSet(nil) failed: %v", err)
	}

	// Should be gone
	_, err := CacheGet("del-key")
	if err != ErrKeyNotFound {
		t.Errorf("expected ErrKeyNotFound after delete, got: %v", err)
	}
}

func TestCacheGets_NilData(t *testing.T) {
	cleanup := setupTestCache(t)
	defer cleanup()

	err := CacheGets("key", nil)
	if err == nil {
		t.Fatal("expected error for nil data parameter")
	}
	if !errors.Is(err, ErrCacheDataNil) {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestCacheFlush_Integration(t *testing.T) {
	cleanup := setupTestCache(t)
	defer cleanup()

	// Set some values
	if err := CacheSet("k1", []byte("v1"), time.Hour); err != nil {
		t.Fatalf("CacheSet k1: %v", err)
	}
	if err := CacheSet("k2", []byte("v2"), time.Hour); err != nil {
		t.Fatalf("CacheSet k2: %v", err)
	}

	// Flush
	if err := CacheFlush(); err != nil {
		t.Fatalf("CacheFlush failed: %v", err)
	}

	// Keys should be gone (bucket deleted)
	_, err := CacheGet("k1")
	if err != ErrKeyNotFound {
		t.Errorf("expected ErrKeyNotFound after flush, got: %v", err)
	}
}

func TestCache_NilBCache(t *testing.T) {
	oldCache := BCache
	BCache = nil
	defer func() { BCache = oldCache }()

	if err := CacheSet("key", []byte("data")); err == nil {
		t.Error("CacheSet with nil BCache should return error")
	}
	if _, err := CacheGet("key"); err == nil {
		t.Error("CacheGet with nil BCache should return error")
	}
	if err := CacheGets("key", &struct{}{}); err == nil {
		t.Error("CacheGets with nil BCache should return error")
	}
	if err := CacheSets("key", "data"); err == nil {
		t.Error("CacheSets with nil BCache should return error")
	}
	if err := CacheFlush(); err == nil {
		t.Error("CacheFlush with nil BCache should return error")
	}
}
