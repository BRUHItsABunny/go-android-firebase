package firebase_client

import (
	"bufio"
	"context"
	"crypto/ecdh"
	"crypto/rand"
	"crypto/tls"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/BRUHItsABunny/ecego"
	firebase_api "github.com/BRUHItsABunny/go-android-firebase/api"
	"github.com/BRUHItsABunny/go-android-firebase/constants"
	"github.com/davecgh/go-spew/spew"
	"go.uber.org/atomic"
	"google.golang.org/protobuf/proto"
)

type MTalkMessageProcessor func(message proto.Message)
type MTalkNotificationProcessor func(notification *firebase_api.DataMessageStanza)

// MTalkErrorProcessor is called for every error the background read loop runs into.
// Errors that ended the loop are reported with fatal set to true.
type MTalkErrorProcessor func(err error, fatal bool)

type MTalkCon struct {
	RawConn net.Conn
	sync.WaitGroup
	OnMessage      MTalkMessageProcessor
	OnNotification MTalkNotificationProcessor
	// OnError receives read-loop errors. The default implementation stores the error so it
	// can be retrieved with Err(), previous versions of this library panicked instead.
	OnError MTalkErrorProcessor
	// Debug makes the default notification handler dump every notification to stdout.
	Debug bool

	streamIdReported int
	streamId         int
	stop             *atomic.Bool
	connected        *atomic.Bool
	Device           *firebase_api.FirebaseDevice
	ECEEngine        *ecego.Engine

	reader   *bufio.Reader
	writeMux sync.Mutex

	errMux  sync.Mutex
	lastErr error
	closed  chan struct{}
}

const MTalkVersion = byte(41)

// DefaultMTalkDialTimeout bounds the TLS handshake with mtalk.google.com.
const DefaultMTalkDialTimeout = 30 * time.Second

// mtalkReadBufferSize keeps the per-message reads out of the syscall path, the protocol is
// parsed one byte at a time so an unbuffered connection meant a syscall per byte.
const mtalkReadBufferSize = 32 * 1024

var (
	mapEncryptionSchemes = map[string]ecego.Version{
		string(ecego.AES128GCM): ecego.AES128GCM,
		string(ecego.AESGCM):    ecego.AESGCM,
		string(ecego.AESGCM128): ecego.AESGCM128,
	}
)

func NewMTalkCon(device *firebase_api.FirebaseDevice) (*MTalkCon, error) {
	if device == nil {
		return nil, firebase_api.ErrNilDevice
	}

	result := &MTalkCon{
		stop:      atomic.NewBool(false),
		connected: atomic.NewBool(false),
		Device:    device,
		closed:    make(chan struct{}),
	}
	result.OnMessage = result.defaultOnMessage
	result.OnNotification = result.defaultOnNotification
	result.OnError = result.defaultOnError

	authSecret, err := loadOrCreateAuthSecret(device)
	if err != nil {
		return nil, err
	}
	privateKey, err := loadOrCreatePrivateKey(device)
	if err != nil {
		return nil, err
	}

	result.ECEEngine = ecego.NewEngine(ecego.SingleKey(privateKey), ecego.WithAuthSecret(authSecret))
	return result, nil
}

// loadOrCreateAuthSecret restores the web push auth secret from the device, generating a
// new one when the device does not carry one yet.
func loadOrCreateAuthSecret(device *firebase_api.FirebaseDevice) ([]byte, error) {
	if device.MTalkAuthSecret != "" {
		authSecret, err := base64.RawURLEncoding.DecodeString(device.MTalkAuthSecret)
		if err != nil {
			return nil, fmt.Errorf("base64.RawURLEncoding.DecodeString[authSecret]: %w", err)
		}
		return authSecret, nil
	}

	authSecret := make([]byte, 16)
	if _, err := rand.Read(authSecret); err != nil {
		return nil, fmt.Errorf("rand.Read[authSecret]: %w", err)
	}
	device.MTalkAuthSecret = base64.RawURLEncoding.EncodeToString(authSecret)
	return authSecret, nil
}

