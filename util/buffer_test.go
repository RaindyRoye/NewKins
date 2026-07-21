package util

import (
	"testing"
)

func TestBufPool_GetAndPut(t *testing.T) {
	bp := newBufPool(1024)
	buf := bp.Get()
	if buf == nil {
		t.Fatal("Get returned nil")
	}
	if len(*buf) != 1024 {
		t.Fatalf("expected buffer size 1024, got %d", len(*buf))
	}
	// Use the buffer
	(*buf)[0] = 42
	bp.Put(buf)

	// Get again — may or may not be the same buffer, but should be valid
	buf2 := bp.Get()
	if buf2 == nil {
		t.Fatal("second Get returned nil")
	}
	if len(*buf2) != 1024 {
		t.Fatalf("expected buffer size 1024, got %d", len(*buf2))
	}
	bp.Put(buf2)
}

func TestBufPool_PutNil(t *testing.T) {
	bp := newBufPool(512)
	// Should not panic
	bp.Put(nil)
}

func TestSmallBufPool(t *testing.T) {
	buf := GetSmallBuf()
	defer PutSmallBuf(buf)
	if buf == nil {
		t.Fatal("GetSmallBuf returned nil")
	}
	if len(*buf) != BufSizeSmall {
		t.Fatalf("expected %d, got %d", BufSizeSmall, len(*buf))
	}
}

func TestMediumBufPool(t *testing.T) {
	buf := GetMediumBuf()
	defer PutMediumBuf(buf)
	if buf == nil {
		t.Fatal("GetMediumBuf returned nil")
	}
	if len(*buf) != BufSizeMedium {
		t.Fatalf("expected %d, got %d", BufSizeMedium, len(*buf))
	}
}

func TestLargeBufPool(t *testing.T) {
	buf := GetLargeBuf()
	defer PutLargeBuf(buf)
	if buf == nil {
		t.Fatal("GetLargeBuf returned nil")
	}
	if len(*buf) != BufSizeLarge {
		t.Fatalf("expected %d, got %d", BufSizeLarge, len(*buf))
	}
}

func TestBufPool_ConcurrentAccess(t *testing.T) {
	bp := newBufPool(256)
	const workers = 10
	const iterations = 100

	done := make(chan bool, workers)
	for i := 0; i < workers; i++ {
		go func() {
			for j := 0; j < iterations; j++ {
				buf := bp.Get()
				(*buf)[0] = 1
				bp.Put(buf)
			}
			done <- true
		}()
	}
	for i := 0; i < workers; i++ {
		<-done
	}
}

func BenchmarkBufPool_Small(b *testing.B) {
	for i := 0; i < b.N; i++ {
		buf := GetSmallBuf()
		PutSmallBuf(buf)
	}
}

func BenchmarkBufPool_Large(b *testing.B) {
	for i := 0; i < b.N; i++ {
		buf := GetLargeBuf()
		PutLargeBuf(buf)
	}
}

func BenchmarkBufPool_SmallAlloc(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = make([]byte, BufSizeSmall)
	}
}

func BenchmarkBufPool_LargeAlloc(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = make([]byte, BufSizeLarge)
	}
}
