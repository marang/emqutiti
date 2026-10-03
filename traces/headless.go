package traces

import (
	"context"
	"crypto/tls"
	"fmt"
	connections "github.com/marang/emqutiti/connections"
	"log"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

// mqttClient wraps the MQTT connection for the tracer.
type mqttClient struct{ client mqtt.Client }

// newMQTTClient establishes an MQTT connection using the provided profile.
func newMQTTClient(p connections.Profile) (*mqttClient, error) {
	opts := mqtt.NewClientOptions()
	opts.AddBroker(p.BrokerURL())
	cid := p.ClientID
	if p.RandomIDSuffix {
		cid = fmt.Sprintf("%s-%d", cid, time.Now().UnixNano())
	}
	opts.SetClientID(cid)
	opts.SetUsername(p.Username)
	if p.Password != "" {
		opts.SetPassword(p.Password)
	}
	if p.MQTTVersion != "" {
		ver, err := strconv.Atoi(p.MQTTVersion)
		if err != nil {
			return nil, fmt.Errorf("invalid MQTT version %q: %w", p.MQTTVersion, err)
		}
		if ver != 0 {
			opts.SetProtocolVersion(uint(ver))
		}
	}
	if p.ConnectTimeout > 0 {
		opts.SetConnectTimeout(time.Duration(p.ConnectTimeout) * time.Second)
	}
	if p.KeepAlive > 0 {
		opts.SetKeepAlive(time.Duration(p.KeepAlive) * time.Second)
	}
	opts.SetAutoReconnect(p.AutoReconnect)
	if p.CleanStart {
		opts.SetCleanSession(true)
	} else {
		opts.SetCleanSession(false)
	}
	if p.LastWillEnabled && p.LastWillTopic != "" {
		opts.SetWill(p.LastWillTopic, p.LastWillPayload, byte(p.LastWillQos), p.LastWillRetain)
	}
	if p.SSL {
		opts.SetTLSConfig(&tls.Config{InsecureSkipVerify: p.SkipTLSVerify})
	}
	client := mqtt.NewClient(opts)
	if token := client.Connect(); token.Wait() && token.Error() != nil {
		return nil, fmt.Errorf("failed to connect: %w", token.Error())
	}
	return &mqttClient{client: client}, nil
}

// Subscribe wraps the underlying client's Subscribe call.
func (m *mqttClient) Subscribe(topic string, qos byte, cb mqtt.MessageHandler) error {
	token := m.client.Subscribe(topic, qos, cb)
	token.Wait()
	return token.Error()
}

// Unsubscribe wraps the underlying client's Unsubscribe call.
func (m *mqttClient) Unsubscribe(topic string) error {
	token := m.client.Unsubscribe(topic)
	token.Wait()
	return token.Error()
}

// Disconnect closes the MQTT connection gracefully.
func (m *mqttClient) Disconnect() {
	if m.client != nil && m.client.IsConnected() {
		m.client.Disconnect(250)
	}
}

// Run executes the tracer headlessly using configuration from config.toml.
func Run(ctx context.Context, key, topics, profileName, startStr, endStr string) error {
	if key == "" || topics == "" {
		return fmt.Errorf("-trace and -topics are required")
	}
	var start, end time.Time
	var err error
	if startStr != "" {
		start, err = time.Parse(time.RFC3339, startStr)
		if err != nil {
			return fmt.Errorf("invalid start time: %w", err)
		}
	}
	if endStr != "" {
		end, err = time.Parse(time.RFC3339, endStr)
		if err != nil {
			return fmt.Errorf("invalid end time: %w", err)
		}
	}
	p, err := connections.LoadProfile(profileName, "")
	if err != nil {
		return err
	}
	connections.ApplyDefaultPassword(p)
	client, err := newMQTTClient(*p)
	if err != nil {
		return fmt.Errorf("connect error: %w", err)
	}
	defer client.Disconnect()

	tlist := strings.Split(topics, ",")
	for i := range tlist {
		tlist[i] = strings.TrimSpace(tlist[i])
	}
	cfg := TracerConfig{Profile: p.Name, Topics: tlist, Start: start, End: end, Key: key}
	if err := tracerClearData(cfg.Profile, cfg.Key); err != nil {
		return fmt.Errorf("clear data: %w", err)
	}
	tr := newTracer(cfg, client)
	if err := tr.Start(); err != nil {
		return fmt.Errorf("trace start: %w", err)
	}
	if err := addTrace(cfg); err != nil {
		return fmt.Errorf("add trace: %w", err)
	}

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt)
	defer signal.Stop(sig)
	var reportErr error
	var contextErr error
	recordReport := func(err error) {
		if err == nil {
			return
		}
		log.Printf("trace '%s': %v", key, err)
		if reportErr == nil {
			reportErr = fmt.Errorf("trace '%s': %w", key, err)
		}
	}
	var endCh <-chan time.Time
	if !end.IsZero() {
		timer := time.NewTimer(time.Until(end))
		defer timer.Stop()
		endCh = timer.C
	}
	// Running is false until SUBACKs arrive; only completion or an explicit
	// stop condition may end startup, including an already-expired window.
wait:
	for {
		select {
		case err := <-tr.report:
			recordReport(err)
		case <-tr.done:
			break wait
		case <-endCh:
			break wait
		case <-sig:
			break wait
		case <-ctx.Done():
			contextErr = ctx.Err()
			break wait
		}
	}

	// Stop cannot cancel a pending MQTT token wait. Give normal teardown a
	// short grace period, then disconnect to release SUBACK waits before joining.
	stopped := make(chan struct{})
	go func() {
		tr.Stop()
		close(stopped)
	}()
	select {
	case <-tr.done:
	case <-time.After(250 * time.Millisecond):
		client.Disconnect()
		<-tr.done
	}
	<-stopped
	for draining := true; draining; {
		select {
		case err := <-tr.report:
			recordReport(err)
		default:
			draining = false
		}
	}

	for t, c := range tr.Counts() {
		log.Printf("%s: %d", t, c)
	}
	if contextErr != nil {
		return contextErr
	}
	return reportErr
}