// loadOrCreatePrivateKey restores the web push key pair from the device, generating a new
// pair when the device does not carry one yet. Both halves are written back to the device
// so that a persisted device keeps receiving encrypted notifications after a restart.
func loadOrCreatePrivateKey(device *firebase_api.FirebaseDevice) (*ecdh.PrivateKey, error) {
	if device.MTalkPrivateKey != "" {
		// NOTE: this used to decode MTalkAuthSecret here, which made a persisted key pair
		// unusable and silently broke decryption of every incoming notification.
		privateKeyBytes, err := base64.RawURLEncoding.DecodeString(device.MTalkPrivateKey)
		if err != nil {
			return nil, fmt.Errorf("base64.RawURLEncoding.DecodeString[privateKey]: %w", err)
		}
		privateKey, err := ecdh.P256().NewPrivateKey(privateKeyBytes)
		if err != nil {
			return nil, fmt.Errorf("ecdh.P256().NewPrivateKey: %w", err)
		}
		device.MTalkPublicKey = base64.RawURLEncoding.EncodeToString(privateKey.PublicKey().Bytes())
		return privateKey, nil
	}

	privateKey, err := ecdh.P256().GenerateKey(rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("ecdh.P256().GenerateKey: %w", err)
	}
	device.MTalkPrivateKey = base64.RawURLEncoding.EncodeToString(privateKey.Bytes())
	device.MTalkPublicKey = base64.RawURLEncoding.EncodeToString(privateKey.PublicKey().Bytes())
	return privateKey, nil
}

// Connect dials mtalk.google.com, logs in and starts the background read loop.
func (c *MTalkCon) Connect() error {
	return c.ConnectContext(context.Background())
}

// ConnectContext is Connect with a caller controlled deadline for dialling and logging in.
func (c *MTalkCon) ConnectContext(ctx context.Context) error {
	if c.connected.Load() {
		return firebase_api.ErrAlreadyConnected
	}
	if err := c.Device.ValidateForCheckinScopedCall(); err != nil {
		return err
	}

	dialer := &tls.Dialer{NetDialer: &net.Dialer{Timeout: DefaultMTalkDialTimeout}}
	address := net.JoinHostPort(constants.MTalkHost, constants.MTalkPort)
	conn, err := dialer.DialContext(ctx, "tcp", address)
	if err != nil {
		return fmt.Errorf("tls.Dial[%s]: %w", address, err)
	}

	c.RawConn = conn
	c.reader = bufio.NewReaderSize(conn, mtalkReadBufferSize)
	c.stop.Store(false)
	c.setErr(nil)
	c.closed = make(chan struct{})

	if deadline, ok := ctx.Deadline(); ok {
		_ = conn.SetDeadline(deadline)
	}

	if err = c.handshake(); err != nil {
		_ = conn.Close()
		c.reader = nil
		c.RawConn = nil
		return err
	}

	// Clear the handshake deadline, the loop below blocks until a message arrives.
	_ = conn.SetDeadline(time.Time{})
	c.connected.Store(true)

	c.Add(1)
	go c.loop()
	return nil
}

// handshake writes the login request and validates the login response.
func (c *MTalkCon) handshake() error {
	if err := c.writeByte(MTalkVersion); err != nil { // Write version first
		return fmt.Errorf("c.writeByte[version]: %w", err)
	}
	if err := c.login(); err != nil {
		return fmt.Errorf("c.login: %w", err)
	}

	version, err := c.readByte() // read version
	if err != nil {
		return fmt.Errorf("c.readByte[version]: %w", err)
	}
	if version != MTalkVersion {
		return fmt.Errorf("mtalk version not consistent: got %d, want %d", version, MTalkVersion)
	}

	loginResp, err := c.readMessage()
	if err != nil {
		return fmt.Errorf("c.readMessage: %w", err)
	}
	loginRespParsed, ok := loginResp.(*firebase_api.LoginResponse)
	if !ok {
		if closeMsg, isClose := loginResp.(*firebase_api.Close); isClose {
			return fmt.Errorf("mtalk closed the connection during login: %s", closeMsg.String())
		}
		return fmt.Errorf("didn't receive login response, got %T", loginResp)
	}
	if loginRespParsed.Error != nil {
		return fmt.Errorf("mtalk login rejected: %s (code %d)",
			loginRespParsed.Error.GetMessage(), loginRespParsed.Error.GetCode())
	}
	return nil
}

