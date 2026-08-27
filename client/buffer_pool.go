package firebase_client

import (
	"sync"
)

// maxPooledBufferSize is the maximum byte capacity retained inside the sync.Pool.
// Buffers larger than 64 KiB bypass the pool to prevent memory retention bloat.
const maxPooledBufferSize = 64 * 1024 // 64 KiB

// bufferPool manages reusable byte slices via pointers (*[]byte) to prevent
// interface boxing overhead (24-byte SliceHeader heap allocations).
var bufferPool = sync.Pool{
	New: func() any {
		buf := make([]byte, maxPooledBufferSize)
		return &buf
	},
}

// getBuffer acquires a byte slice buffer with at least the requested length.
// If size exceeds maxPooledBufferSize, a dedicated non-pooled buffer is allocated.
func getBuffer(size int) *[]byte {
	if size <= 0 {
		buf := make([]byte, 0)
		return &buf
	}
	if size > maxPooledBufferSize {
		buf := make([]byte, size)
		return &buf
	}

	bufPtr := bufferPool.Get().(*[]byte)
	if cap(*bufPtr) < size {
		buf := make([]byte, maxPooledBufferSize)
		return &buf
	}
	*bufPtr = (*bufPtr)[:size]
	return bufPtr
}

// putBuffer returns a buffer to the pool if its capacity strictly matches
// maxPooledBufferSize, discarding oversized or malformed buffers.
func putBuffer(bufPtr *[]byte) {
	if bufPtr == nil || cap(*bufPtr) != maxPooledBufferSize {
		return
	}
	*bufPtr = (*bufPtr)[:0]
	bufferPool.Put(bufPtr)
}
