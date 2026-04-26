package gostcfg

import "testing"

func TestNodeProxyAddsEgressInterfaceMetadata(t *testing.T) {
	cfg := NodeProxy(18080, 18081, "proxy", "secret", "2600:1700:abcd::1234!", "ipv6")
	if len(cfg.Services) != 2 {
		t.Fatalf("services = %d, want 2", len(cfg.Services))
	}
	for _, svc := range cfg.Services {
		if svc.Interface != "2600:1700:abcd::1234!" {
			t.Fatalf("service %s interface = %q", svc.Name, svc.Interface)
		}
		if svc.Resolver != "resolver-ipv6" {
			t.Fatalf("service %s resolver = %q, want resolver-ipv6", svc.Name, svc.Resolver)
		}
	}
	if len(cfg.Resolvers) != 1 {
		t.Fatalf("resolvers = %d, want 1", len(cfg.Resolvers))
	}
	if got := cfg.Resolvers[0].Nameservers[0].Only; got != "ipv6" {
		t.Fatalf("resolver only = %q, want ipv6", got)
	}
}

func TestNodeProxyOmitsEmptyEgressInterface(t *testing.T) {
	cfg := NodeProxy(18080, 0, "proxy", "secret", "", "")
	if len(cfg.Services) != 1 {
		t.Fatalf("services = %d, want 1", len(cfg.Services))
	}
	if cfg.Services[0].Interface != "" {
		t.Fatalf("interface = %q, want empty", cfg.Services[0].Interface)
	}
	if cfg.Services[0].Resolver != "" {
		t.Fatalf("resolver = %q, want empty", cfg.Services[0].Resolver)
	}
	if cfg.Resolvers != nil {
		t.Fatalf("resolvers = %#v, want nil", cfg.Resolvers)
	}
}

func TestNodeProxyAddsPreferIPv6Resolver(t *testing.T) {
	cfg := NodeProxy(18080, 18081, "proxy", "secret", "", "prefer_ipv6")
	if len(cfg.Services) != 2 {
		t.Fatalf("services = %d, want 2", len(cfg.Services))
	}
	for _, svc := range cfg.Services {
		if svc.Interface != "" {
			t.Fatalf("service %s interface = %q, want empty", svc.Name, svc.Interface)
		}
		if svc.Resolver != "resolver-prefer-ipv6" {
			t.Fatalf("service %s resolver = %q, want resolver-prefer-ipv6", svc.Name, svc.Resolver)
		}
	}
	if len(cfg.Resolvers) != 1 {
		t.Fatalf("resolvers = %d, want 1", len(cfg.Resolvers))
	}
	ns := cfg.Resolvers[0].Nameservers[0]
	if ns.Prefer != "ipv6" || ns.Only != "" {
		t.Fatalf("nameserver = %#v, want prefer ipv6 without only", ns)
	}
}
