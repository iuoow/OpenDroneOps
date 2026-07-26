package dji

import (
	"errors"
	"testing"
)

func TestParseTopicTable(t *testing.T) {
	tests := []struct {
		name      string
		raw       string
		kind      TopicKind
		direction Direction
		deviceSN  string
		gatewaySN string
	}{
		{"osd", "thing/product/AIR-1/osd", TopicOSD, DirectionDeviceToCloud, "AIR-1", ""},
		{"state", "thing/product/AIR-1/state", TopicState, DirectionDeviceToCloud, "AIR-1", ""},
		{"services", "thing/product/DOCK-1/services", TopicServices, DirectionCloudToDevice, "", "DOCK-1"},
		{"services reply", "thing/product/DOCK-1/services_reply", TopicServicesReply, DirectionDeviceToCloud, "", "DOCK-1"},
		{"events", "thing/product/DOCK-1/events", TopicEvents, DirectionDeviceToCloud, "", "DOCK-1"},
		{"events reply", "thing/product/DOCK-1/events_reply", TopicEventsReply, DirectionCloudToDevice, "", "DOCK-1"},
		{"requests", "thing/product/DOCK-1/requests", TopicRequests, DirectionDeviceToCloud, "", "DOCK-1"},
		{"requests reply", "thing/product/DOCK-1/requests_reply", TopicRequestsReply, DirectionCloudToDevice, "", "DOCK-1"},
		{"status", "sys/product/DOCK-1/status", TopicStatus, DirectionDeviceToCloud, "", "DOCK-1"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := ParseTopic(test.raw)
			if err != nil {
				t.Fatalf("ParseTopic() error = %v", err)
			}
			if got.Kind != test.kind || got.Direction != test.direction || got.DeviceSN != test.deviceSN || got.GatewaySN != test.gatewaySN {
				t.Fatalf("ParseTopic() = %+v", got)
			}
		})
	}
}

func TestParseTopicRejectsMalformedAndDRC(t *testing.T) {
	for _, raw := range []string{"", "thing/product//osd", "thing/product/AIR-1/#", "thing/product/AIR-1/drc/down", "sys/product/DOCK-1/online"} {
		_, err := ParseTopic(raw)
		if err == nil {
			t.Fatalf("ParseTopic(%q) unexpectedly succeeded", raw)
		}
		if !errors.Is(err, ErrMalformedTopic) && !errors.Is(err, ErrUnsupportedTopic) {
			t.Fatalf("ParseTopic(%q) error = %v", raw, err)
		}
	}
}
