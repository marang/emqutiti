package emqutiti

import (
	"context"
	"io"
	"net"
	"slices"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/marang/emqutiti/constants"
	"github.com/marang/emqutiti/topics"
	"github.com/mochi-co/mqtt/server"
	"github.com/mochi-co/mqtt/server/events"
	"github.com/mochi-co/mqtt/server/listeners"
)

const retainedClearTimeout = 3 * time.Second

type retainedClearPacket struct {
	MQTTMessage
	qos byte
}

func TestRetainedClearIntegration(t *testing.T) {
	for _, test := range []struct {
		name, enter string
		retained    bool
		marked      bool
	}{
		{"retained_csi_u_selected", "\x1b[13;6u", true, false},
		{"retained_modify_other_keys_marked", "\x1b[27;6;13~", true, true},
		{"normal_csi_u_selected_preserves_retained", "\x1b[13;5u", false, false},
		{"normal_modify_other_keys_marked_preserves_retained", "\x1b[27;5;13~", false, true},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Setenv("HOME", t.TempDir())
			t.Setenv("XDG_CONFIG_HOME", t.TempDir())
			previousImport := importFile
			importFile = ""
			t.Cleanup(func() { importFile = previousImport })
			m := reviewFixture(t, 80, 24)
			m.SetMode(constants.ModeClient)
			m.SetFocus(idMessage)
			m.history.SetItems(nil)

			const prefix = "retained-clear"
			const publisherID = "retained-clear-editor"
			const seed = "stored\nmessage"
			const guardPayload = "leave this retained message alone"
			first, second, guard := prefix+"/first", prefix+"/second", prefix+"/guard"
			targets := []string{first}
			m.topics.Items = []topics.Item{
				{Name: first, Publish: test.marked},
				{Name: second, Publish: test.marked},
				{Name: guard, Subscribed: true},
			}
			m.topics.RebuildActiveTopicList()
			m.topics.SetSelected(0)
			if test.marked {
				targets = append(targets, second)
				m.topics.SetSelected(2) // A selected non-target must not join marked targets.
			}
			if got := m.publishTargets(); !slices.Equal(got, targets) {
				t.Fatalf("publish targets = %q, want %q", got, targets)
			}

			received := make(chan retainedClearPacket, 16)
			broker, addr := retainedClearBroker(t, func(cl events.Client, pk events.Packet) (events.Packet, error) {
				// OnMessage runs after retention is evaluated; copy the actual inbound packet.
				if cl.ID == publisherID {
					received <- retainedClearPacket{
						MQTTMessage: MQTTMessage{Topic: pk.TopicName, Payload: string(pk.Payload), Retained: pk.FixedHeader.Retain},
						qos:         pk.FixedHeader.Qos,
					}
				}
				return pk, nil
			})
			publisher := retainedClearClient(t, addr, publisherID)
			m.mqttClient = &MQTTClient{Client: publisher, publishTimeout: retainedClearTimeout}
			m.connections.Active = publisherID
			if err := broker.Publish(guard, []byte(guardPayload), true); err != nil {
				t.Fatal(err)
			}

			m.message.SetPayload(seed)
			retainedClearInput(t, m, "\x1b[13;6u", seed, true, targets, 0)
			retainedClearPackets(t, m.mqttClient, received, prefix, "seed", seed, true, targets)
			before := map[string]string{guard: guardPayload}
			for _, topic := range targets {
				before[topic] = seed
			}
			retainedClearState(t, broker, prefix, before)
			retainedClearReplay(t, addr, prefix, "before", before)

			// Feed ordinary DEL/Backspace bytes through the real parser and root editor.
			retainedClearInput(t, m, strings.Repeat("\x7f", len(seed))+test.enter, "", test.retained, targets, len(seed))
			retainedClearPackets(t, m.mqttClient, received, prefix, "empty", "", test.retained, targets)
			after := map[string]string{guard: guardPayload}
			if !test.retained {
				for _, topic := range targets {
					after[topic] = seed
				}
			}
			retainedClearState(t, broker, prefix, after)
			retainedClearReplay(t, addr, prefix, "after", after)

			items := m.history.Items()
			if len(items) != 2*len(targets) {
				t.Fatalf("publish history has %d entries, want %d: %+v", len(items), 2*len(targets), items)
			}
			for i, item := range items {
				payload, retained := seed, true
				if i >= len(targets) {
					payload, retained = "", test.retained
				}
				if item.Kind != "pub" || item.Topic != targets[i%len(targets)] || item.Payload != payload || item.Retained != retained {
					t.Fatalf("publish history entry %d = %+v", i, item)
				}
			}
		})
	}
}

