package firebase_client

import (
	"crypto/tls"
	"errors"
	"fmt"
	firebase_api "github.com/BRUHItsABunny/go-android-firebase/api"
	"github.com/BRUHItsABunny/go-android-firebase/constants"
	"github.com/davecgh/go-spew/spew"
	"go.uber.org/atomic"
	"google.golang.org/protobuf/proto"
	"io"
	"net"
	"strconv"
	"sync"
)

type MTalkMessageProcessor func(message proto.Message)
type MTalkNotificationProcessor func(notification *firebase_api.DataMessageStanza)

type MTalkCon struct {
	RawConn net.Conn
	sync.WaitGroup
	OnMessage        MTalkMessageProcessor
	OnNotification   MTalkNotificationProcessor
	lastPersistentId string
	streamIdReported int
	streamId         int
	stop             *atomic.Bool
	Device           *firebase_api.FirebaseDevice
}

const MTalkVersion = byte(41)

func NewMTalkCon(device *firebase_api.FirebaseDevice) *MTalkCon {
	result := &MTalkCon{stop: atomic.NewBool(false), Device: device}
	result.OnMessage = result.defaultOnMessage
	result.OnNotification = result.defaultOnNotification
	return result
}

func (c *MTalkCon) Connect() error {
	// Connect
	connV2, err := tls.Dial("tcp", fmt.Sprintf("%s:%s", constants.MTalkHost, constants.MTalkPort), nil)
	if err != nil {
		return fmt.Errorf("tls.Dial: %w", err)
	}
	c.RawConn = connV2

	// login
	err = c.writeByte(MTalkVersion) // Write version first
	if err != nil {
		return fmt.Errorf("c.writeByte[version]: %w", err)
	}

	err = c.login()
	if err != nil {
		return fmt.Errorf("result.login: %w", err)
	}

	version, err := c.readByte() // read version
	if err != nil {
		return fmt.Errorf("c.readByte[version]: %w", err)
	}

	if version != MTalkVersion {
		return errors.New("mtalk version not consistent")
	}
	// fmt.Println(fmt.Sprintf("version: %d", version))

	loginResp, err := c.readMessage()
	if err != nil {
		return fmt.Errorf("result.readMessage: %w", err)
	}
	loginRespParsed, ok := loginResp.(*firebase_api.LoginResponse)
	if !ok {
		return errors.New("didn't receive login response")
	}
	if loginRespParsed.Error != nil {
		return errors.New(*loginRespParsed.Error.Message)
	}
	// Start goroutine to read in background and notify us?
	c.Add(1)
	go c.loop()
	return nil
}

func (c *MTalkCon) loop() {
	for {
		msg, err := c.readMessage()
		if err != nil {
			panic(err)
		}
		c.OnMessage(msg)
		if c.stop.Load() {
			break
		}
	}
	c.Done()
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
		err := c.writeMessage(firebase_api.MCSTag_MCS_HEARTBEAT_ACK_TAG, response)
		if err != nil {
			err = fmt.Errorf("c.writeMessage[PingAck]: %w", err)
			panic(err)
		}
		break
	case *firebase_api.DataMessageStanza:
		if parsedMsg.PersistentId != nil {
			c.Device.MTalkLastPersistentId = *parsedMsg.PersistentId
		}
		c.OnNotification(parsedMsg)
		break
	}
}

func (c *MTalkCon) defaultOnNotification(notification *firebase_api.DataMessageStanza) {
	fmt.Println(spew.Sdump(notification))
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

	var result proto.Message
	switch firebase_api.MCSTag(int(tag)) {
	case firebase_api.MCSTag_MCS_HEARTBEAT_PING_TAG:
		result = &firebase_api.HeartbeatPing{}
		break
	case firebase_api.MCSTag_MCS_HEARTBEAT_ACK_TAG:
		result = &firebase_api.HeartbeatAck{}
		break
	case firebase_api.MCSTag_MCS_LOGIN_REQUEST_TAG:
		result = &firebase_api.LoginRequest{}
		break
	case firebase_api.MCSTag_MCS_LOGIN_RESPONSE_TAG:
		result = &firebase_api.LoginResponse{}
		break
	case firebase_api.MCSTag_MCS_CLOSE_TAG:
		result = &firebase_api.Close{}
		break
	case firebase_api.MCSTag_MCS_IQ_STANZA_TAG:
		result = &firebase_api.IqStanza{}
		break
	case firebase_api.MCSTag_MCS_DATA_MESSAGE_STANZA_TAG:
		result = &firebase_api.DataMessageStanza{}
		break
	}
	err = proto.Unmarshal(data, result)
	if err != nil {
		return nil, fmt.Errorf("proto.Unmarshal: %w", err)
	}
	c.streamId++
	// fmt.Println("IO:IN:\n", spew.Sdump(result))
	return result, nil
}

