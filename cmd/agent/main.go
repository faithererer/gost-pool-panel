package main

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"gost-pool-panel/internal/gostcfg"
	"gost-pool-panel/internal/model"
)

const agentVersion = "0.2.0"
const defaultGostVersion = "3.2.6"

type Config struct {
	Server        string `json:"server"`
	RegisterToken string `json:"registerToken"`
	NodeName      string `json:"nodeName"`
	NodeID        string `json:"nodeId"`
	AgentToken    string `json:"agentToken"`
}

type Agent struct {
	cfgPath string
	cfg     Config
	client  *http.Client
}

func main() {
	if runtime.GOOS != "linux" {
		log.Fatalf("gost-pool-agent only supports Linux nodes, current OS is %s", runtime.GOOS)
	}

	var cfg Config
	var cfgPath string
	flag.StringVar(&cfg.Server, "server", getenv("GPP_SERVER", ""), "panel server URL")
	flag.StringVar(&cfg.RegisterToken, "token", getenv("GPP_REGISTER_TOKEN", ""), "register token")
	flag.StringVar(&cfg.NodeName, "name", getenv("GPP_NODE_NAME", ""), "node name")
	flag.StringVar(&cfgPath, "config", getenv("GPP_CONFIG", "/opt/gost-pool-agent/agent.json"), "config path")
	flag.Parse()

	a := &Agent{cfgPath: cfgPath, cfg: cfg, client: &http.Client{Timeout: 15 * time.Second}}
	if err := a.loadConfig(); err != nil {
		log.Printf("config load skipped: %v", err)
	}
	if a.cfg.Server == "" {
		log.Fatal("--server is required")
	}
	if a.cfg.NodeID == "" {
		if err := a.register(); err != nil {
			log.Fatalf("register failed: %v", err)
		}
	}

	ticker := time.NewTicker(20 * time.Second)
	defer ticker.Stop()
	for {
		if err := a.heartbeat(); err != nil {
			if isUnauthorized(err) {
				log.Printf("heartbeat unauthorized, trying to re-register with current register token")
				a.cfg.NodeID = ""
				a.cfg.AgentToken = ""
				if regErr := a.register(); regErr != nil {
					log.Printf("re-register failed: %v", regErr)
				}
				<-ticker.C
				continue
			}
			log.Printf("heartbeat failed: %v", err)
		}
		if err := a.pollTasks(); err != nil {
			log.Printf("task polling failed: %v", err)
		}
		<-ticker.C
	}
}

func (a *Agent) loadConfig() error {
	b, err := os.ReadFile(a.cfgPath)
	if err != nil {
		return err
	}
	var saved Config
	if err := json.Unmarshal(b, &saved); err != nil {
		return err
	}
	if a.cfg.Server == "" {
		a.cfg.Server = saved.Server
	}
	if a.cfg.RegisterToken == "" {
		a.cfg.RegisterToken = saved.RegisterToken
	}
	if a.cfg.NodeName == "" {
		a.cfg.NodeName = saved.NodeName
	}
	a.cfg.NodeID = saved.NodeID
	a.cfg.AgentToken = saved.AgentToken
	return nil
}

