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
	HeartbeatInterval uint32              `json:"heartbeat_interval"`
	SessionID         string              `json:"session_id"`
	SubscriptionLimit int32               `json:"subscription_limit"`
	Actor             *primitive.ObjectID `json:"actor,omitempty"`
}

type AckPayload struct {
	Command string          `json:"command"`
	Data    json.RawMessage `json:"data"`
}

type HeartbeatPayload struct {
	Count uint64 `json:"count"`
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
	// Hash is a hash of the target object, used for deduping
	Hash *uint32 `json:"hash,omitempty"`
	// A list of effects to apply to the session when this dispatch is received
	Effect *SessionEffect `json:"effect,omitempty"`
	// A list of subscriptions that match this dispatch
	Matches []uint32 `json:"matches,omitempty"`
	// A list of conditions where at least one must have all its fields match a subscription in order for this dispatch to be delivered
	Conditions []EventCondition `json:"condition,omitempty"`
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

func (evc EventCondition) Match(other EventCondition) bool {
	for k, v := range evc {
		if other[k] != v {
			return false
		}
	}

	return true
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
