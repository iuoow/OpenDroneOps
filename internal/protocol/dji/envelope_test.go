package dji

import (
	"errors"
	"testing"
)

func TestDecodeEnvelopePreservesUnknownFields(t *testing.T) {
	payload := []byte(`{"tid":"tid-1","bid":"bid-1","timestamp":1700000000000,"gateway":"DOCK-1","method":"thing/osd","need_reply":1,"seq":7,"data":{"latitude":31.2},"future_field":{"x":true}}`)
	envelope, err := DecodeEnvelope(payload)
	if err != nil {
		t.Fatalf("DecodeEnvelope() error = %v", err)
	}
	if envelope.TID != "tid-1" || envelope.BID != "bid-1" || envelope.Timestamp != 1700000000000 || envelope.Method != "thing/osd" {
		t.Fatalf("decoded envelope has wrong fields: %+v", envelope)
	}
	if envelope.NeedReply == nil || !*envelope.NeedReply || envelope.Seq == nil || *envelope.Seq != 7 {
		t.Fatalf("decoded optional fields are wrong: %+v", envelope)
	}
	if string(envelope.Data) != `{"latitude":31.2}` || string(envelope.Unknown["future_field"]) != `{"x":true}` {
		t.Fatalf("payload preservation failed: data=%s unknown=%s", envelope.Data, envelope.Unknown["future_field"])
	}
}

func TestDecodeEnvelopeAcceptsBooleanNeedReplyAndRejectsInvalidMessages(t *testing.T) {
	envelope, err := DecodeEnvelope([]byte(`{"need_reply":false,"data":[]}`))
	if err != nil || envelope.NeedReply == nil || *envelope.NeedReply {
		t.Fatalf("boolean need_reply failed: envelope=%+v err=%v", envelope, err)
	}
	for _, payload := range []string{`{"tid":"x"}`, `{"data":{}]`, `{"need_reply":2,"data":{}}`} {
		_, err := DecodeEnvelope([]byte(payload))
		if err == nil {
			t.Fatalf("DecodeEnvelope(%s) unexpectedly succeeded", payload)
		}
		if !errors.Is(err, ErrMissingData) && !errors.Is(err, ErrInvalidJSON) && !errors.Is(err, ErrInvalidField) {
			t.Fatalf("DecodeEnvelope(%s) error = %v", payload, err)
		}
	}
}

func TestParseMessageIncludesPayloadHash(t *testing.T) {
	message, err := ParseMessage("thing/product/AIR-1/osd", []byte(`{"data":{"battery":99}}`))
	if err != nil {
		t.Fatalf("ParseMessage() error = %v", err)
	}
	if message.Topic.Kind != TopicOSD || message.PayloadHash == [32]byte{} {
		t.Fatalf("message metadata missing: %+v", message)
	}
}