func (a *Agent) saveConfig() error {
	if err := os.MkdirAll(filepath.Dir(a.cfgPath), 0o700); err != nil {
		return err
	}
	b, err := json.MarshalIndent(a.cfg, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(a.cfgPath, b, 0o600)
}

func (a *Agent) register() error {
	if a.cfg.RegisterToken == "" {
		return errors.New("--token is required for first registration")
	}
	hostname, _ := os.Hostname()
	gostVersion := gostVersion()
	req := map[string]string{
		"token":        a.cfg.RegisterToken,
		"name":         a.cfg.NodeName,
		"hostname":     hostname,
		"os":           linuxPrettyName(),
		"arch":         runtime.GOARCH,
		"agentVersion": agentVersion,
		"gostVersion":  gostVersion,
		"gostStatus":   gostStatus(gostVersion),
	}
	var resp struct {
		NodeID     string `json:"nodeId"`
		AgentToken string `json:"agentToken"`
	}
	if err := a.postJSON("/api/agent/register", "", req, &resp); err != nil {
		return err
	}
	a.cfg.NodeID = resp.NodeID
	a.cfg.AgentToken = resp.AgentToken
	return a.saveConfig()
}

func (a *Agent) heartbeat() error {
	hostname, _ := os.Hostname()
	gostVersion := gostVersion()
	req := map[string]any{
		"hostname":     hostname,
		"os":           linuxPrettyName(),
		"arch":         runtime.GOARCH,
		"agentVersion": agentVersion,
		"gostVersion":  gostVersion,
		"gostStatus":   gostStatus(gostVersion),
	}
	var resp model.Node
	return a.postJSON("/api/agent/heartbeat", a.authHeader(), req, &resp)
}

func (a *Agent) pollTasks() error {
	var tasks []model.Task
	if err := a.getJSON("/api/agent/tasks", a.authHeader(), &tasks); err != nil {
		return err
	}
	for _, t := range tasks {
		status, result, errText := a.executeTask(t)
		req := map[string]string{"status": status, "result": result, "error": errText}
		if err := a.postJSON("/api/agent/tasks/"+t.ID+"/result", a.authHeader(), req, nil); err != nil {
			log.Printf("report task %s failed: %v", t.ID, err)
			continue
		}
		if t.Type == "uninstall_agent" && status == model.TaskStatusSuccess {
			if err := a.scheduleSelfUninstall(); err != nil {
				log.Printf("schedule self uninstall failed: %v", err)
			}
			return nil
		}
	}
	return nil
}

func (a *Agent) executeTask(t model.Task) (string, string, string) {
	switch t.Type {
	case "sync_node_proxy":
		return a.syncNodeProxy(t.Payload)
	case "restart_gost":
		return runCommand("systemctl", "restart", "gost")
	case "apply_config":
		if err := backupAndWrite("/etc/gost/gost.json", []byte(t.Payload)); err != nil {
			return model.TaskStatusFailed, "", err.Error()
		}
		status, result, errText := runCommand("systemctl", "restart", "gost")
		if status == model.TaskStatusFailed {
			return status, result, errText
		}
		return model.TaskStatusSuccess, "gost config applied\n" + result, ""
	case "update_ports":
		return a.syncNodeProxy(t.Payload)
	case "uninstall_agent":
		return model.TaskStatusSuccess, "agent uninstall scheduled; GOST service and /etc/gost will be kept", ""
	default:
		return model.TaskStatusFailed, "", "unknown task type: " + t.Type
	}
}

type nodeProxyPayload struct {
	HTTPPort    int    `json:"httpPort"`
	SocksPort   int    `json:"socksPort"`
	Username    string `json:"username"`
	Password    string `json:"password"`
	GostVersion string `json:"gostVersion"`
}

func (a *Agent) syncNodeProxy(payload string) (string, string, string) {
	if os.Geteuid() != 0 {
		return model.TaskStatusFailed, "", "sync_node_proxy requires root"
	}
	var p nodeProxyPayload
	if payload != "" {
		if err := json.Unmarshal([]byte(payload), &p); err != nil {
			return model.TaskStatusFailed, "", "invalid payload: " + err.Error()
		}
	}
	if p.HTTPPort == 0 {
		p.HTTPPort = 18080
	}
	if p.SocksPort == 0 {
		p.SocksPort = 18081
	}
	if p.Username == "" || p.Password == "" {
		return model.TaskStatusFailed, "", "username and password are required"
	}
	if p.GostVersion == "" {
		p.GostVersion = defaultGostVersion
	}
	if err := ensureGostInstalled(p.GostVersion); err != nil {
		return model.TaskStatusFailed, "", err.Error()
	}
	cfg := gostcfg.NodeProxy(p.HTTPPort, p.SocksPort, p.Username, p.Password)
	b, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return model.TaskStatusFailed, "", err.Error()
	}
	if err := backupAndWrite("/etc/gost/gost.json", b); err != nil {
		return model.TaskStatusFailed, "", err.Error()
	}
	if err := writeGostService(); err != nil {
		return model.TaskStatusFailed, "", err.Error()
	}
	if status, result, errText := runCommand("systemctl", "daemon-reload"); status == model.TaskStatusFailed {
		return status, result, errText
	}
	if status, result, errText := runCommand("systemctl", "enable", "gost"); status == model.TaskStatusFailed {
		return status, result, errText
	}
	if status, result, errText := runCommand("systemctl", "restart", "gost"); status == model.TaskStatusFailed {
		return status, result, errText
	}
	return model.TaskStatusSuccess, fmt.Sprintf("GOST proxy synced: http=%d socks5=%d version=%s", p.HTTPPort, p.SocksPort, p.GostVersion), ""
}