func retainedClearBroker(t *testing.T, onMessage events.OnMessage) (*server.Server, string) {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	addr := ln.Addr().String()
	if err := ln.Close(); err != nil {
		t.Fatal(err)
	}
	broker := server.New()
	broker.Events.OnMessage = onMessage
	t.Cleanup(func() {
		if err := broker.Close(); err != nil {
			t.Errorf("close isolated broker: %v", err)
		}
	})
	// Binding must succeed before any client is created; never dial a fallback broker.
	if err := broker.AddListener(listeners.NewTCP("retained-clear-test", addr), nil); err != nil {
		t.Fatal(err)
	}
	if err := broker.Serve(); err != nil {
		t.Fatal(err)
	}
	return broker, "tcp://" + addr
}

func retainedClearClient(t *testing.T, addr, id string) mqtt.Client {
	t.Helper()
	opts := mqtt.NewClientOptions().AddBroker(addr).SetClientID(id).
		SetProtocolVersion(4).SetCleanSession(true).SetAutoReconnect(false).
		SetConnectRetry(false).SetConnectTimeout(retainedClearTimeout).
		SetWriteTimeout(retainedClearTimeout).SetOrderMatters(true).
		SetStore(mqtt.NewMemoryStore())
	client := mqtt.NewClient(opts)
	t.Cleanup(func() { client.Disconnect(250) })
	retainedClearToken(t, client.Connect(), "connect "+id)
	return client
}

func retainedClearToken(t *testing.T, token mqtt.Token, action string) {
	t.Helper()
	if !token.WaitTimeout(retainedClearTimeout) {
		t.Fatalf("%s timed out", action)
	}
	if err := token.Error(); err != nil {
		t.Fatalf("%s: %v", action, err)
	}
}

func retainedClearInput(t *testing.T, m *model, raw, payload string, retained bool, targets []string, backspaces int) {
	t.Helper()
	before := m.message.Input().Value()
	var deletions, dispatched, drafts []string
	var presses []ctrlEnterMsg
	harness := &ctrlEnterCaptureModel{apply: func(msg tea.Msg) tea.Cmd {
		_, cmd := m.Update(msg)
		if key, ok := msg.(tea.KeyMsg); ok && key.Type == tea.KeyBackspace && !key.Alt && !key.Paste {
			deletions = append(deletions, m.message.Input().Value())
		}
		if press, ok := msg.(ctrlEnterMsg); ok {
			presses = append(presses, press)
			dispatched = append(dispatched, m.pendingPublishTargets()...)
			drafts = append(drafts, m.message.Input().Value())
			applyMQTTCommand(m, cmd) // Execute real Paho commands and apply typed results through root.Update.
		}
		return nil
	}}
	r := framingCtrlEnterReader(strings.NewReader(raw + "\x04"))
	defer r.close()
	ctx, cancel := context.WithTimeout(context.Background(), 3*retainedClearTimeout)
	defer cancel()
	// The harness does not run root.Init or root's save-on-quit path, so demo profiles stay disconnected.
	p := tea.NewProgram(harness, tea.WithInput(r), tea.WithFilter(r.filter), tea.WithOutput(io.Discard),
		tea.WithContext(ctx), tea.WithoutRenderer(), tea.WithoutSignalHandler())
	if _, err := p.Run(); err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(presses, []ctrlEnterMsg{{press: true, retained: retained}}) || !slices.Equal(dispatched, targets) || !slices.Equal(drafts, []string{payload}) {
		t.Fatalf("publish input: presses=%+v targets=%q drafts=%q", presses, dispatched, drafts)
	}
	if len(deletions) != backspaces {
		t.Fatalf("ordinary Backspace events = %d, want %d", len(deletions), backspaces)
	}
	for i, value := range deletions {
		if want := before[:len(before)-i-1]; value != want {
			t.Fatalf("draft after Backspace %d = %q, want %q", i+1, value, want)
		}
	}
	if m.message.Input().Value() != payload || m.pendingPublishes() != 0 || m.mqttOps.publishError != "" {
		t.Fatalf("publish completion: draft=%q pending=%d error=%q", m.message.Input().Value(), m.pendingPublishes(), m.mqttOps.publishError)
	}
}

