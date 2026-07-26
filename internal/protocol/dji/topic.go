package dji

import (
	"errors"
	"fmt"
	"strings"
)

type TopicKind string

const (
	TopicOSD           TopicKind = "OSD"
	TopicState         TopicKind = "STATE"
	TopicServices      TopicKind = "SERVICES"
	TopicServicesReply TopicKind = "SERVICES_REPLY"
	TopicEvents        TopicKind = "EVENTS"
	TopicEventsReply   TopicKind = "EVENTS_REPLY"
	TopicRequests      TopicKind = "REQUESTS"
	TopicRequestsReply TopicKind = "REQUESTS_REPLY"
	TopicStatus        TopicKind = "STATUS"
)

type Direction string

const (
	DirectionDeviceToCloud Direction = "DEVICE_TO_CLOUD"
	DirectionCloudToDevice Direction = "CLOUD_TO_DEVICE"
)

var (
	ErrMalformedTopic   = errors.New("malformed DJI topic")
	ErrUnsupportedTopic = errors.New("unsupported DJI topic")
)

type Topic struct {
	Raw       string
	Kind      TopicKind
	DeviceSN  string
	GatewaySN string
	Direction Direction
}

func ParseTopic(raw string) (Topic, error) {
	if raw == "" || strings.ContainsAny(raw, "#+") {
		return Topic{}, fmt.Errorf("%w: %q", ErrMalformedTopic, raw)
	}
	parts := strings.Split(raw, "/")
	if len(parts) != 4 || parts[0] != "thing" && parts[0] != "sys" {
		return Topic{}, fmt.Errorf("%w: %q", ErrMalformedTopic, raw)
	}
	if parts[0] == "thing" {
		if parts[1] != "product" || parts[2] == "" || parts[3] == "" {
			return Topic{}, fmt.Errorf("%w: %q", ErrMalformedTopic, raw)
		}
		topic := Topic{Raw: raw}
		switch parts[3] {
		case "osd":
			topic.Kind, topic.DeviceSN, topic.Direction = TopicOSD, parts[2], DirectionDeviceToCloud
		case "state":
			topic.Kind, topic.DeviceSN, topic.Direction = TopicState, parts[2], DirectionDeviceToCloud
		case "services":
			topic.Kind, topic.GatewaySN, topic.Direction = TopicServices, parts[2], DirectionCloudToDevice
		case "services_reply":
			topic.Kind, topic.GatewaySN, topic.Direction = TopicServicesReply, parts[2], DirectionDeviceToCloud
		case "events":
			topic.Kind, topic.GatewaySN, topic.Direction = TopicEvents, parts[2], DirectionDeviceToCloud
		case "events_reply":
			topic.Kind, topic.GatewaySN, topic.Direction = TopicEventsReply, parts[2], DirectionCloudToDevice
		case "requests":
			topic.Kind, topic.GatewaySN, topic.Direction = TopicRequests, parts[2], DirectionDeviceToCloud
		case "requests_reply":
			topic.Kind, topic.GatewaySN, topic.Direction = TopicRequestsReply, parts[2], DirectionCloudToDevice
		default:
			return Topic{}, fmt.Errorf("%w: %q", ErrUnsupportedTopic, raw)
		}
		return topic, nil
	}
	if parts[1] != "product" || parts[2] == "" || parts[3] != "status" {
		return Topic{}, fmt.Errorf("%w: %q", ErrMalformedTopic, raw)
	}
	return Topic{
		Raw: raw, Kind: TopicStatus, GatewaySN: parts[2], Direction: DirectionDeviceToCloud,
	}, nil
}
