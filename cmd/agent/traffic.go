package main

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

const trafficCommentPrefix = "gost-pool-panel"

type TrafficState struct {
	Key      string `json:"key,omitempty"`
	InBytes  int64  `json:"inBytes,omitempty"`
	OutBytes int64  `json:"outBytes,omitempty"`
}

type trafficCounter struct {
	InBytes  int64
	OutBytes int64
}

func (a *Agent) reportTraffic(httpPort, socksPort int) error {
	ports := trafficPorts(httpPort, socksPort)
	if len(ports) == 0 {
		return nil
	}
	if os.Geteuid() != 0 {
		return errors.New("traffic report requires root")
	}
	if err := ensureTrafficCounterRules(ports); err != nil {
		return err
	}
	current, err := readTrafficCounters(ports)
	if err != nil {
		return err
	}

	key := trafficKey(ports)
	if a.cfg.Traffic.Key != key {
		a.cfg.Traffic = TrafficState{Key: key, InBytes: current.InBytes, OutBytes: current.OutBytes}
		return a.saveConfig()
	}

	download := current.InBytes - a.cfg.Traffic.InBytes
	upload := current.OutBytes - a.cfg.Traffic.OutBytes
	if download < 0 {
		download = current.InBytes
	}
	if upload < 0 {
		upload = current.OutBytes
	}
	if download == 0 && upload == 0 {
		return nil
	}

	req := map[string]int64{
		"uploadBytes":   upload,
		"downloadBytes": download,
	}
	if err := a.postJSON("/api/agent/traffic", a.authHeader(), req, nil); err != nil {
		return err
	}
	a.cfg.Traffic = TrafficState{Key: key, InBytes: current.InBytes, OutBytes: current.OutBytes}
	return a.saveConfig()
}

func trafficPorts(httpPort, socksPort int) []int {
	seen := make(map[int]bool)
	for _, port := range []int{httpPort, socksPort} {
		if port <= 0 || port > 65535 || seen[port] {
			continue
		}
		seen[port] = true
	}
	ports := make([]int, 0, len(seen))
	for port := range seen {
		ports = append(ports, port)
	}
	sort.Ints(ports)
	return ports
}

func trafficKey(ports []int) string {
	parts := make([]string, 0, len(ports))
	for _, port := range ports {
		parts = append(parts, strconv.Itoa(port))
	}
	return strings.Join(parts, ",")
}

func ensureTrafficCounterRules(ports []int) error {
	var ok bool
	var failures []string
	for _, bin := range []string{"iptables", "ip6tables"} {
		if _, err := exec.LookPath(bin); err != nil {
			continue
		}
		familyOK := true
		for _, port := range ports {
			if err := ensureTrafficCounterRule(bin, "INPUT", "--dport", port, trafficComment("in", port)); err != nil {
				familyOK = false
				failures = append(failures, fmt.Sprintf("%s input:%d: %v", bin, port, err))
			}
			if err := ensureTrafficCounterRule(bin, "OUTPUT", "--sport", port, trafficComment("out", port)); err != nil {
				familyOK = false
				failures = append(failures, fmt.Sprintf("%s output:%d: %v", bin, port, err))
			}
		}
		if familyOK {
			ok = true
		}
	}
	if ok {
		return nil
	}
	if len(failures) == 0 {
		return errors.New("iptables/ip6tables not found")
	}
	return errors.New(strings.Join(failures, "; "))
}

func ensureTrafficCounterRule(bin, chain, portFlag string, port int, comment string) error {
	args := []string{"-w", "-C", chain, "-p", "tcp", portFlag, strconv.Itoa(port), "-m", "comment", "--comment", comment}
	if err := runQuiet(bin, args...); err == nil {
		return nil
	}
	args = []string{"-w", "-I", chain, "1", "-p", "tcp", portFlag, strconv.Itoa(port), "-m", "comment", "--comment", comment}
	return runQuiet(bin, args...)
}

func readTrafficCounters(ports []int) (trafficCounter, error) {
	portSet := make(map[int]bool, len(ports))
	for _, port := range ports {
		portSet[port] = true
	}
	var out trafficCounter
	var ok bool
	var failures []string
	for _, bin := range []string{"iptables-save", "ip6tables-save"} {
		if _, err := exec.LookPath(bin); err != nil {
			continue
		}
		b, err := exec.Command(bin, "-c").Output()
		if err != nil {
			failures = append(failures, fmt.Sprintf("%s: %v", bin, err))
			continue
		}
		counter := parseTrafficCounters(string(b), portSet)
		out.InBytes += counter.InBytes
		out.OutBytes += counter.OutBytes
		ok = true
	}
	if ok {
		return out, nil
	}
	if len(failures) == 0 {
		return out, errors.New("iptables-save/ip6tables-save not found")
	}
	return out, errors.New(strings.Join(failures, "; "))
}

var trafficCounterLineRE = regexp.MustCompile(`^\[\d+:(\d+)\].*--comment "?` + trafficCommentPrefix + `:(in|out):(\d+)"?`)

func parseTrafficCounters(output string, ports map[int]bool) trafficCounter {
	var counter trafficCounter
	for _, line := range strings.Split(output, "\n") {
		m := trafficCounterLineRE.FindStringSubmatch(strings.TrimSpace(line))
		if len(m) != 4 {
			continue
		}
		bytes, err := strconv.ParseInt(m[1], 10, 64)
		if err != nil {
			continue
		}
		port, err := strconv.Atoi(m[3])
		if err != nil || !ports[port] {
			continue
		}
		switch m[2] {
		case "in":
			counter.InBytes += bytes
		case "out":
			counter.OutBytes += bytes
		}
	}
	return counter
}

func trafficComment(direction string, port int) string {
	return fmt.Sprintf("%s:%s:%d", trafficCommentPrefix, direction, port)
}

func runQuiet(name string, args ...string) error {
	cmd := exec.Command(name, args...)
	if b, err := cmd.CombinedOutput(); err != nil {
		msg := strings.TrimSpace(string(b))
		if msg == "" {
			return err
		}
		return fmt.Errorf("%w: %s", err, msg)
	}
	return nil
}
