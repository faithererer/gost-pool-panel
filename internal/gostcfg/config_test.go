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
	if len(cfg.Services) != 4 {
		t.Fatalf("services = %d, want 4", len(cfg.Services))
	}
	for _, svc := range cfg.Services[:2] {
		if svc.Handler.Chain != "node-prefer-ipv6-fallback" {
			t.Fatalf("service %s chain = %q, want node-prefer-ipv6-fallback", svc.Name, svc.Handler.Chain)
		}
		if svc.Handler.Retries != 1 {
			t.Fatalf("service %s retries = %d, want 1", svc.Name, svc.Handler.Retries)
		}
		if svc.Interface != "" || svc.Resolver != "" {
			t.Fatalf("service %s interface/resolver = %q/%q, want empty", svc.Name, svc.Interface, svc.Resolver)
		}
	}
	if cfg.Services[2].Addr != "127.0.0.1:61080" || cfg.Services[2].Resolver != "resolver-ipv6" {
		t.Fatalf("v6 egress service = %#v, want loopback resolver-ipv6", cfg.Services[2])
	}
	if cfg.Services[3].Addr != "127.0.0.1:61081" || cfg.Services[3].Resolver != "resolver-ipv4" {
		t.Fatalf("v4 egress service = %#v, want loopback resolver-ipv4", cfg.Services[3])
	}
	if len(cfg.Chains) != 1 {
		t.Fatalf("chains = %d, want 1", len(cfg.Chains))
	}
	chain := cfg.Chains[0]
	if chain.Name != "node-prefer-ipv6-fallback" || chain.Selector.Strategy != "fifo" || chain.Selector.MaxFails != 1 {
		t.Fatalf("chain selector = %#v", chain)
	}
	if len(chain.Hops) != 1 || len(chain.Hops[0].Nodes) != 2 {
		t.Fatalf("chain hops = %#v, want one hop with two nodes", chain.Hops)
	}
	if chain.Hops[0].Nodes[0].Addr != "127.0.0.1:61080" {
		t.Fatalf("first fallback node addr = %q, want v6 loopback", chain.Hops[0].Nodes[0].Addr)
	}
	if chain.Hops[0].Nodes[1].Addr != "127.0.0.1:61081" {
		t.Fatalf("second fallback node addr = %q, want v4 loopback", chain.Hops[0].Nodes[1].Addr)
	}
	if len(cfg.Resolvers) != 2 {
		t.Fatalf("resolvers = %d, want 2", len(cfg.Resolvers))
	}
	if cfg.Resolvers[0].Name != "resolver-ipv6" || cfg.Resolvers[0].Nameservers[0].Only != "ipv6" {
		t.Fatalf("first resolver = %#v, want resolver-ipv6", cfg.Resolvers[0])
	}
	if cfg.Resolvers[1].Name != "resolver-ipv4" || cfg.Resolvers[1].Nameservers[0].Only != "ipv4" {
		t.Fatalf("second resolver = %#v, want resolver-ipv4", cfg.Resolvers[1])
	}
}
