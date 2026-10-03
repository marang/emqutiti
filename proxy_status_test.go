package emqutiti

import (
	"bytes"
	"log"
	"net"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/marang/emqutiti/proxy"
)

func TestStartProxyStatusLoggerWritesLog(t *testing.T) {
	m, _ := initialModel(nil)
	output, flags := log.Writer(), log.Flags()
	t.Cleanup(func() { log.SetOutput(output); log.SetFlags(flags) })
	log.SetFlags(0)
	log.SetOutput(m.logs)
	stop := startProxyStatusLogger("")
	stop()
	lines := m.logs.Lines()
	if len(lines) == 0 {
		t.Fatalf("expected log line, got none")
	}
	if regexp.MustCompile(`^\d{4}/\d{2}/\d{2}`).MatchString(lines[0]) {
		t.Fatalf("unexpected log prefix: %q", lines[0])
	}
}

func TestLogProxyStatusRPC(t *testing.T) {
	for _, stalled := range []bool{false, true} {
		name := "success"
		if stalled {
			name = "deadline"
		}
		t.Run(name, func(t *testing.T) {
			var logs bytes.Buffer
			output, flags := log.Writer(), log.Flags()
			log.SetOutput(&logs)
			log.SetFlags(0)
			t.Cleanup(func() { log.SetOutput(output); log.SetFlags(flags) })
			var addr string
			if stalled {
				listener, err := net.Listen("tcp", "127.0.0.1:0")
				if err != nil {
					t.Fatal(err)
				}
				addr = listener.Addr().String()
				connections := make(chan net.Conn, 1)
				// Accept the transport without completing its HTTP/2 handshake.
				go func() {
					defer close(connections)
					if conn, err := listener.Accept(); err == nil {
						connections <- conn
					}
				}()
				t.Cleanup(func() {
					listener.Close()
					for conn := range connections {
						conn.Close()
					}
				})
			} else {
				server, err := proxy.StartProxy("127.0.0.1:0")
				if err != nil {
					t.Fatal(err)
				}
				t.Cleanup(server.Stop)
				addr = server.Addr()
			}
			started := time.Now()
			logProxyStatus(addr)
			if elapsed := time.Since(started); elapsed > 5*time.Second {
				t.Fatalf("status RPC did not respect its deadline: %s", elapsed)
			}
			if stalled {
				if !strings.Contains(logs.String(), "DeadlineExceeded") {
					t.Fatalf("missing status deadline diagnostic: %s", logs.String())
				}
			} else if !strings.Contains(logs.String(), "clients:1 published:0 subscribed:0 deletes:0") {
				t.Fatalf("missing real proxy status: %s", logs.String())
			}
		})
	}
}
