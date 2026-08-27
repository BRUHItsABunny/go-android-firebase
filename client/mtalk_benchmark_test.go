package firebase_client

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"errors"
	"io"
	"net"
	"testing"

	firebase_api "github.com/BRUHItsABunny/go-android-firebase/api"
	"google.golang.org/protobuf/proto"
)

// discardConn implements a zero-allocation discard sink for write benchmarks.
type discardConn struct {
	net.Conn
}

func (d *discardConn) Write(b []byte) (int, error) {
	return len(b), nil
}

func (d *discardConn) Close() error {
	return nil
}

// repeatingByteReader provides an infinite stream of varint bytes without allocations.
type repeatingByteReader struct {
	data  [binary.MaxVarintLen64]byte
	len   int
	index int
}

func newRepeatingByteReader(val uint64) *repeatingByteReader {
	r := &repeatingByteReader{}
	r.len = binary.PutUvarint(r.data[:], val)
	return r
}

func (r *repeatingByteReader) Read(b []byte) (int, error) {
	for i := range b {
		b[i] = r.data[r.index]
		r.index++
		if r.index >= r.len {
			r.index = 0
		}
	}
	return len(b), nil
}

func BenchmarkWriteVarInt(b *testing.B) {
	conn := &discardConn{}
	mtalk := &MTalkCon{
		RawConn: conn,
	}

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		if err := mtalk.writeVarIntLocked(1048576); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkReadVarInt(b *testing.B) {
	reader := newRepeatingByteReader(1048576)
	mtalk := &MTalkCon{
		reader: bufio.NewReaderSize(reader, 64*1024),
	}

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		val, err := mtalk.readVarInt()
		if err != nil {
			b.Fatal(err)
		}
		if val != 1048576 {
			b.Fatalf("unexpected val: %d", val)
		}
	}
}

func TestMTalk_FullFrameTransmission(t *testing.T) {
	t.Parallel()

	serverConn, clientConn := net.Pipe()
	defer func() {
		_ = serverConn.Close()
		_ = clientConn.Close()
	}()

	sender := &MTalkCon{
		RawConn: clientConn,
	}

	receiver := &MTalkCon{
		reader:  bufio.NewReader(serverConn),
		RawConn: serverConn,
	}

	testFrames := []struct {
		name    string
		tag     firebase_api.MCSTag
		message proto.Message
	}{
		{
			name: "HeartbeatPing",
			tag:  firebase_api.MCSTag_MCS_HEARTBEAT_PING_TAG,
			message: &firebase_api.HeartbeatPing{
				StreamId:             proto.Int32(42),
				LastStreamIdReceived: proto.Int32(10),
				Status:               proto.Int64(1),
			},
		},
		{
			name: "HeartbeatAck",
			tag:  firebase_api.MCSTag_MCS_HEARTBEAT_ACK_TAG,
			message: &firebase_api.HeartbeatAck{
				StreamId:             proto.Int32(43),
				LastStreamIdReceived: proto.Int32(42),
				Status:               proto.Int64(1),
			},
		},
		{
			name: "DataMessageStanza with payload",
			tag:  firebase_api.MCSTag_MCS_DATA_MESSAGE_STANZA_TAG,
			message: &firebase_api.DataMessageStanza{
				Id:           proto.String("msg-id-12345"),
				From:         proto.String("project-id-99"),
				Category:     proto.String("com.example.app"),
				PersistentId: proto.String("pid-8888"),
				RawData:      bytes.Repeat([]byte{0xDE, 0xAD, 0xBE, 0xEF}, 1024), // 4 KiB payload
			},
		},
	}

	for _, frame := range testFrames {
		frame := frame
		t.Run(frame.name, func(t *testing.T) {
			errChan := make(chan error, 1)
			go func() {
				errChan <- sender.writeMessage(frame.tag, frame.message)
			}()

			receivedMsg, err := receiver.readMessage()
			if err != nil {
				t.Fatalf("receiver.readMessage() error: %v", err)
			}

			writeErr := <-errChan
			if writeErr != nil {
				t.Fatalf("sender.writeMessage() error: %v", writeErr)
			}

			if !proto.Equal(receivedMsg, frame.message) {
				t.Fatalf("frame mismatch:\nsent: %v\nreceived: %v", frame.message, receivedMsg)
			}
		})
	}
}

func TestMTalk_FrameReadEOFHandling(t *testing.T) {
	t.Parallel()

	serverConn, clientConn := net.Pipe()
	_ = clientConn.Close() // Immediate close to trigger EOF on first byte

	receiver := &MTalkCon{
		reader:  bufio.NewReader(serverConn),
		RawConn: serverConn,
	}

	_, err := receiver.readMessage()
	if err == nil {
		t.Fatal("expected error on closed pipe, got nil")
	}
	if !errors.Is(err, io.EOF) {
		t.Fatalf("expected io.EOF, got: %v", err)
	}
}
