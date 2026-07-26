package dji

import (
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
)

var (
	ErrInvalidJSON  = errors.New("invalid DJI envelope JSON")
	ErrMissingData  = errors.New("DJI envelope requires data")
	ErrInvalidField = errors.New("invalid DJI envelope field")
)

type Envelope struct {
	TID       string
	BID       string
	Timestamp int64
	Gateway   string
	Method    string
	NeedReply *bool
	Seq       *int64
	Data      json.RawMessage
	Unknown   map[string]json.RawMessage
}

type Message struct {
	Topic       Topic
	Envelope    Envelope
	PayloadHash [sha256.Size]byte
}

func DecodeEnvelope(payload []byte) (Envelope, error) {
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(payload, &fields); err != nil {
		return Envelope{}, fmt.Errorf("%w: %v", ErrInvalidJSON, err)
	}
	data, ok := fields["data"]
	if !ok {
		return Envelope{}, ErrMissingData
	}
	envelope := Envelope{Data: append(json.RawMessage(nil), data...), Unknown: make(map[string]json.RawMessage)}
	for key, value := range fields {
		switch key {
		case "tid":
			if err := json.Unmarshal(value, &envelope.TID); err != nil {
				return Envelope{}, fmt.Errorf("%w: tid", ErrInvalidField)
			}
		case "bid":
			if err := json.Unmarshal(value, &envelope.BID); err != nil {
				return Envelope{}, fmt.Errorf("%w: bid", ErrInvalidField)
			}
		case "timestamp":
			if err := json.Unmarshal(value, &envelope.Timestamp); err != nil {
				return Envelope{}, fmt.Errorf("%w: timestamp", ErrInvalidField)
			}
		case "gateway":
			if err := json.Unmarshal(value, &envelope.Gateway); err != nil {
				return Envelope{}, fmt.Errorf("%w: gateway", ErrInvalidField)
			}
		case "method":
			if err := json.Unmarshal(value, &envelope.Method); err != nil {
				return Envelope{}, fmt.Errorf("%w: method", ErrInvalidField)
			}
		case "need_reply":
			needReply, err := decodeNeedReply(value)
			if err != nil {
				return Envelope{}, err
			}
			envelope.NeedReply = &needReply
		case "seq":
			var seq int64
			if err := json.Unmarshal(value, &seq); err != nil {
				return Envelope{}, fmt.Errorf("%w: seq", ErrInvalidField)
			}
			envelope.Seq = &seq
		case "data":
			continue
		default:
			envelope.Unknown[key] = append(json.RawMessage(nil), value...)
		}
	}
	return envelope, nil
}

func ParseMessage(topic string, payload []byte) (Message, error) {
	parsedTopic, err := ParseTopic(topic)
	if err != nil {
		return Message{}, err
	}
	envelope, err := DecodeEnvelope(payload)
	if err != nil {
		return Message{}, err
	}
	return Message{
		Topic:       parsedTopic,
		Envelope:    envelope,
		PayloadHash: sha256.Sum256(payload),
	}, nil
}

func decodeNeedReply(value json.RawMessage) (bool, error) {
	var boolean bool
	if err := json.Unmarshal(value, &boolean); err == nil {
		return boolean, nil
	}
	var integer int
	if err := json.Unmarshal(value, &integer); err == nil && (integer == 0 || integer == 1) {
		return integer == 1, nil
	}
	return false, fmt.Errorf("%w: need_reply must be boolean or 0/1", ErrInvalidField)
}