func ensureGostInstalled(version string) error {
	if _, err := exec.LookPath("gost"); err == nil {
		return nil
	}
	arch := runtime.GOARCH
	if arch != "amd64" && arch != "arm64" {
		return fmt.Errorf("unsupported GOST arch: %s", arch)
	}
	url := fmt.Sprintf("https://github.com/go-gost/gost/releases/download/v%s/gost_%s_linux_%s.tar.gz", version, version, arch)
	resp, err := http.Get(url)
	if err != nil {
		return fmt.Errorf("download GOST failed: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("download GOST failed: http %d from %s", resp.StatusCode, url)
	}
	gz, err := gzip.NewReader(resp.Body)
	if err != nil {
		return fmt.Errorf("read GOST archive failed: %w", err)
	}
	defer gz.Close()
	tr := tar.NewReader(gz)
	tmpPath := fmt.Sprintf("/tmp/gost-%s-%d", version, time.Now().UTC().UnixNano())
	defer os.Remove(tmpPath)
	found := false
	for {
		hdr, err := tr.Next()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return fmt.Errorf("extract GOST archive failed: %w", err)
		}
		if hdr.FileInfo().IsDir() || filepath.Base(hdr.Name) != "gost" {
			continue
		}
		f, err := os.OpenFile(tmpPath, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o755)
		if err != nil {
			return err
		}
		if _, err := io.Copy(f, tr); err != nil {
			_ = f.Close()
			return err
		}
		if err := f.Close(); err != nil {
			return err
		}
		found = true
		break
	}
	if !found {
		return errors.New("GOST binary not found in release archive")
	}
	if err := os.MkdirAll("/usr/local/bin", 0o755); err != nil {
		return err
	}
	if err := os.Rename(tmpPath, "/usr/local/bin/gost"); err != nil {
		return err
	}
	return os.Chmod("/usr/local/bin/gost", 0o755)
}

func writeGostService() error {
	unit := `[Unit]
Description=GOST Proxy Service
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
ExecStart=/usr/local/bin/gost -C /etc/gost/gost.json
Restart=always
RestartSec=3
LimitNOFILE=1048576

[Install]
WantedBy=multi-user.target
`
	return os.WriteFile("/etc/systemd/system/gost.service", []byte(unit), 0o644)
}

func (a *Agent) scheduleSelfUninstall() error {
	installDir := filepath.Dir(a.cfgPath)
	if installDir == "." || installDir == "/" {
		installDir = "/opt/gost-pool-agent"
	}
	scriptPath := fmt.Sprintf("/tmp/gost-pool-agent-uninstall-%d.sh", time.Now().UTC().Unix())
	script := fmt.Sprintf(`#!/usr/bin/env sh
set -eu
LOG="/tmp/gost-pool-agent-uninstall.log"
{
  echo "[gost-pool-agent] uninstall started at $(date -u +%%Y-%%m-%%dT%%H:%%M:%%SZ)"
  systemctl disable --now gost-pool-agent.service 2>/dev/null || true
  rm -f /etc/systemd/system/gost-pool-agent.service
  systemctl daemon-reload 2>/dev/null || true
  rm -rf %s
  echo "[gost-pool-agent] uninstall finished; GOST service and /etc/gost were kept"
} >> "$LOG" 2>&1
rm -f "$0"
`, shellQuote(installDir))
	if err := os.WriteFile(scriptPath, []byte(script), 0o700); err != nil {
		return err
	}

	if _, errText, err := runCommandWithError("systemd-run", "--unit", "gost-pool-agent-uninstall", "--description", "Uninstall GOST Pool Agent", "--on-active=2s", "/bin/sh", scriptPath); err == nil {
		return nil
	} else {
		log.Printf("systemd-run unavailable, falling back to nohup: %s", errText)
	}

	cmd := exec.Command("nohup", "/bin/sh", "-c", "sleep 2; /bin/sh "+shellQuote(scriptPath))
	cmd.Stdout = nil
	cmd.Stderr = nil
	return cmd.Start()
}

