package main

import "testing"

func TestTrafficPortsNormalizesPorts(t *testing.T) {
	got := trafficPorts(18081, 18080)
	if len(got) != 2 || got[0] != 18080 || got[1] != 18081 {
		t.Fatalf("trafficPorts = %#v", got)
	}

	got = trafficPorts(18080, 18080)
	if len(got) != 1 || got[0] != 18080 {
		t.Fatalf("trafficPorts duplicate = %#v", got)
	}

	got = trafficPorts(0, 70000)
	if len(got) != 0 {
		t.Fatalf("trafficPorts invalid = %#v", got)
	}
}

func TestParseTrafficCounters(t *testing.T) {
	output := `
[12:3456] -A INPUT -p tcp -m tcp --dport 18080 -m comment --comment "gost-pool-panel:in:18080"
[9:2000] -A INPUT -p tcp -m tcp --dport 18081 -m comment --comment "gost-pool-panel:in:18081"
[6:7000] -A OUTPUT -p tcp -m tcp --sport 18080 -m comment --comment "gost-pool-panel:out:18080"
[2:3000] -A OUTPUT -p tcp -m tcp --sport 18081 -m comment --comment "gost-pool-panel:out:18081"
[1:9999] -A OUTPUT -p tcp -m tcp --sport 28080 -m comment --comment "gost-pool-panel:out:28080"
`
	got := parseTrafficCounters(output, map[int]bool{18080: true, 18081: true})
	if got.InBytes != 5456 || got.OutBytes != 10000 {
		t.Fatalf("counter = %#v", got)
	}
}

func TestParseTrafficCountersAllowsUnquotedComment(t *testing.T) {
	output := `[1:42] -A INPUT -p tcp --dport 18080 -m comment --comment gost-pool-panel:in:18080`
	got := parseTrafficCounters(output, map[int]bool{18080: true})
	if got.InBytes != 42 || got.OutBytes != 0 {
		t.Fatalf("counter = %#v", got)
	}
}
