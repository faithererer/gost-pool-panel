package gostcfg

import "testing"

func TestNodeProxyAddsEgressInterfaceMetadata(t *testing.T) {
	cfg := NodeProxy(18080, 18081, "proxy", "secret", "2600:1700:abcd::1234!")
	if len(cfg.Services) != 2 {
		t.Fatalf("services = %d, want 2", len(cfg.Services))
	}
	for _, svc := range cfg.Services {
		got, ok := svc.Metadata["interface"]
		if !ok {
			t.Fatalf("service %s missing interface metadata", svc.Name)
		}
		if got != "2600:1700:abcd::1234!" {
			t.Fatalf("service %s interface = %v", svc.Name, got)
		}
	}
}

func TestNodeProxyOmitsEmptyEgressInterface(t *testing.T) {
	cfg := NodeProxy(18080, 0, "proxy", "secret", "")
	if len(cfg.Services) != 1 {
		t.Fatalf("services = %d, want 1", len(cfg.Services))
	}
	if cfg.Services[0].Metadata != nil {
		t.Fatalf("metadata = %#v, want nil", cfg.Services[0].Metadata)
	}
}
