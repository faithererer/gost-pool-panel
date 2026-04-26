package main

import "testing"

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

func TestResolverOnlyForCustomIPv6Address(t *testing.T) {
	got := resolverOnlyForEgress("custom", "2600:1700:abcd::1234")
	if got != "ipv6" {
		t.Fatalf("resolverOnlyForEgress() = %q, want ipv6", got)
	}
}
