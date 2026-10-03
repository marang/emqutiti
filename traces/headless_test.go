package traces

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"log"
	"net"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/marang/emqutiti/proxy"
	"github.com/mochi-co/mqtt/server"
	"github.com/mochi-co/mqtt/server/events"
	"github.com/mochi-co/mqtt/server/listeners"
)

func headlessTestEnvironment(t *testing.T) (*bytes.Buffer, *server.Server) {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("HOME", dir)
	t.Setenv("EMQUTITI_HOME", dir)

	p, err := proxy.StartProxy("127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	previousAddr := proxyAddr
	SetProxyAddr(p.Addr())
	t.Cleanup(func() {
		p.Stop()
		SetProxyAddr(previousAddr)
	})

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	addr := ln.Addr().String()
	if err := ln.Close(); err != nil {
		t.Fatal(err)
	}
	broker := server.New()
	if err := broker.AddListener(listeners.NewTCP("headless-test", addr), nil); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := broker.Close(); err != nil {
			t.Errorf("close broker: %v", err)
		}
	})
	if err := broker.Serve(); err != nil {
		t.Fatal(err)
	}
	host, port, err := net.SplitHostPort(addr)
	if err != nil {
		t.Fatal(err)
	}
	config := fmt.Sprintf("[[profiles]]\nname = 'headless-test'\nschema = 'tcp'\nhost = %q\nport = %s\nclient_id = 'headless-test'\nclean_start = true\nconnect_timeout = 2\n", host, port)
	if err := os.WriteFile(filepath.Join(dir, "config.toml"), []byte(config), 0o600); err != nil {
		t.Fatal(err)
	}

	var logs bytes.Buffer
	previousLog := log.Writer()
	log.SetOutput(&logs)
	t.Cleanup(func() { log.SetOutput(previousLog) })
	return &logs, broker
}

func TestHeadlessRunReturnsCompletedSubscriptionFailure(t *testing.T) {
	logs, _ := headlessTestEnvironment(t)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := Run(ctx, "failed-trace", "bad/#/suffix", "headless-test", "", "")
	if !errors.Is(err, mqtt.ErrInvalidTopicMultilevel) {
		t.Fatalf("completed subscription failure returned %v, want %v", err, mqtt.ErrInvalidTopicMultilevel)
	}
	if text := logs.String(); !strings.Contains(text, "failed-trace") || !strings.Contains(text, "subscribe bad/#/suffix:") {
		t.Fatalf("subscription diagnostic missing trace/topic context: %q", text)
	}
}

func TestHeadlessRunReturnsPlannedSubscriptionFailure(t *testing.T) {
	logs, _ := headlessTestEnvironment(t)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	start := time.Now().Add(2 * time.Second).Format(time.RFC3339)

	err := Run(ctx, "planned-failure", "bad/#/suffix", "headless-test", start, "")
	if !errors.Is(err, mqtt.ErrInvalidTopicMultilevel) {
		t.Fatalf("planned subscription failure returned %v, want %v", err, mqtt.ErrInvalidTopicMultilevel)
	}
	if text := logs.String(); !strings.Contains(text, "planned-failure") || !strings.Contains(text, "subscribe bad/#/suffix:") {
		t.Fatalf("planned subscription diagnostic missing trace/topic context: %q", text)
	}
}

func TestHeadlessRunPastEndStillCompletes(t *testing.T) {
	logs, _ := headlessTestEnvironment(t)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	end := time.Now().Add(-time.Second).Format(time.RFC3339)
	result := make(chan error, 1)
	go func() {
		result <- Run(ctx, "finished-trace", "headless/topic", "headless-test", "", end)
	}()
	select {
	case err := <-result:
		if err != nil {
			t.Fatalf("finished trace returned %v", err)
		}
	case <-time.After(2 * time.Second):
		cancel()
		t.Fatal("finished trace waited for a runtime end timer instead of leaving the end loop")
	}
	if text := logs.String(); !strings.Contains(text, "headless/topic: 0") {
		t.Fatalf("clean completion lost count summary: %q", text)
	}
}