func (a *Agent) authHeader() string {
	return "Bearer " + a.cfg.NodeID + ":" + a.cfg.AgentToken
}

func (a *Agent) postJSON(path, auth string, reqBody any, out any) error {
	b, err := json.Marshal(reqBody)
	if err != nil {
		return err
	}
	req, err := http.NewRequest(http.MethodPost, strings.TrimRight(a.cfg.Server, "/")+path, bytes.NewReader(b))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	if auth != "" {
		req.Header.Set("Authorization", auth)
	}
	resp, err := a.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return decodeResponse(resp, out)
}

func (a *Agent) getJSON(path, auth string, out any) error {
	req, err := http.NewRequest(http.MethodGet, strings.TrimRight(a.cfg.Server, "/")+path, nil)
	if err != nil {
		return err
	}
	if auth != "" {
		req.Header.Set("Authorization", auth)
	}
	resp, err := a.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return decodeResponse(resp, out)
}

func decodeResponse(resp *http.Response, out any) error {
	b, _ := io.ReadAll(resp.Body)
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("http %d: %s", resp.StatusCode, strings.TrimSpace(string(b)))
	}
	if out == nil || len(b) == 0 {
		return nil
	}
	return json.Unmarshal(b, out)
}

func isUnauthorized(err error) bool {
	if err == nil {
		return false
	}
	return strings.Contains(err.Error(), "http 401:")
}

func backupAndWrite(path string, content []byte) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	if old, err := os.ReadFile(path); err == nil {
		backup := fmt.Sprintf("%s.bak.%s", path, time.Now().UTC().Format("20060102150405"))
		if err := os.WriteFile(backup, old, 0o600); err != nil {
			return err
		}
	}
	return os.WriteFile(path, content, 0o600)
}

func runCommand(name string, args ...string) (string, string, string) {
	cmd := exec.Command(name, args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return model.TaskStatusFailed, string(out), err.Error()
	}
	return model.TaskStatusSuccess, string(out), ""
}

func runCommandWithError(name string, args ...string) (string, string, error) {
	cmd := exec.Command(name, args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return string(out), err.Error(), err
	}
	return string(out), "", nil
}

func commandOutput(name string, args ...string) string {
	cmd := exec.Command(name, args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

func gostVersion() string {
	if _, err := exec.LookPath("gost"); err != nil {
		return "not installed"
	}
	out, errText, err := runCommandWithError("gost", "-V")
	version := strings.TrimSpace(out)
	if version != "" {
		return version
	}
	if err == nil {
		return "installed"
	}
	out, errText, err = runCommandWithError("gost", "-version")
	version = strings.TrimSpace(out)
	if version != "" {
		return version
	}
	if err == nil {
		return "installed"
	}
	if errText != "" {
		return "installed"
	}
	return "unknown"
}

func systemctlStatus(service string) string {
	if _, err := exec.LookPath("systemctl"); err != nil {
		return "systemctl unavailable"
	}
	out, _, err := runCommandWithError("systemctl", "is-active", service)
	out = strings.TrimSpace(out)
	if out != "" {
		return out
	}
	if err != nil {
		return "not installed"
	}
	if out == "" {
		return "unknown"
	}
	return out
}

func gostStatus(version string) string {
	if strings.TrimSpace(version) == "not installed" {
		return "not installed"
	}
	return systemctlStatus("gost")
}

func linuxPrettyName() string {
	b, err := os.ReadFile("/etc/os-release")
	if err != nil {
		return "linux"
	}
	for _, line := range strings.Split(string(b), "\n") {
		if strings.HasPrefix(line, "PRETTY_NAME=") {
			return strings.Trim(strings.TrimPrefix(line, "PRETTY_NAME="), `"`)
		}
	}
	return "linux"
}

func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func shellQuote(v string) string {
	return "'" + strings.ReplaceAll(v, "'", "'\"'\"'") + "'"
}
