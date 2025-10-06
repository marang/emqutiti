package emqutiti

import (
	"net"
	"os"
	"path/filepath"
	"testing"

	connections "github.com/marang/emqutiti/connections"
)

func TestInitProxyWritesConfig(t *testing.T) {
	dir := t.TempDir()
	oldHome := os.Getenv("HOME")
	os.Setenv("HOME", dir)
	defer os.Setenv("HOME", oldHome)
	oldCfg := os.Getenv("EMQUTITI_HOME")
	cfgDir := filepath.Join(dir, ".config", "emqutiti")
	os.Setenv("EMQUTITI_HOME", cfgDir)
	defer os.Setenv("EMQUTITI_HOME", oldCfg)

	addr, p := initProxy()
	if addr == "" {
		t.Fatalf("no addr returned")
	}
	if p == nil {
		conn, err := net.Dial("tcp", addr)
		if err != nil {
			t.Fatalf("expected reachable proxy at %s: %v", addr, err)
		}
		conn.Close()
		t.Skip("proxy already running; config persistence not verified")
	}
	defer p.Stop()
	if got := connections.LoadProxyAddr(); got != addr {
		t.Fatalf("config addr %q != %q", got, addr)
	}
	// ensure port is listening
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		t.Fatalf("dial proxy: %v", err)
	}
	conn.Close()
}
