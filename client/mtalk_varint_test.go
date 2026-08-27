package firebase_client

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"errors"
	"io"
	"math"
	"net"
	"testing"

	firebase_api "github.com/BRUHItsABunny/go-android-firebase/api"
)

// mockConn implements a minimal in-memory net.Conn for isolated unit testing.
type mockConn struct {
	net.Conn
	readBuf  *bytes.Buffer
	writeBuf *bytes.Buffer
	closed   bool
}

func newMockConn(input []byte) *mockConn {
	return &mockConn{
		readBuf:  bytes.NewBuffer(input),
		writeBuf: bytes.NewBuffer(nil),
	}
}

func (m *mockConn) Read(b []byte) (n int, err error) {
	if m.closed {
		return 0, net.ErrClosed
	}
	return m.readBuf.Read(b)
}

func (m *mockConn) Write(b []byte) (n int, err error) {
	if m.closed {
		return 0, net.ErrClosed
	}
	return m.writeBuf.Write(b)
}

func (m *mockConn) Close() error {
	m.closed = true
	return nil
}

func TestReadVarInt_TableDriven(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		input     []byte
		wantVal   int
		wantErr   bool
		targetErr error
	}{
		{
			name:    "zero value",
			input:   []byte{0x00},
			wantVal: 0,
			wantErr: false,
		},
		{
			name:    "single byte minimum non-zero",
			input:   []byte{0x01},
			wantVal: 1,
			wantErr: false,
		},
		{
			name:    "single byte maximum (127)",
			input:   []byte{0x7F},
			wantVal: 127,
			wantErr: false,
		},
		{
			name:    "two bytes minimum (128)",
			input:   []byte{0x80, 0x01},
			wantVal: 128,
			wantErr: false,
		},
		{
			name:    "arbitrary two bytes (300)",
			input:   []byte{0xAC, 0x02},
			wantVal: 300,
			wantErr: false,
		},
		{
			name:    "three bytes (65535)",
			input:   []byte{0xFF, 0xFF, 0x03},
			wantVal: 65535,
			wantErr: false,
		},
		{
			name:    "four bytes (1048576)",
			input:   []byte{0x80, 0x80, 0x40},
			wantVal: 1048576,
			wantErr: false,
		},
		{
			name: "exact maxMessageSize (32 MiB = 33554432)",
			input: func() []byte {
				buf := make([]byte, binary.MaxVarintLen64)
				n := binary.PutUvarint(buf, uint64(maxMessageSize))
				return buf[:n]
			}(),
			wantVal: maxMessageSize,
			wantErr: false,
		},
		{
			name: "exceeds maxMessageSize by 1",
			input: func() []byte {
				buf := make([]byte, binary.MaxVarintLen64)
				n := binary.PutUvarint(buf, uint64(maxMessageSize+1))
				return buf[:n]
			}(),
			wantVal: 0,
			wantErr: true,
		},
		{
			name:      "clean EOF on empty buffer",
			input:     []byte{},
			wantVal:   0,
			wantErr:   true,
			targetErr: io.EOF,
		},
		{
			name:      "truncated multi-byte varint",
			input:     []byte{0x80},
			wantVal:   0,
			wantErr:   true,
			targetErr: io.ErrUnexpectedEOF,
		},
		{
			name: "overflow with continuation bit in 10th byte",
			input: []byte{
				0x80, 0x80, 0x80, 0x80, 0x80,
				0x80, 0x80, 0x80, 0x80, 0x80, 0x01,
			},
			wantVal: 0,
			wantErr: true,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			conn := newMockConn(tc.input)
			mtalk := &MTalkCon{
				reader:  bufio.NewReader(conn),
				RawConn: conn,
			}

			val, err := mtalk.readVarInt()
			if tc.wantErr {
				if err == nil {
					t.Fatalf("readVarInt() expected error, got nil (val: %d)", val)
				}
				if tc.targetErr != nil && !errors.Is(err, tc.targetErr) {
					t.Fatalf("readVarInt() error mismatch: got %v, want %v", err, tc.targetErr)
				}
				return
			}

			if err != nil {
				t.Fatalf("readVarInt() unexpected error: %v", err)
			}
			if val != tc.wantVal {
				t.Fatalf("readVarInt() = %d, want %d", val, tc.wantVal)
			}
		})
	}
}

