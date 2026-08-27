package firebase_client

import (
	"sync"
	"testing"
)

func TestBufferPool_GetAndPut(t *testing.T) {
	t.Parallel()

	sizes := []int{1, 16, 1024, 32 * 1024, maxPooledBufferSize}

	for _, size := range sizes {
		size := size
		t.Run(string(rune(size)), func(t *testing.T) {
			t.Parallel()

			bufPtr := getBuffer(size)
			if bufPtr == nil {
				t.Fatal("getBuffer returned nil")
			}
			if len(*bufPtr) != size {
				t.Fatalf("expected len %d, got %d", size, len(*bufPtr))
			}
			if cap(*bufPtr) < size {
				t.Fatalf("expected cap >= %d, got %d", size, cap(*bufPtr))
			}

			// Write test pattern
			(*bufPtr)[0] = 0xAA
			(*bufPtr)[size-1] = 0xBB

			putBuffer(bufPtr)
		})
	}
}

func TestBufferPool_OversizedHandling(t *testing.T) {
	t.Parallel()

	oversized := maxPooledBufferSize + 1024
	bufPtr := getBuffer(oversized)
	if bufPtr == nil {
		t.Fatal("getBuffer returned nil for oversized request")
	}
	if len(*bufPtr) != oversized {
		t.Fatalf("expected len %d, got %d", oversized, len(*bufPtr))
	}
	if cap(*bufPtr) < oversized {
		t.Fatalf("expected cap >= %d, got %d", oversized, cap(*bufPtr))
	}

	// putBuffer must safely discard oversized buffers without panicking
	putBuffer(bufPtr)
}

func TestBufferPool_EdgeCases(t *testing.T) {
	t.Parallel()

	// Zero size
	zeroBuf := getBuffer(0)
	if zeroBuf == nil || len(*zeroBuf) != 0 {
		t.Fatalf("expected empty buffer, got %v", zeroBuf)
	}
	putBuffer(zeroBuf)

	// Negative size
	negBuf := getBuffer(-10)
	if negBuf == nil || len(*negBuf) != 0 {
		t.Fatalf("expected empty buffer for negative size, got %v", negBuf)
	}
	putBuffer(negBuf)

	// Nil putBuffer
	putBuffer(nil)
}

func TestBufferPool_ConcurrentAccess(t *testing.T) {
	t.Parallel()

	const workers = 20
	const iterations = 500

	var wg sync.WaitGroup
	wg.Add(workers)

	for w := 0; w < workers; w++ {
		go func() {
			defer wg.Done()
			for i := 0; i < iterations; i++ {
				size := (i % 64) * 1024
				bufPtr := getBuffer(size)
				if len(*bufPtr) != size {
					t.Errorf("expected len %d, got %d", size, len(*bufPtr))
					return
				}
				putBuffer(bufPtr)
			}
		}()
	}

	wg.Wait()
}

func BenchmarkBufferPool_GetPut(b *testing.B) {
	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		bufPtr := getBuffer(4096)
		putBuffer(bufPtr)
	}
}