func TestHeadlessRunPastEndUnblocksPendingSubscription(t *testing.T) {
	_, broker := headlessTestEnvironment(t)
	entered, release := make(chan struct{}), make(chan struct{})
	unblock := sync.OnceFunc(func() { close(release) })
	t.Cleanup(unblock)
	broker.Events.OnSubscribe = func(string, events.Client, byte) {
		close(entered)
		<-release
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	end := time.Now().Add(-time.Second).Format(time.RFC3339)
	result := make(chan error, 1)
	go func() {
		result <- Run(ctx, "pending-trace", "headless/pending", "headless-test", "", end)
	}()
	select {
	case <-entered:
	case <-time.After(2 * time.Second):
		t.Fatal("test broker did not receive the subscription")
	}
	select {
	case err := <-result:
		if err == nil || !strings.Contains(err.Error(), "subscribe headless/pending:") {
			t.Fatalf("interrupted subscription returned %v, want its runtime failure", err)
		}
	case <-time.After(time.Second):
		unblock()
		select {
		case <-result:
		case <-time.After(time.Second):
			t.Error("headless run did not finish after releasing SUBACK")
		}
		t.Fatal("completion waited for SUBACK instead of disconnecting the client")
	}
}

func TestHeadlessRunDelayedSubscriptionCapturesUntilEnd(t *testing.T) {
	for _, window := range []string{"immediate", "planned", "open-ended"} {
		t.Run(window, func(t *testing.T) {
			logs, broker := headlessTestEnvironment(t)
			entered, release := make(chan struct{}), make(chan struct{})
			unblock := sync.OnceFunc(func() { close(release) })
			broker.Events.OnSubscribe = func(string, events.Client, byte) {
				close(entered)
				<-release
			}
			ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
			defer cancel()
			startStr, endStr := "", ""
			end := time.Now().Add(4 * time.Second).Truncate(time.Second)
			if window == "planned" {
				start := time.Now().Add(2 * time.Second).Truncate(time.Second)
				startStr = start.Format(time.RFC3339)
				end = start.Add(3 * time.Second)
			}
			if window != "open-ended" {
				endStr = end.Format(time.RFC3339)
			}
			result := make(chan error, 1)
			go func() {
				result <- Run(ctx, window, "headless/active", "headless-test", startStr, endStr)
			}()
			finished := false
			t.Cleanup(func() {
				unblock()
				cancel()
				if !finished {
					select {
					case <-result:
					case <-time.After(2 * time.Second):
						t.Error("headless worker did not exit during cleanup")
					}
				}
			})
			select {
			case <-entered:
			case <-time.After(3 * time.Second):
				t.Fatal("broker did not receive the subscription")
			}
			select {
			case err := <-result:
				finished = true
				t.Fatalf("trace exited before SUBACK: %v", err)
			case <-time.After(700 * time.Millisecond):
			}
			unblock()
			for _, payload := range []string{"one", "two"} {
				if err := broker.Publish("headless/active", []byte(payload), false); err != nil {
					t.Fatal(err)
				}
				deadline := time.Now().Add(time.Second)
				for {
					messages, err := tracerMessages("headless-test", window)
					if err != nil {
						t.Fatal(err)
					}
					found := false
					for _, message := range messages {
						found = found || message.Payload == payload && message.Topic == "headless/active"
					}
					if found {
						break
					}
					if time.Now().After(deadline) {
						t.Fatalf("active trace did not persist %q: %+v", payload, messages)
					}
					time.Sleep(10 * time.Millisecond)
				}
			}
			select {
			case err := <-result:
				finished = true
				t.Fatalf("trace exited before its end: %v", err)
			default:
			}
			if window == "open-ended" {
				cancel()
			}
			select {
			case err := <-result:
				finished = true
				if window == "open-ended" {
					if !errors.Is(err, context.Canceled) {
						t.Fatalf("open-ended trace returned %v, want cancellation", err)
					}
				} else if err != nil || time.Now().Before(end) {
					t.Fatalf("trace did not finish at its natural end %s: %v", end, err)
				}
			case <-time.After(time.Until(end) + time.Second):
				t.Fatal("trace did not complete at its end")
			}
			if text := logs.String(); !strings.Contains(text, "headless/active: 2") {
				t.Fatalf("completion lost captured message counts: %q", text)
			}
		})
	}
}

func TestHeadlessRunCancellationUnblocksPendingSubscription(t *testing.T) {
	for _, action := range []string{"context", "interrupt", "end"} {
		t.Run(action, func(t *testing.T) {
			logs, broker := headlessTestEnvironment(t)
			entered, release := make(chan struct{}), make(chan struct{})
			unblock := sync.OnceFunc(func() { close(release) })
			broker.Events.OnSubscribe = func(string, events.Client, byte) {
				close(entered)
				<-release
			}
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			result := make(chan error, 1)
			startTime := time.Now().Add(2 * time.Second).Truncate(time.Second)
			start := startTime.Format(time.RFC3339)
			end := ""
			deadline := time.Second
			if action == "end" {
				end = startTime.Add(time.Second).Format(time.RFC3339)
				deadline = 2 * time.Second
			}
			go func() {
				result <- Run(ctx, action, "headless/stalled", "headless-test", start, end)
			}()
			finished := false
			t.Cleanup(func() {
				unblock()
				cancel()
				if !finished {
					select {
					case <-result:
					case <-time.After(2 * time.Second):
						t.Error("headless worker did not exit during cleanup")
					}
				}
			})
			select {
			case <-entered:
			case <-time.After(3 * time.Second):
				t.Fatal("broker did not receive the subscription")
			}
			if action == "context" {
				cancel()
			} else if action == "interrupt" {
				process, err := os.FindProcess(os.Getpid())
				if err != nil {
					t.Fatal(err)
				}
				if err := process.Signal(os.Interrupt); err != nil {
					t.Fatal(err)
				}
			}
			select {
			case err := <-result:
				finished = true
				if action == "context" && !errors.Is(err, context.Canceled) {
					t.Fatalf("cancellation returned %v, want context.Canceled", err)
				}
				if action != "context" && (err == nil || !strings.Contains(err.Error(), "subscribe headless/stalled:")) {
					t.Fatalf("interrupted subscription returned %v, want its runtime failure", err)
				}
			case <-time.After(deadline):
				t.Fatal("cancellation waited for SUBACK instead of disconnecting")
			}
			if text := logs.String(); !strings.Contains(text, "subscribe headless/stalled:") {
				t.Fatalf("teardown did not drain the subscription failure: %q", text)
			}
		})
	}
}