// Connected reports whether the read loop is running.
func (c *MTalkCon) Connected() bool {
	return c.connected.Load()
}

// Err returns the error that ended the read loop, if any.
func (c *MTalkCon) Err() error {
	c.errMux.Lock()
	defer c.errMux.Unlock()
	return c.lastErr
}

func (c *MTalkCon) setErr(err error) {
	c.errMux.Lock()
	c.lastErr = err
	c.errMux.Unlock()
}

// Closed returns a channel that is closed once the read loop has stopped, which makes it
// possible to select on a dropped connection instead of polling Connected.
func (c *MTalkCon) Closed() <-chan struct{} {
	return c.closed
}

// Close stops the read loop and closes the connection. It is safe to call more than once,
// a second call reports ErrNotConnected. It waits for the read loop to finish, so do not
// call it from inside OnMessage/OnNotification, those run on that very loop.
func (c *MTalkCon) Close() error {
	if !c.connected.Load() {
		return firebase_api.ErrNotConnected
	}
	c.stop.Store(true)

	var err error
	if c.RawConn != nil {
		// Closing unblocks the read loop, which then exits and marks us disconnected.
		if closeErr := c.RawConn.Close(); closeErr != nil && !errors.Is(closeErr, net.ErrClosed) {
			err = fmt.Errorf("c.RawConn.Close: %w", closeErr)
		}
	}
	c.Wait()
	return err
}

// loop reads messages until the connection drops or Close is called.
// Errors are reported through OnError, this used to panic and take the whole process down.
func (c *MTalkCon) loop() {
	defer func() {
		c.connected.Store(false)
		close(c.closed)
		c.WaitGroup.Done()
	}()

	for {
		msg, err := c.readMessage()
		if err != nil {
			if c.stop.Load() || errors.Is(err, net.ErrClosed) || errors.Is(err, io.EOF) {
				// Expected when Close was called or the peer hung up.
				if !c.stop.Load() {
					c.reportError(fmt.Errorf("mtalk connection lost: %w", err), true)
				}
				return
			}
			c.reportError(fmt.Errorf("c.readMessage: %w", err), true)
			return
		}

		c.dispatch(msg)

		if c.stop.Load() {
			return
		}
	}
}

// dispatch hands a message to the user's handler, containing panics so that one bad
// handler cannot take the read loop (or the process) with it.
func (c *MTalkCon) dispatch(msg proto.Message) {
	defer func() {
		if r := recover(); r != nil {
			c.reportError(fmt.Errorf("panic in message handler: %v", r), false)
		}
	}()
	if c.OnMessage != nil {
		c.OnMessage(msg)
	}
}

func (c *MTalkCon) reportError(err error, fatal bool) {
	if fatal {
		c.setErr(err)
	}
	if c.OnError != nil {
		c.OnError(err, fatal)
	}
}

func (c *MTalkCon) defaultOnError(err error, fatal bool) {
	if c.Debug {
		fmt.Printf("mtalk error (fatal=%t): %v\n", fatal, err)
	}
}

func (c *MTalkCon) defaultOnMessage(msg proto.Message) {
	switch parsedMsg := msg.(type) {
	case *firebase_api.HeartbeatPing:
		response := &firebase_api.HeartbeatAck{
			Status: parsedMsg.Status,
		}
		if c.streamId != c.streamIdReported {
			c.streamIdReported = c.streamId
			response.LastStreamIdReceived = proto.Int32(int32(c.streamIdReported))
		}
		if err := c.writeMessage(firebase_api.MCSTag_MCS_HEARTBEAT_ACK_TAG, response); err != nil {
			c.reportError(fmt.Errorf("c.writeMessage[PingAck]: %w", err), false)
		}
	case *firebase_api.Close:
		c.reportError(errors.New("mtalk sent a close stanza"), false)
	case *firebase_api.DataMessageStanza:
		c.handleDataMessage(parsedMsg)
	}
}

