package traces

import (
	"fmt"
	"net"
	"strconv"
	"testing"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	connections "github.com/marang/emqutiti/connections"
	"github.com/marang/emqutiti/proxy"
	"github.com/mochi-co/mqtt/server"
	"github.com/mochi-co/mqtt/server/listeners"
)

func TestTraceWithLiveBroker(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	dir := t.TempDir()
	t.Setenv("HOME", dir)

	prevProxy := proxyAddr
	p, err := proxy.StartProxy("127.0.0.1:0")
	if err != nil {
		t.Fatalf("start proxy: %v", err)
	}
	SetProxyAddr(p.Addr())
	t.Cleanup(func() {
		p.Stop()
		SetProxyAddr(prevProxy)
	})

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("reserve port: %v", err)
	}
	addr := ln.Addr().String()
	if err := ln.Close(); err != nil {
		t.Fatalf("close temp listener: %v", err)
	}

	broker := server.New()
	tcp := listeners.NewTCP("test", addr)
	if err := broker.AddListener(tcp, nil); err != nil {
		t.Fatalf("add listener: %v", err)
	}
	if err := broker.Serve(); err != nil {
		t.Fatalf("serve broker: %v", err)
	}
	t.Cleanup(func() {
		if err := broker.Close(); err != nil {
			t.Logf("close broker: %v", err)
		}
	})

	host, portStr, err := net.SplitHostPort(addr)
	if err != nil {
		t.Fatalf("split host port: %v", err)
	}
	port, err := strconv.Atoi(portStr)
	if err != nil {
		t.Fatalf("parse port: %v", err)
	}

	profile := connections.Profile{
		Name:       "integration",
		Schema:     "tcp",
		Host:       host,
		Port:       port,
		ClientID:   fmt.Sprintf("trace-client-%d", time.Now().UnixNano()),
		CleanStart: true,
	}

	client, err := newMQTTClient(profile)
	if err != nil {
		t.Fatalf("new mqtt client: %v", err)
	}
	defer client.Disconnect()

	key := fmt.Sprintf("trace-%d", time.Now().UnixNano())
	topic := "test/trace"
	cfg := TracerConfig{
		Profile: profile.Name,
		Topics:  []string{topic},
		Start:   time.Now().Add(-100 * time.Millisecond),
		Key:     key,
	}
	if err := tracerClearData(cfg.Profile, cfg.Key); err != nil {
		t.Fatalf("clear data: %v", err)
	}

	tracer := newTracer(cfg, client)
	if err := tracer.Start(); err != nil {
		t.Fatalf("start trace: %v", err)
	}
	t.Cleanup(tracer.Stop)

	deadline := time.Now().Add(5 * time.Second)
	for !tracer.Running() {
		if time.Now().After(deadline) {
			t.Fatalf("tracer never started")
		}
		time.Sleep(10 * time.Millisecond)
	}

	pubOpts := mqtt.NewClientOptions()
	pubOpts.AddBroker(profile.BrokerURL())
	pubOpts.SetClientID(fmt.Sprintf("publisher-%d", time.Now().UnixNano()))
	pubOpts.SetConnectTimeout(2 * time.Second)
	publisher := mqtt.NewClient(pubOpts)
	if token := publisher.Connect(); token.Wait() && token.Error() != nil {
		t.Fatalf("publisher connect: %v", token.Error())
	}
	defer publisher.Disconnect(250)

	payload := "integration payload"
	token := publisher.Publish(topic, 0, false, payload)
	token.Wait()
	if err := token.Error(); err != nil {
		t.Fatalf("publish: %v", err)
	}

	waitDeadline := time.Now().Add(5 * time.Second)
	for {
		counts := tracer.Counts()
		if counts[topic] > 0 {
			break
		}
		if time.Now().After(waitDeadline) {
			t.Fatalf("no messages captured")
		}
		time.Sleep(20 * time.Millisecond)
	}

	tracer.Stop()

	msgs, err := tracer.Messages()
	if err != nil {
		t.Fatalf("messages: %v", err)
	}
	if len(msgs) == 0 {
		t.Fatalf("expected at least one message")
	}
	found := false
	for _, m := range msgs {
		if m.Topic == topic && m.Payload == payload {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected payload %q on topic %q", payload, topic)
	}
}
