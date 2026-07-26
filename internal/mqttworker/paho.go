package mqttworker

import (
	"context"
	"crypto/tls"
	"errors"
	"net/url"
	"time"

	"github.com/eclipse/paho.golang/autopaho"
	"github.com/eclipse/paho.golang/paho"
)

type BrokerConfig struct {
	URL            string
	ClientID       string
	Username       string
	Password       string
	KeepAlive      time.Duration
	SessionExpiry  time.Duration
	CleanStart     bool
	TLSConfig      *tls.Config
	OnConnectError func(error)
}

type MQTTBroker struct {
	manager *autopaho.ConnectionManager
}

func ConnectBroker(ctx context.Context, config BrokerConfig, onMessage func(RawMessage)) (*MQTTBroker, error) {
	if ctx == nil || onMessage == nil {
		return nil, errors.New("MQTT broker context and message callback are required")
	}
	if config.URL == "" || config.ClientID == "" {
		return nil, errors.New("MQTT broker URL and client ID are required")
	}
	serverURL, err := url.Parse(config.URL)
	if err != nil {
		return nil, err
	}
	keepAlive := uint16(config.KeepAlive / time.Second)
	if keepAlive == 0 {
		keepAlive = 30
	}
	sessionExpiry := uint32(config.SessionExpiry / time.Second)
	clientConfig := paho.ClientConfig{
		ClientID: config.ClientID,
		OnPublishReceived: []func(paho.PublishReceived) (bool, error){
			func(received paho.PublishReceived) (bool, error) {
				packet := received.Packet
				if packet == nil {
					return false, nil
				}
				onMessage(RawMessage{
					Topic: packet.Topic, Payload: append([]byte(nil), packet.Payload...),
					QoS: packet.QoS, Retain: packet.Retain, ReceivedAt: time.Now().UTC(),
				})
				return false, nil
			},
		},
	}
	brokerConfig := autopaho.ClientConfig{
		ServerUrls:                    []*url.URL{serverURL},
		TlsCfg:                        config.TLSConfig,
		KeepAlive:                     keepAlive,
		CleanStartOnInitialConnection: config.CleanStart,
		SessionExpiryInterval:         sessionExpiry,
		ConnectUsername:               config.Username,
		ConnectPassword:               []byte(config.Password),
		OnConnectionUp: func(manager *autopaho.ConnectionManager, _ *paho.Connack) {
			go subscribeInbound(ctx, manager)
		},
		OnConnectError: config.OnConnectError,
		ClientConfig:   clientConfig,
	}
	manager, err := autopaho.NewConnection(ctx, brokerConfig)
	if err != nil {
		return nil, err
	}
	return &MQTTBroker{manager: manager}, nil
}

func (b *MQTTBroker) Close(ctx context.Context) error {
	if b == nil || b.manager == nil {
		return nil
	}
	return b.manager.Disconnect(ctx)
}

func subscribeInbound(ctx context.Context, manager *autopaho.ConnectionManager) {
	_, _ = manager.Subscribe(ctx, &paho.Subscribe{Subscriptions: []paho.SubscribeOptions{
		{Topic: "thing/product/+/osd", QoS: 1},
		{Topic: "thing/product/+/state", QoS: 1},
		{Topic: "thing/product/+/services_reply", QoS: 1},
		{Topic: "thing/product/+/events", QoS: 1},
		{Topic: "thing/product/+/requests", QoS: 1},
		{Topic: "sys/product/+/status", QoS: 1},
	}})
}