func (c *MTalkCon) writeMessage(tag firebase_api.MCSTag, message proto.Message) error {
	// fmt.Println("IO:OUT:\n", spew.Sdump(message))
	protoBytes, err := proto.Marshal(message)
	if err != nil {
		return fmt.Errorf("proto.Marshal: %w", err)
	}
	err = c.writeByte(uint8(tag))
	if err != nil {
		return fmt.Errorf("c.writeByte[tag]: %w", err)
	}
	err = c.writeVarInt(len(protoBytes))
	if err != nil {
		return fmt.Errorf("c.writeVarInt: %w", err)
	}
	err = c.writeBytes(protoBytes)
	if err != nil {
		return fmt.Errorf("c.writeByte[protobytes]: %w", err)
	}
	return nil
}

func (c *MTalkCon) login() error {
	authSvc := firebase_api.LoginRequest_ANDROID_ID
	request := &firebase_api.LoginRequest{
		Id:        proto.String("gms-22.48.14-000"),
		Domain:    proto.String("mcs.android.com"),
		User:      proto.String(strconv.FormatInt(c.Device.CheckinAndroidID, 10)),
		Resource:  proto.String(strconv.FormatInt(c.Device.CheckinAndroidID, 10)),
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
		ReceivedPersistentId: []string{c.lastPersistentId},
		AdaptiveHeartbeat:    proto.Bool(false),
		UseRmq2:              proto.Bool(true),
		AuthService:          &authSvc,
		NetworkType:          proto.Int32(1),
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
	}
	return int(result), nil
}

func (c *MTalkCon) writeVarInt(value int) error {
	for {
		if (value & ^0x7F) == 0 {
			err := c.writeByte(byte(value))
			if err != nil {
				return fmt.Errorf("c.writeByte[0]: %w", err)
			}
			break
		} else {
			err := c.writeByte(byte((value & 0x7F) | 0x80))
			if err != nil {
				return fmt.Errorf("c.writeByte: %w", err)
			}
			u := uint32(value)
			value = int(u >> 7)
		}
	}
	return nil
}

func (c *MTalkCon) readBytes(len int) ([]byte, error) {
	buf := make([]byte, len)
	var result []byte
	read, err := c.RawConn.Read(buf)
	if err != nil && !errors.Is(err, io.EOF) {
		err = fmt.Errorf(" c.RawConn.Read: %w", err)
	} else {
		err = nil
	}
	result = buf[:read]
	// fmt.Println(fmt.Sprintf("%s\tIO:BYTESIN:%s", time.Now().Format(time.RFC3339), hex.EncodeToString(result)))
	return result, err
}

func (c *MTalkCon) readByte() (byte, error) {
	buf, err := c.readBytes(1)
	if err != nil {
		return 0, err
	}
	if len(buf) != 1 {
		return 0, errors.New("no data read")
	}
	return buf[0], nil
}

func (c *MTalkCon) writeBytes(data []byte) error {
	// fmt.Println(fmt.Sprintf("%s\tIO:BYTESOUT:%s", time.Now().Format(time.RFC3339), hex.EncodeToString(data)))
	_, err := c.RawConn.Write(data)
	if err != nil {
		// return fmt.Errorf("c.IO.WriteMessage: %w", err)
		return fmt.Errorf("c.RawConn.Write: %w", err)
	}
	return nil
}

func (c *MTalkCon) writeByte(data byte) error {
	return c.writeBytes([]byte{data})
}