// handleDataMessage decrypts a notification (when it is encrypted) and forwards it.
// A decryption failure is reported through OnError and the message is dropped, previous
// versions panicked and killed the process on a single malformed notification.
func (c *MTalkCon) handleDataMessage(parsedMsg *firebase_api.DataMessageStanza) {
	if parsedMsg.PersistentId != nil {
		c.Device.MTalkLastPersistentId = *parsedMsg.PersistentId
	}

	// What encryption scheme is it using
	encryptionParams := ecego.OperationalParams{}
	parsedAppData := make(map[string]string, len(parsedMsg.AppData))
	for _, appData := range parsedMsg.AppData {
		key := strings.ToLower(appData.GetKey())
		parsedAppData[key] = appData.GetValue()
		if key == "content-encoding" {
			encryptionParams.Version = mapEncryptionSchemes[appData.GetValue()]
		}
	}

	// Parse params for scheme
	switch encryptionParams.Version {
	case ecego.AESGCM128, ecego.AESGCM:
		if encryptionAppData, ok := parsedAppData["encryption"]; ok {
			if saltStr, ok := parseAppDataValue(encryptionAppData)["salt"]; ok {
				salt, err := base64.RawURLEncoding.DecodeString(saltStr)
				if err != nil {
					c.reportError(fmt.Errorf("base64.RawURLEncoding.DecodeString[%s:salt]: %w", encryptionParams.Version, err), false)
					return
				}
				encryptionParams.Salt = salt
			}
		}
		if cryptoKeyAppData, ok := parsedAppData["crypto-key"]; ok {
			if dhStr, ok := parseAppDataValue(cryptoKeyAppData)["dh"]; ok {
				dh, err := base64.RawURLEncoding.DecodeString(dhStr)
				if err != nil {
					c.reportError(fmt.Errorf("base64.RawURLEncoding.DecodeString[%s:dh]: %w", encryptionParams.Version, err), false)
					return
				}
				encryptionParams.DH = dh
			}
		}

		// Some senders wrap the ciphertext in {"raw_data": "<base64>"}.
		parsedRawData := map[string]string{}
		if err := json.Unmarshal(parsedMsg.RawData, &parsedRawData); err == nil {
			if rawData, ok := parsedRawData["raw_data"]; ok {
				// NOTE: this used to assign the decoded bytes only when decoding *failed*.
				rawDataBytes, decodeErr := base64.StdEncoding.DecodeString(rawData)
				if decodeErr == nil {
					parsedMsg.RawData = rawDataBytes
				}
			}
		}
	}

	// Decrypt and replace raw data
	if encryptionParams.Version != "" {
		plaintext, err := c.ECEEngine.Decrypt(parsedMsg.RawData, nil, encryptionParams)
		if err != nil {
			c.reportError(fmt.Errorf("c.ECEEngine.Decrypt[%s]: %w", encryptionParams.Version, err), false)
			return
		}
		parsedMsg.RawData = plaintext
	}

	if c.OnNotification != nil {
		c.OnNotification(parsedMsg)
	}
}

// defaultOnNotification drops notifications unless Debug is set, in which case it dumps
// them to stdout. Set OnNotification to consume them.
func (c *MTalkCon) defaultOnNotification(notification *firebase_api.DataMessageStanza) {
	if c.Debug {
		fmt.Println(spew.Sdump(notification))
	}
}

