package traces

import (
	"os"
	"sync"
	"testing"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/marang/emqutiti/proxy"
)

type fakeMessage struct {
	topic   string
	payload []byte
}

func (f fakeMessage) Duplicate() bool   { return false }
func (f fakeMessage) Qos() byte         { return 0 }
func (f fakeMessage) Retained() bool    { return false }
func (f fakeMessage) Topic() string     { return f.topic }
func (f fakeMessage) MessageID() uint16 { return 0 }
func (f fakeMessage) Payload() []byte   { return f.payload }
func (f fakeMessage) Ack()              {}

type fakeClient struct {
	subs  map[string]mqtt.MessageHandler
	subCh chan struct{}
	wg    *sync.WaitGroup
	mu    sync.RWMutex
}

func newFakeClient() *fakeClient {
	return &fakeClient{subs: make(map[string]mqtt.MessageHandler), subCh: make(chan struct{}, 1)}
}

func (f *fakeClient) Subscribe(topic string, qos byte, cb mqtt.MessageHandler) error {
	f.mu.Lock()
	f.subs[topic] = func(c mqtt.Client, m mqtt.Message) {
		cb(c, m)
		if f.wg != nil {
			f.wg.Done()
		}
	}
	f.mu.Unlock()
	select {
	case f.subCh <- struct{}{}:
	default:
	}
	return nil
}
func (f *fakeClient) Unsubscribe(topic string) error {
	f.mu.Lock()
	delete(f.subs, topic)
	f.mu.Unlock()
	return nil
}
func (f *fakeClient) Disconnect() {}

func (f *fakeClient) publish(topic, payload string) {
	f.mu.RLock()
	cb, ok := f.subs[topic]
	f.mu.RUnlock()
	if ok {
		cb(nil, fakeMessage{topic: topic, payload: []byte(payload)})
	}
}

func TestTraceStartAndStore(t *testing.T) {
	dir := t.TempDir()
	os.Setenv("HOME", dir)
	p, err := proxy.StartProxy("127.0.0.1:0")
	if err != nil {
		t.Fatalf("start proxy: %v", err)
	}
	SetProxyAddr(p.Addr())
	t.Cleanup(p.Stop)

	cfg := TracerConfig{
		Profile: "test",
		Topics:  []string{"a"},
		Start:   time.Now().Add(-time.Millisecond),
		End:     time.Now().Add(200 * time.Millisecond),
		Key:     "k1",
	}
	fc := newFakeClient()
	var wg sync.WaitGroup
	wg.Add(2)
	fc.wg = &wg
	tr := newTracer(cfg, fc)
	if err := tr.Start(); err != nil {
		t.Fatalf("start: %v", err)
	}
	<-fc.subCh
	done := tr.done
	fc.publish("a", "one")
	fc.publish("a", "two")
	wg.Wait()
	tr.Stop()
	<-done

	msgs, err := tracerMessages("test", "k1")
	if err != nil {
		t.Fatalf("messages: %v", err)
	}
	if len(msgs) != 2 {
		t.Fatalf("expected 2 messages, got %d", len(msgs))
	}
	for _, m := range msgs {
		if m.Topic != "a" {
			t.Fatalf("bad topic %s", m.Topic)
		}
	}
}
