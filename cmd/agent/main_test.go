package main

import (
	"errors"
	"strings"
	"testing"
)

func TestParseRouteSourceIPv6(t *testing.T) {
	out := "2606:4700:4700::1111 from :: via fe80::1 dev eth0 src 2600:1700:abcd::1234 metric 1024 pref medium"
	got := parseRouteSourceIP(out, "ipv6")
	want := "2600:1700:abcd::1234"
	if got != want {
		t.Fatalf("parseRouteSourceIP() = %q, want %q", got, want)
	}
}

func TestParseRouteSourceIPv4(t *testing.T) {
	out := "1.1.1.1 via 172.31.0.1 dev eth0 src 172.31.1.20 uid 0"
	got := parseRouteSourceIP(out, "ipv4")
	want := "172.31.1.20"
	if got != want {
		t.Fatalf("parseRouteSourceIP() = %q, want %q", got, want)
	}
}

func TestResolveEgressInterfaceCustom(t *testing.T) {
	got, err := resolveEgressInterface("custom", "eth0")
	if err != nil {
		t.Fatal(err)
	}
	if got != "eth0" {
		t.Fatalf("resolveEgressInterface() = %q, want eth0", got)
	}
}

func TestResolverOnlyForEgressIPv6(t *testing.T) {
	got := resolverOnlyForEgress("ipv6", "2600:1700:abcd::1234!")
	if got != "ipv6" {
		t.Fatalf("resolverOnlyForEgress() = %q, want ipv6", got)
	}
}

func TestPreferIPv6DoesNotForceInterface(t *testing.T) {
	iface, err := resolveEgressInterface("prefer_ipv6", "")
	if err != nil {
		t.Fatal(err)
	}
	if iface != "" {
		t.Fatalf("resolveEgressInterface() = %q, want empty", iface)
	}
	got := resolverOnlyForEgress("prefer_ipv6", "")
	if got != "prefer_ipv6" {
		t.Fatalf("resolverOnlyForEgress() = %q, want prefer_ipv6", got)
	}
}

func TestPreferIPv6DegradesToIPv4WhenProbeFails(t *testing.T) {
	oldProbe := probeIPv6EgressFunc
	probeIPv6EgressFunc = func() error {
		return errors.New("network unreachable")
	}
	defer func() { probeIPv6EgressFunc = oldProbe }()

	got, note := resolverForSync("prefer_ipv6", "")
	if got != "ipv4" {
		t.Fatalf("resolverForSync() resolver = %q, want ipv4", got)
	}
	if !strings.Contains(note, "network unreachable") {
		t.Fatalf("resolverForSync() note = %q, want probe error", note)
	}
}

func TestPreferIPv6KeepsPreferResolverWhenProbeSucceeds(t *testing.T) {
	oldProbe := probeIPv6EgressFunc
	probeIPv6EgressFunc = func() error { return nil }
	defer func() { probeIPv6EgressFunc = oldProbe }()

	got, note := resolverForSync("prefer_ipv6", "")
	if got != "prefer_ipv6" || note != "" {
		t.Fatalf("resolverForSync() = %q, %q; want prefer_ipv6, empty note", got, note)
	}
}

func TestResolverOnlyForCustomIPv6Address(t *testing.T) {
	got := resolverOnlyForEgress("custom", "2600:1700:abcd::1234")
	if got != "ipv6" {
		t.Fatalf("resolverOnlyForEgress() = %q, want ipv6", got)
	}
}