func (c *MTalkCon) readMessage() (proto.Message, error) {
	tag, err := c.readByte()
	if err != nil {
		return nil, fmt.Errorf("c.readByte: %w", err)
	}
	length, err := c.readVarInt()
	if err != nil {
		return nil, fmt.Errorf("c.readVarInt: %w", err)
	}
	data, err := c.readBytes(length)
	if err != nil {
		return nil, fmt.Errorf("c.readBytes data: %w", err)
	}

	result, err := newMessageForTag(firebase_api.MCSTag(int(tag)))
	if err != nil {
		return nil, err
	}
	if err = proto.Unmarshal(data, result); err != nil {
		return nil, fmt.Errorf("proto.Unmarshal[%x]: %w", data, err)
	}

	c.streamId++
	return result, nil
}

// newMessageForTag returns an empty message of the type the MCS tag denotes.
func newMessageForTag(tag firebase_api.MCSTag) (proto.Message, error) {
	switch tag {
	case firebase_api.MCSTag_MCS_HEARTBEAT_PING_TAG:
		return &firebase_api.HeartbeatPing{}, nil
	case firebase_api.MCSTag_MCS_HEARTBEAT_ACK_TAG:
		return &firebase_api.HeartbeatAck{}, nil
	case firebase_api.MCSTag_MCS_LOGIN_REQUEST_TAG:
		return &firebase_api.LoginRequest{}, nil
	case firebase_api.MCSTag_MCS_LOGIN_RESPONSE_TAG:
		return &firebase_api.LoginResponse{}, nil
	case firebase_api.MCSTag_MCS_CLOSE_TAG:
		return &firebase_api.Close{}, nil
	case firebase_api.MCSTag_MCS_IQ_STANZA_TAG:
		return &firebase_api.IqStanza{}, nil
	case firebase_api.MCSTag_MCS_DATA_MESSAGE_STANZA_TAG:
		return &firebase_api.DataMessageStanza{}, nil
	}
	return nil, fmt.Errorf("unknown tag: %d", tag)
}

func (c *MTalkCon) writeMessage(tag firebase_api.MCSTag, message proto.Message) error {
	protoBytes, err := proto.Marshal(message)
	if err != nil {
		return fmt.Errorf("proto.Marshal: %w", err)
	}

	// One message must reach the wire in one piece, concurrent writers would interleave.
	c.writeMux.Lock()
	defer c.writeMux.Unlock()

	if err = c.writeByteLocked(uint8(tag)); err != nil {
		return fmt.Errorf("c.writeByte[tag]: %w", err)
	}
	if err = c.writeVarIntLocked(len(protoBytes)); err != nil {
		return fmt.Errorf("c.writeVarInt: %w", err)
	}
	if err = c.writeBytesLocked(protoBytes); err != nil {
		return fmt.Errorf("c.writeBytes[protobytes]: %w", err)
	}
	return nil
}

func (c *MTalkCon) login() error {
	authSvc := firebase_api.LoginRequest_ANDROID_ID
	androidID := strconv.FormatInt(c.Device.CheckinAndroidID, 10)
	request := &firebase_api.LoginRequest{
		Id:        proto.String("gms-22.48.14-000"),
		Domain:    proto.String("mcs.android.com"),
		User:      proto.String(androidID),
		Resource:  proto.String(androidID),
		AuthToken: proto.String(strconv.FormatUint(c.Device.CheckinSecurityToken, 10)),
		DeviceId:  proto.String(fmt.Sprintf("android-%s", c.Device.Device.Id.ToHexString())),
		Setting: []*firebase_api.Setting{{
			Name:  proto.String("new_vc"),
			Value: proto.String("1"),
		}, {
			Name:  proto.String("os_ver"),
			Value: proto.String(fmt.Sprintf("android-%d", c.Device.Device.Version)),
		}, {
			Name:  proto.String("ERR"),
			Value: proto.String("20"),
		}, {
			Name:  proto.String("CT"),
			Value: proto.String("8"),
		}, {
			Name:  proto.String("CONOK"),
			Value: proto.String("3"),
		}, {
			Name:  proto.String("u:f"),
			Value: proto.String("0"),
		}, {
			Name:  proto.String("networkOn"),
			Value: proto.String("0"),
		}},
		AdaptiveHeartbeat: proto.Bool(false),
		UseRmq2:           proto.Bool(true),
		AuthService:       &authSvc,
		NetworkType:       proto.Int32(1),
	}
	// Only acknowledge a persistent id when we actually have one, an empty entry makes
	// the server replay every pending message.
	if c.Device.MTalkLastPersistentId != "" {
		request.ReceivedPersistentId = []string{c.Device.MTalkLastPersistentId}
	}
	return c.writeMessage(firebase_api.MCSTag_MCS_LOGIN_REQUEST_TAG, request)
}

