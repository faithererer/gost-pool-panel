package panel

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"gost-pool-panel/internal/gostcfg"
	"gost-pool-panel/internal/model"
)

func TestProxyAddrIPv6(t *testing.T) {
	got := proxyAddr("http://[2600:1700:abcd::1234]:13000", 28080)
	want := "[2600:1700:abcd::1234]:28080"
	if got != want {
		t.Fatalf("proxyAddr() = %q, want %q", got, want)
	}
}

func TestWritePoolConfigUsesBracketedIPv6NodeAddress(t *testing.T) {
	s := &Server{cfg: Config{DataPath: filepath.Join(t.TempDir(), "state.json")}}
	path, err := s.writePoolConfig(
		model.Pool{ID: "pool_test", HTTPPort: 28080, GroupIDs: []string{"group_test"}},
		[]model.Node{{ID: "node_test", PublicIP: "2600:1700:abcd::1234", HTTPPort: 18080}},
		model.Settings{ProxyUsername: "proxy", ProxyPassword: "secret"},
	)
	if err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var cfg gostcfg.Config
	if err := json.Unmarshal(b, &cfg); err != nil {
		t.Fatal(err)
	}
	got := cfg.Chains[0].Hops[0].Nodes[0].Addr
	want := "[2600:1700:abcd::1234]:18080"
	if got != want {
		t.Fatalf("node addr = %q, want %q", got, want)
	}
}
