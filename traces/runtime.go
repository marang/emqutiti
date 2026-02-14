package traces

import (
	"context"
	"fmt"
	"sync"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/marang/emqutiti/proxy"
)

// Config defines the trace parameters.
type TracerConfig struct {
	Profile string
	Topics  []string
	Start   time.Time
	End     time.Time
	Key     string
}

// Client abstracts the MQTT client used by the tracer.
type Client interface {
	Subscribe(topic string, qos byte, cb mqtt.MessageHandler) error
	Unsubscribe(topic string) error
	Disconnect()
}

// Tracer collects MQTT messages within a time range and stores them.
type Tracer struct {
	cfg     TracerConfig
	mu      sync.Mutex
	running bool
	ready   bool
	counts  map[string]int
	client  Client
	cancel  context.CancelFunc
	done    chan struct{}
	report  chan error
}

// newTracer creates a new Tracer with the given config.
func newTracer(cfg TracerConfig, c Client) *Tracer {
	return &Tracer{cfg: cfg, client: c, report: make(chan error, 1)}
}

func (t *Tracer) reportErr(err error) {
	if t.report != nil {
		select {
		case t.report <- err:
		default:
		}
	}
}

// Start begins the trace.
func (t *Tracer) Start() error {
	t.mu.Lock()
	if t.running {
		t.mu.Unlock()
		return fmt.Errorf("trace already running")
	}
	t.running = true
	t.ready = false
	t.counts = make(map[string]int)
	for _, tp := range t.cfg.Topics {
		t.counts[tp] = 0
	}
	t.mu.Unlock()

	client := t.client

	cl, conn, err := proxy.NewClient(addr())
	if err != nil {
		t.mu.Lock()
		t.running = false
		t.mu.Unlock()
		return err
	}

	ctx, cancel := context.WithCancel(context.Background())
	t.cancel = cancel
	t.done = make(chan struct{})

	go func() {
		defer func() {
			client.Disconnect()
			conn.Close()
			cancel()
			t.mu.Lock()
			t.running = false
			t.ready = false
			t.cancel = nil
			t.mu.Unlock()
			close(t.done)
		}()

		delay := time.Until(t.cfg.Start)
		if delay > 0 {
			select {
			case <-time.After(delay):
			case <-ctx.Done():
				return
			}
		}

		endCh := make(<-chan time.Time)
		var timer *time.Timer
		if !t.cfg.End.IsZero() {
			if d := time.Until(t.cfg.End); d > 0 {
				timer = time.NewTimer(d)
				endCh = timer.C
			}
		}
		if timer != nil {
			defer timer.Stop()
		}

		for _, topic := range t.cfg.Topics {
			if err := client.Subscribe(topic, 0, func(_ mqtt.Client, m mqtt.Message) {
				ts := time.Now()
				if !t.cfg.End.IsZero() && ts.After(t.cfg.End) {
					return
				}
				if ts.Before(t.cfg.Start) {
					return
				}
				if err := tracerAddClient(cl, t.cfg.Profile, t.cfg.Key, TracerMessage{Timestamp: ts, Topic: m.Topic(), Payload: string(m.Payload()), Kind: "trace", Retained: m.Retained()}); err != nil {
					t.reportErr(fmt.Errorf("tracerAdd: %w", err))
					return
				}
				t.mu.Lock()
				for _, sub := range t.cfg.Topics {
					if tracerMatch(sub, m.Topic()) {
						t.counts[sub]++
					}
				}
				t.mu.Unlock()
			}); err != nil {
				fmt.Printf("subscribe %s: %v\n", topic, err)
				return
			}
		}

		t.mu.Lock()
		t.ready = true
		t.mu.Unlock()

		for {
			select {
			case <-ctx.Done():
				return
			case <-endCh:
				return
			}
		}
	}()

	return nil
}

// Stop terminates the trace.
func (t *Tracer) Stop() {
	t.mu.Lock()
	if !t.running {
		t.mu.Unlock()
		return
	}
	t.running = false
	t.ready = false
	if t.cancel != nil {
		t.cancel()
	}
	done := t.done
	t.mu.Unlock()
	if done != nil {
		<-done
	}
}

// Running reports whether the trace is currently active.
func (t *Tracer) Running() bool {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.running && t.ready && time.Now().After(t.cfg.Start) && (t.cfg.End.IsZero() || time.Now().Before(t.cfg.End))
}

// Planned reports whether the trace start time is in the future.
func (t *Tracer) Planned() bool { return time.Now().Before(t.cfg.Start) }

// Config returns the trace configuration.
func (t *Tracer) Config() TracerConfig { return t.cfg }

// Counts returns the per-topic message counts.
func (t *Tracer) Counts() map[string]int {
	t.mu.Lock()
	defer t.mu.Unlock()
	out := make(map[string]int, len(t.counts))
	for k, v := range t.counts {
		out[k] = v
	}
	return out
}

// Messages returns the stored trace messages.
func (t *Tracer) Messages() ([]TracerMessage, error) {
	return tracerMessages(t.cfg.Profile, t.cfg.Key)
}