func (c *MTalkCon) readVarInt() (int, error) {
	shift := uint(0)
	result := int64(0)
	for {
		b, err := c.readByte()
		if err != nil {
			return 0, fmt.Errorf("c.readByte: %w", err)
		}
		result |= int64(b&0x7f) << shift
		if (b & 0x80) != 0x80 {
			break
		}
		shift += 7
		if shift > 35 {
			return 0, errors.New("varint is too long")
		}
	}
	if result < 0 {
		return 0, fmt.Errorf("negative varint: %d", result)
	}
	return int(result), nil
}

func (c *MTalkCon) writeVarIntLocked(value int) error {
	for {
		if (value & ^0x7F) == 0 {
			if err := c.writeByteLocked(byte(value)); err != nil {
				return fmt.Errorf("c.writeByte[0]: %w", err)
			}
			return nil
		}
		if err := c.writeByteLocked(byte((value & 0x7F) | 0x80)); err != nil {
			return fmt.Errorf("c.writeByte: %w", err)
		}
		value = int(uint32(value) >> 7)
	}
}

// maxMessageSize guards against a corrupt length prefix asking us to allocate gigabytes.
const maxMessageSize = 32 << 20 // 32 MiB

func (c *MTalkCon) readBytes(length int) ([]byte, error) {
	if c.reader == nil {
		return nil, firebase_api.ErrNotConnected
	}
	if length < 0 || length > maxMessageSize {
		return nil, fmt.Errorf("refusing to read %d bytes, message length is out of range", length)
	}
	if length == 0 {
		return nil, nil
	}

	buf := make([]byte, length)
	// NOTE: this used to swallow io.EOF and hand back a zeroed buffer, which turned a
	// dropped connection into an endless stream of bogus "messages".
	if _, err := io.ReadFull(c.reader, buf); err != nil {
		if errors.Is(err, io.ErrUnexpectedEOF) {
			return nil, fmt.Errorf("io.ReadFull: %w", io.EOF)
		}
		return nil, fmt.Errorf("io.ReadFull: %w", err)
	}
	return buf, nil
}

func (c *MTalkCon) readByte() (byte, error) {
	if c.reader == nil {
		return 0, firebase_api.ErrNotConnected
	}
	b, err := c.reader.ReadByte()
	if err != nil {
		return 0, fmt.Errorf("reader.ReadByte: %w", err)
	}
	return b, nil
}

func (c *MTalkCon) writeBytesLocked(data []byte) error {
	if c.RawConn == nil {
		return firebase_api.ErrNotConnected
	}
	if _, err := c.RawConn.Write(data); err != nil {
		return fmt.Errorf("c.RawConn.Write: %w", err)
	}
	return nil
}

func (c *MTalkCon) writeByteLocked(data byte) error {
	return c.writeBytesLocked([]byte{data})
}

// writeByte writes a single byte, taking the write lock itself.
func (c *MTalkCon) writeByte(data byte) error {
	c.writeMux.Lock()
	defer c.writeMux.Unlock()
	return c.writeByteLocked(data)
}

func parseAppDataValue(value string) map[string]string {
	result := make(map[string]string)
	for _, pair := range strings.Split(value, ";") {
		if pair == "" {
			continue
		}
		if key, val, found := strings.Cut(pair, "="); found {
			result[strings.TrimSpace(key)] = val
		}
	}
	return result
}
