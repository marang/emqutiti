package emqutiti

import (
	"errors"
	"fmt"
	connections "github.com/marang/emqutiti/connections"
	mqttclient "github.com/marang/emqutiti/mqttclient"
	"sync"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

const defaultTokenTimeout = 5 * time.Second

type MQTTMessage struct {
	Topic    string
	Payload  string
	Retained bool
}

type MQTTClient struct {
	Client mqtt.Client
	// MessageChan receives published messages. It closes when Disconnect is
	// called, so consumers must handle channel closure.
	MessageChan        chan MQTTMessage
	publishTimeout     time.Duration
	subscribeTimeout   time.Duration
	unsubscribeTimeout time.Duration
	done               chan struct{}
	closeOnce          sync.Once
	mu                 sync.RWMutex
}

// waitToken blocks until the MQTT token completes or the timeout expires.
// It returns any error from the token or a timeout error.
func waitToken(token mqtt.Token, timeout time.Duration, action string) error {
	if timeout <= 0 {
		timeout = defaultTokenTimeout
	}
	if !token.WaitTimeout(timeout) {
		return fmt.Errorf("%s timeout after %v", action, timeout)
	}
	if err := token.Error(); err != nil {
		return fmt.Errorf("%s failed: %w", action, err)
	}
	return nil
}

// NewMQTTClient creates and configures a new MQTT client based on the profile
// details. Status updates are delivered via the provided callback.
func NewMQTTClient(p connections.Profile, fn statusFunc) (*MQTTClient, error) {
	opts := mqtt.NewClientOptions()

	// Build option list from profile.
	optionFns := []mqttclient.ClientOption{
		mqttclient.WithBroker(p.BrokerURL()),
		mqttclient.WithClientID(p.ClientID, p.RandomIDSuffix),
		mqttclient.WithAuth(p.Username, p.Password),
		mqttclient.WithTimeouts(p.ConnectTimeout, p.KeepAlive),
		mqttclient.WithSession(p.AutoReconnect, p.CleanStart),
		mqttclient.WithWill(p.LastWillEnabled, p.LastWillTopic, p.LastWillPayload, p.LastWillQos, p.LastWillRetain),
	}

	if opt, err := mqttclient.WithVersion(p.MQTTVersion); err != nil {
		return nil, err
	} else {
		optionFns = append(optionFns, opt)
	}

	if opt, err := mqttclient.WithTLS(p.SSL, p.SkipTLSVerify, p.CACertPath, p.ClientCertPath, p.ClientKeyPath); err != nil {
		return nil, err
	} else {
		optionFns = append(optionFns, opt)
	}

	for _, opt := range optionFns {
		opt(opts)
	}

	opts.OnConnect = func(client mqtt.Client) {
		if fn != nil {
			fn("Connected to MQTT broker")
		}
	}
	opts.OnConnectionLost = func(client mqtt.Client, err error) {
		if fn != nil {
			fn(fmt.Sprintf("Connection lost: %v", err))
		}
	}

	msgChan := make(chan MQTTMessage, 20)
	done := make(chan struct{})
	mc := &MQTTClient{MessageChan: msgChan, done: done}
	opts.SetDefaultPublishHandler(func(client mqtt.Client, m mqtt.Message) {
		_ = mc.enqueueMessage(m, fn)
	})

	client := mqtt.NewClient(opts)
	mc.Client = client
	if token := client.Connect(); token.Wait() && token.Error() != nil {
		mc.Disconnect()
		return nil, fmt.Errorf("failed to connect: %w", token.Error())
	}

	pubTimeout := time.Duration(p.PublishTimeout) * time.Second
	subTimeout := time.Duration(p.SubscribeTimeout) * time.Second
	unsubTimeout := time.Duration(p.UnsubscribeTimeout) * time.Second

	mc.publishTimeout = pubTimeout
	mc.subscribeTimeout = subTimeout
	mc.unsubscribeTimeout = unsubTimeout
	return mc, nil
}

// Publish sends the payload to the given topic using the underlying client.
// It waits for the publish token to complete and returns any error from the
// broker.
func (m *MQTTClient) Publish(topic string, qos byte, retained bool, payload interface{}) error {
	token := m.Client.Publish(topic, qos, retained, payload)
	return waitToken(token, m.publishTimeout, "publish")
}

// Subscribe registers callback for messages on topic at the specified QoS.
// The method blocks until the broker acknowledges the subscription and
// returns an error if the request fails.
func (m *MQTTClient) Subscribe(topic string, qos byte, callback mqtt.MessageHandler) error {
	token := m.Client.Subscribe(topic, qos, callback)
	return waitToken(token, m.subscribeTimeout, "subscribe")
}

// Unsubscribe removes the subscription for the topic. It waits for
// completion and returns an error if the unsubscribe request fails.
func (m *MQTTClient) Unsubscribe(topic string) error {
	token := m.Client.Unsubscribe(topic)
	return waitToken(token, m.unsubscribeTimeout, "unsubscribe")
}

// Disconnect cleanly closes the connection to the broker. It also closes
// MessageChan to signal completion; consumers must handle channel closure.
func (m *MQTTClient) Disconnect() {
	if m.Client != nil && m.Client.IsConnected() {
		// Allow up to 250 milliseconds for pending work to complete.
		m.Client.Disconnect(250)
	}
	// Close MessageChan after disconnecting to stop message delivery.
	m.closeOnce.Do(func() {
		m.mu.Lock()
		defer m.mu.Unlock()
		if m.done != nil {
			close(m.done)
		}
		if m.MessageChan != nil {
			close(m.MessageChan)
			m.MessageChan = nil
		}
	})
}

func (m *MQTTClient) safeMessageChan() chan MQTTMessage {
	if m == nil {
		return nil
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.MessageChan
}

func (m *MQTTClient) enqueueMessage(msg mqtt.Message, fn statusFunc) error {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if m.MessageChan == nil {
		return errors.New("message channel is closed")
	}
	out := MQTTMessage{Topic: msg.Topic(), Payload: string(msg.Payload()), Retained: msg.Retained()}
	select {
	case <-m.done:
		return errors.New("message channel is closed")
	case m.MessageChan <- out:
		return nil
	default:
		if fn != nil {
			fn(fmt.Sprintf("Dropped MQTT message on %s: message buffer full", msg.Topic()))
		}
		return nil
	}
}
