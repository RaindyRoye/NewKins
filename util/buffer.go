package util

import "sync"

// bufPool is a generic sync.Pool for reusable byte slices.
// Separate pools are maintained for different buffer sizes to avoid
// over-allocating for small reads or under-allocating for large streams.
type bufPool struct {
	pool sync.Pool
	size int
}

func newBufPool(size int) *bufPool {
	return &bufPool{
		pool: sync.Pool{
			New: func() any {
				b := make([]byte, size)
				return &b
			},
		},
		size: size,
	}
}

// Get returns a *[]byte of the configured size from the pool.
// Callers MUST return it with Put when done.
func (bp *bufPool) Get() *[]byte {
	return bp.pool.Get().(*[]byte)
}

// Put returns a buffer to the pool. Safe to call with nil.
func (bp *bufPool) Put(b *[]byte) {
	if b != nil {
		bp.pool.Put(b)
	}
}

// BufSizeSmall is the small I/O buffer size (1 KiB), used for lightweight
// reads such as serving small static files.
const BufSizeSmall = 1024

// BufSizeMedium is the medium I/O buffer size (8 KiB), used for log
// streaming and similar operations.
const BufSizeMedium = 8192

// BufSizeLarge is the large I/O buffer size (32 KiB), used for file
// downloads and uploads where throughput matters.
const BufSizeLarge = 32768

var (
	smallBufPool  = newBufPool(BufSizeSmall)
	mediumBufPool = newBufPool(BufSizeMedium)
	largeBufPool  = newBufPool(BufSizeLarge)
)

// GetSmallBuf returns a reusable 1 KiB byte buffer from the pool.
// Callers MUST call PutSmallBuf when done.
func GetSmallBuf() *[]byte { return smallBufPool.Get() }

// PutSmallBuf returns a small buffer to the pool. Safe to call with nil.
func PutSmallBuf(b *[]byte) { smallBufPool.Put(b) }

// GetMediumBuf returns a reusable 8 KiB byte buffer from the pool.
// Callers MUST call PutMediumBuf when done.
func GetMediumBuf() *[]byte { return mediumBufPool.Get() }

// PutMediumBuf returns a medium buffer to the pool. Safe to call with nil.
func PutMediumBuf(b *[]byte) { mediumBufPool.Put(b) }

// GetLargeBuf returns a reusable 32 KiB byte buffer from the pool.
// Callers MUST call PutLargeBuf when done.
func GetLargeBuf() *[]byte { return largeBufPool.Get() }

// PutLargeBuf returns a large buffer to the pool. Safe to call with nil.
func PutLargeBuf(b *[]byte) { largeBufPool.Put(b) }
