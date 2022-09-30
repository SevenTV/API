package events

import (
	"encoding/json"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type AnyPayload interface {
	json.RawMessage | HelloPayload | AckPayload | HeartbeatPayload |
		SubscribePayload | UnsubscribePayload | DispatchPayload | SignalPayload |
		ErrorPayload | EndOfStreamPayload
}

type HelloPayload struct {
	HeartbeatInterval int64               `json:"heartbeat_interval"`
	SessionID         string              `json:"session_id"`
	Actor             *primitive.ObjectID `json:"actor,omitempty"`
}

type AckPayload struct {
	RequestID string         `json:"request_id"`
	Data      map[string]any `json:"data"`
}

type HeartbeatPayload struct {
	Count int64 `json:"count"`
}

type SubscribePayload struct {
	Type      EventType         `json:"type"`
	Condition map[string]string `json:"condition"`
}

type UnsubscribePayload struct {
	Type      EventType         `json:"type"`
	Condition map[string]string `json:"condition"`
}

type DispatchPayload struct {
	Type EventType `json:"type"`
	// Detailed changes to an object
	Body ChangeMap `json:"body"`
	// A map of conditions that must match subscriptions in order for this dispatch to be delivered
	Condition EventCondition `json:"condition,omitempty"`
}

type EventCondition map[string]string

func (evc EventCondition) SetObjectID(id primitive.ObjectID) EventCondition {
	evc["object_id"] = id.Hex()

	return evc
}

func (evc EventCondition) Set(key string, value string) EventCondition {
	evc[key] = value

	return evc
}

type SignalPayload struct {
	Sender SignalUser `json:"sender"`
	Host   SignalUser `json:"host"`
}

type SignalUser struct {
	ID          primitive.ObjectID `json:"id"`
	ChannelID   string             `json:"channel_id"`
	Username    string             `json:"username"`
	DisplayName string             `json:"display_name"`
}

type ErrorPayload struct {
	Message       string         `json:"message"`
	MessageLocale string         `json:"message_locale,omitempty"`
	Fields        map[string]any `json:"fields"`
}

type EndOfStreamPayload struct {
	Code    CloseCode `json:"code"`
	Message string    `json:"message"`
}