func TestReadVarInt_NilReader(t *testing.T) {
	t.Parallel()

	mtalk := &MTalkCon{
		reader:  nil,
		RawConn: nil,
	}

	val, err := mtalk.readVarInt()
	if !errors.Is(err, firebase_api.ErrNotConnected) {
		t.Fatalf("readVarInt() on nil reader: got %v (val: %d), want ErrNotConnected", err, val)
	}
}

func TestWriteVarInt_TableDriven(t *testing.T) {
	t.Parallel()

	testValues := []int{
		0,
		1,
		127,
		128,
		300,
		65535,
		1048576,
		maxMessageSize,
	}

	for _, val := range testValues {
		val := val
		t.Run(string(rune(val)), func(t *testing.T) {
			t.Parallel()

			conn := newMockConn(nil)
			mtalk := &MTalkCon{
				RawConn: conn,
			}

			err := mtalk.writeVarIntLocked(val)
			if err != nil {
				t.Fatalf("writeVarIntLocked(%d) unexpected error: %v", val, err)
			}

			expected := make([]byte, binary.MaxVarintLen64)
			expectedN := binary.PutUvarint(expected, uint64(val))
			expectedBytes := expected[:expectedN]

			actualBytes := conn.writeBuf.Bytes()
			if !bytes.Equal(actualBytes, expectedBytes) {
				t.Fatalf("writeVarIntLocked(%d) = %x, want %x", val, actualBytes, expectedBytes)
			}
		})
	}
}

func TestWriteVarInt_Disconnected(t *testing.T) {
	t.Parallel()

	mtalk := &MTalkCon{
		RawConn: nil,
	}

	err := mtalk.writeVarIntLocked(100)
	if !errors.Is(err, firebase_api.ErrNotConnected) {
		t.Fatalf("writeVarIntLocked() on nil conn: got %v, want ErrNotConnected", err)
	}
}

func TestWriteVarInt_NegativeValue(t *testing.T) {
	t.Parallel()

	conn := newMockConn(nil)
	mtalk := &MTalkCon{
		RawConn: conn,
	}

	err := mtalk.writeVarIntLocked(-1)
	if err == nil {
		t.Fatal("writeVarIntLocked(-1) expected error for negative value, got nil")
	}
}

func TestVarint_RoundTrip(t *testing.T) {
	t.Parallel()

	serverConn, clientConn := net.Pipe()
	defer func() {
		_ = serverConn.Close()
		_ = clientConn.Close()
	}()

	clientMTalk := &MTalkCon{
		RawConn: clientConn,
	}

	serverMTalk := &MTalkCon{
		reader:  bufio.NewReader(serverConn),
		RawConn: serverConn,
	}

	testValues := []int{
		0,
		1,
		127,
		128,
		255,
		256,
		16384,
		65535,
		1000000,
		maxMessageSize,
	}

	for _, val := range testValues {
		errChan := make(chan error, 1)

		go func(v int) {
			errChan <- clientMTalk.writeVarIntLocked(v)
		}(val)

		readVal, err := serverMTalk.readVarInt()
		if err != nil {
			t.Fatalf("roundtrip read failed for value %d: %v", val, err)
		}

		writeErr := <-errChan
		if writeErr != nil {
			t.Fatalf("roundtrip write failed for value %d: %v", val, writeErr)
		}

		if readVal != val {
			t.Fatalf("roundtrip mismatch: wrote %d, read %d", val, readVal)
		}
	}
}

func TestReadVarInt_MathMaxIntSanity(t *testing.T) {
	t.Parallel()

	// 64-bit varint representing MaxUint64
	buf := make([]byte, binary.MaxVarintLen64)
	n := binary.PutUvarint(buf, math.MaxUint64)

	conn := newMockConn(buf[:n])
	mtalk := &MTalkCon{
		reader:  bufio.NewReader(conn),
		RawConn: conn,
	}

	val, err := mtalk.readVarInt()
	if err == nil {
		t.Fatalf("readVarInt() expected error for MaxUint64, got nil (val: %d)", val)
	}
}
