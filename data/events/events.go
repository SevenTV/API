package events

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/seventv/common/utils"
)

type Message[D AnyPayload] struct {
	Op        Opcode `json:"op"`
	Timestamp int64  `json:"t"`
	Data      D      `json:"d"`
	Sequence  uint64 `json:"s,omitempty"`
}

func NewMessage[D AnyPayload](op Opcode, data D) Message[D] {
	msg := Message[D]{
		Op:        op,
		Timestamp: time.Now().UnixMilli(),
		Data:      data,
	}

	return msg
}

func (e Message[D]) ToRaw() Message[json.RawMessage] {
	switch x := utils.ToAny(e.Data).(type) {
	case json.RawMessage:
		return Message[json.RawMessage]{
			Op:        e.Op,
			Timestamp: e.Timestamp,
			Data:      x,
			Sequence:  e.Sequence,
		}
	}

	raw, _ := json.Marshal(e.Data)

	return Message[json.RawMessage]{
		Op:        e.Op,
		Timestamp: e.Timestamp,
		Data:      raw,
		Sequence:  e.Sequence,
	}
}

func ConvertMessage[D AnyPayload](c Message[json.RawMessage]) (Message[D], error) {
	var d D
	err := json.Unmarshal(c.Data, &d)
	c2 := Message[D]{
		Op:        c.Op,
		Timestamp: c.Timestamp,
		Data:      d,
		Sequence:  c.Sequence,
	}

	return c2, err
}

type Opcode uint8

const (
	// Default ops (0-32)
	OpcodeDispatch    Opcode = 0 // R - Server dispatches data to the client
	OpcodeHello       Opcode = 1 // R - Server greets the client
	OpcodeHeartbeat   Opcode = 2 // R - Keep the connection alive
	OpcodeReconnect   Opcode = 4 // R - Server demands that the client reconnects
	OpcodeAck         Opcode = 5 // R - Acknowledgement of an action
	OpcodeError       Opcode = 6 // R - Extra error context in cases where the closing frame is not enough
	OpcodeEndOfStream Opcode = 7 // R - The connection's data stream is ending

	// Commands (33-64)
	OpcodeIdentify    Opcode = 33 // S - Authenticate the session
	OpcodeResume      Opcode = 34 // S - Resume the previous session and receive missed events
	OpcodeSubscribe   Opcode = 35 // S - Subscribe to an event
	OpcodeUnsubscribe Opcode = 36 // S - Unsubscribe from an event
	OpcodeSignal      Opcode = 37 // S - Emit a spectator signal
)

func (op Opcode) String() string {
	switch op {
	case OpcodeDispatch:
		return "DISPATCH"
	case OpcodeHello:
		return "HELLO"
	case OpcodeHeartbeat:
		return "HEARTBEAT"
	case OpcodeReconnect:
		return "RECONNECT"
	case OpcodeAck:
		return "ACK"
	case OpcodeError:
		return "ERROR"
	case OpcodeEndOfStream:
		return "END_OF_STREAM"

	case OpcodeIdentify:
		return "IDENTIFY"
	case OpcodeResume:
		return "RESUME"
	case OpcodeSubscribe:
		return "SUBSCRIBE"
	case OpcodeSignal:
		return "SIGNAL"
	default:
		return "UNDOCUMENTED_OPERATION"
	}
}

func (op Opcode) PublishKey() string {
	return fmt.Sprintf("events:%s:%s", "op", strings.ToLower(op.String()))
}

type CloseCode uint16

const (
	CloseCodeServerError           CloseCode = 4000 // an error occured on the server's end
	CloseCodeUnknownOperation      CloseCode = 4001 // the client sent an unexpected opcode
	CloseCodeInvalidPayload        CloseCode = 4002 // the client sent a payload that couldn't be decoded
	CloseCodeAuthFailure           CloseCode = 4003 // the client unsucessfully tried to identify
	CloseCodeAlreadyIdentified     CloseCode = 4004 // the client wanted to identify again
	CloseCodeRateLimit             CloseCode = 4005 // the client is being rate-limited
	CloseCodeRestart               CloseCode = 4006 // the server is restarting and the client should reconnect
	CloseCodeMaintenance           CloseCode = 4007 // the server is in maintenance mode and not accepting connections
	CloseCodeTimeout               CloseCode = 4008 // the client was idle for too long
	CloseCodeAlreadySubscribed     CloseCode = 4009 // the client tried to subscribe to an event twice
	CloseCodeNotSubscribed         CloseCode = 4010 // the client tried to unsubscribe from an event they weren't subscribing to
	CloseCodeInsufficientPrivilege CloseCode = 4011 // the client did something that they did not have permission for
)

func (c CloseCode) String() string {
	switch c {
	case CloseCodeServerError:
		return "Internal Server Error"
	case CloseCodeUnknownOperation:
		return "Unknown Operation"
	case CloseCodeInvalidPayload:
		return "Invalid Payload"
	case CloseCodeAuthFailure:
		return "Authentication Failed"
	case CloseCodeAlreadyIdentified:
		return "Already identified"
	case CloseCodeRateLimit:
		return "Rate limit reached"
	case CloseCodeRestart:
		return "Server is restarting"
	case CloseCodeMaintenance:
		return "Maintenance Mode"
	case CloseCodeTimeout:
		return "Timeout"
	case CloseCodeAlreadySubscribed:
		return "Already Subscribed"
	case CloseCodeNotSubscribed:
		return "Not Subscribed"
	case CloseCodeInsufficientPrivilege:
		return "Insufficient Privilege"
	default:
		return "Undocumented Closure"
	}
}