func retainedClearPackets(t *testing.T, client *MQTTClient, received <-chan retainedClearPacket, prefix, phase, payload string, retained bool, targets []string) {
	t.Helper()
	marker := prefix + "/publish-barrier"
	if err := client.Publish(marker, 1, false, phase); err != nil {
		t.Fatal(err)
	}
	deadline := time.NewTimer(retainedClearTimeout)
	defer deadline.Stop()
	seen := make(map[string]bool)
	for {
		select {
		case packet := <-received:
			if packet.Topic == marker {
				if packet.Payload != phase || packet.Retained || packet.qos != 1 || len(seen) != len(targets) {
					t.Fatalf("%s broker barrier = %+v, received targets = %v, want %q", phase, packet, seen, targets)
				}
				return
			}
			if !slices.Contains(targets, packet.Topic) || seen[packet.Topic] || packet.Payload != payload || packet.Retained != retained || packet.qos != 0 {
				t.Fatalf("%s unexpected broker PUBLISH = %+v; want targets=%q payload=%q retained=%t QoS=0", phase, packet, targets, payload, retained)
			}
			seen[packet.Topic] = true
		case <-deadline.C:
			t.Fatalf("%s broker did not process the publisher barrier", phase)
		}
	}
}

func retainedClearState(t *testing.T, broker *server.Server, prefix string, want map[string]string) {
	t.Helper()
	messages := broker.Topics.Messages(prefix + "/#")
	if len(messages) != len(want) {
		t.Fatalf("broker retained state has %d entries, want %v: %+v", len(messages), want, messages)
	}
	for _, message := range messages {
		payload, ok := want[message.TopicName]
		if !ok || string(message.Payload) != payload || !message.FixedHeader.Retain {
			t.Fatalf("unexpected retained state: %+v, want %v", message, want)
		}
	}
}

func retainedClearReplay(t *testing.T, addr, prefix, phase string, want map[string]string) {
	t.Helper()
	received := make(chan MQTTMessage, 16)
	client := retainedClearClient(t, addr, "retained-clear-subscriber-"+phase)
	defer client.Disconnect(250)
	retainedClearToken(t, client.Subscribe(prefix+"/#", 0, func(_ mqtt.Client, message mqtt.Message) {
		received <- MQTTMessage{Topic: message.Topic(), Payload: string(message.Payload()), Retained: message.Retained()}
	}), "subscribe "+phase)

	// Mochi sends SUBACK before retained replay. A marker on this same connection
	// is processed after replay; ordered Paho callbacks delimit it without sleeping.
	marker := prefix + "/subscriber-barrier"
	retainedClearToken(t, client.Publish(marker, 1, false, phase), "subscriber barrier "+phase)
	deadline := time.NewTimer(retainedClearTimeout)
	defer deadline.Stop()
	seen := make(map[string]bool)
	for {
		select {
		case message := <-received:
			if message.Topic == marker {
				if message.Payload != phase || message.Retained || len(seen) != len(want) {
					t.Fatalf("%s retained replay = %v, want %v; barrier=%+v", phase, seen, want, message)
				}
				return
			}
			payload, ok := want[message.Topic]
			if !ok || seen[message.Topic] || message.Payload != payload || !message.Retained {
				t.Fatalf("%s unexpected retained replay = %+v, want %v", phase, message, want)
			}
			seen[message.Topic] = true
		case <-deadline.C:
			t.Fatalf("%s subscriber did not receive its replay barrier", phase)
		}
	}
}
