package panel

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"gost-pool-panel/internal/model"
	"gost-pool-panel/internal/store"
)

const sessionCookie = "gpp_session"

type Server struct {
	cfg         Config
	store       *store.Store
	poolRuntime *poolRuntimeManager
}

func NewServer(cfg Config, st *store.Store) *Server {
	return &Server{cfg: cfg, store: st, poolRuntime: newPoolRuntimeManager()}
}

func (s *Server) Routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/login", s.handleLogin)
	mux.HandleFunc("/logout", s.handleLogout)
	mux.HandleFunc("/install.sh", s.handleInstallScript)
	mux.HandleFunc("/downloads/", s.handleDownload)
	mux.HandleFunc("/assets/", s.handleFrontendAsset)

	mux.HandleFunc("/api/agent/register-token/check", s.handleRegisterTokenCheck)
	mux.HandleFunc("/api/agent/register", s.handleAgentRegister)
	mux.HandleFunc("/api/agent/heartbeat", s.handleAgentHeartbeat)
	mux.HandleFunc("/api/agent/tasks", s.handleAgentTasks)
	mux.HandleFunc("/api/agent/tasks/", s.handleAgentTaskResult)
	mux.HandleFunc("/api/agent/traffic", s.handleAgentTraffic)

	mux.HandleFunc("/api/admin/login", s.handleAdminLogin)
	mux.HandleFunc("/api/admin/logout", s.handleAdminLogout)
	mux.HandleFunc("/api/admin/session", s.handleAdminSession)
	mux.HandleFunc("/api/admin/", s.requireAdminAPIFunc(s.handleAdminAPI))
	mux.HandleFunc("/", s.requireLoginFunc(s.handleApp))
	return logRequests(mux)
}

func (s *Server) requireLoginFunc(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !s.validSession(r) {
			http.Redirect(w, r, "/login", http.StatusFound)
			return
		}
		next(w, r)
	}
}

func (s *Server) handleLogin(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		if s.frontendAvailable() {
			s.serveFrontend(w, r)
			return
		}
		s.renderFallbackLogin(w, "")
		return
	}
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if err := r.ParseForm(); err != nil {
		s.renderFallbackLogin(w, "表单解析失败")
		return
	}
	if r.FormValue("username") != s.cfg.AdminUser || r.FormValue("password") != s.cfg.AdminPassword {
		s.renderFallbackLogin(w, "账号或密码错误")
		return
	}
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookie,
		Value:    s.signSession(time.Now().Add(24 * time.Hour)),
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})
	http.Redirect(w, r, "/", http.StatusFound)
}

func (s *Server) handleLogout(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, &http.Cookie{Name: sessionCookie, Value: "", Path: "/", MaxAge: -1})
	http.Redirect(w, r, "/login", http.StatusFound)
}

func (s *Server) handleApp(w http.ResponseWriter, r *http.Request) {
	if s.frontendAvailable() {
		s.serveFrontend(w, r)
		return
	}
	if r.Method == http.MethodPost {
		s.handleFormPost(w, r)
		return
	}
	http.Error(w, "frontend dist not found. run: npm run build --prefix frontend", http.StatusServiceUnavailable)
}

func (s *Server) handleFormPost(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "form parse failed", http.StatusBadRequest)
		return
	}
	switch r.URL.Path {
	case "/tokens":
		ttlHours, _ := strconv.Atoi(defaultString(r.FormValue("ttl_hours"), "24"))
		if ttlHours <= 0 {
			ttlHours = 24
		}
		if _, err := s.store.CreateRegisterToken(r.FormValue("name"), time.Duration(ttlHours)*time.Hour); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		http.Redirect(w, r, "/tokens", http.StatusFound)
	case "/groups":
		if _, err := s.store.CreateGroup(r.FormValue("name"), r.FormValue("remark")); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		http.Redirect(w, r, "/groups", http.StatusFound)
	case "/nodes/groups":
		if err := s.store.AssignGroups(r.FormValue("node_id"), r.Form["group_id"]); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		http.Redirect(w, r, "/nodes", http.StatusFound)
	case "/nodes/tasks":
		nodeID := r.FormValue("node_id")
		taskType := r.FormValue("task_type")
		payload := r.FormValue("payload")
		if taskType == "sync_node_proxy" || taskType == "update_ports" {
			var err error
			payload, err = s.buildNodeProxyPayload(r)
			if err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			httpPort, _ := strconv.Atoi(defaultString(r.FormValue("http_port"), "18080"))
			socksPort, _ := strconv.Atoi(defaultString(r.FormValue("socks_port"), "18081"))
			egressMode, egressInterface, err := egressFromForm(r)
			if err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			if err := s.store.UpdateNodeProxyConfig(nodeID, httpPort, socksPort, egressMode, egressInterface); err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			taskType = "sync_node_proxy"
		}
		if _, err := s.store.CreateTask(nodeID, taskType, payload); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		http.Redirect(w, r, "/tasks", http.StatusFound)
	case "/nodes/delete":
		if err := s.store.DeleteNode(r.FormValue("node_id")); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		http.Redirect(w, r, "/nodes", http.StatusFound)
	case "/nodes/cleanup-uninstalled":
		if _, err := s.store.DeleteUninstalledNodes(); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		http.Redirect(w, r, "/nodes", http.StatusFound)
	case "/pools":
		httpPort, _ := strconv.Atoi(r.FormValue("http_port"))
		socksPort, _ := strconv.Atoi(r.FormValue("socks_port"))
		pool, err := s.store.CreatePool(r.FormValue("name"), r.Form["group_id"], httpPort, socksPort, r.FormValue("strategy"))
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if err := s.restartPoolRuntime(pool.ID); err != nil {
			http.Redirect(w, r, "/pools", http.StatusFound)
			return
		}
		http.Redirect(w, r, "/pools", http.StatusFound)
	case "/pools/restart":
		if err := s.restartPoolRuntime(r.FormValue("pool_id")); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		http.Redirect(w, r, "/pools", http.StatusFound)
	case "/settings":
		if err := s.store.UpdateSettings(r.FormValue("proxy_username"), r.FormValue("proxy_password")); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if _, err := s.syncAllNodeProxyTasks(); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		s.restartEnabledPoolRuntimes()
		http.Redirect(w, r, "/settings", http.StatusFound)
	default:
		http.NotFound(w, r)
	}
}

func (s *Server) handleInstallScript(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/x-shellscript; charset=utf-8")
	fmt.Fprint(w, strings.ReplaceAll(installScript, "{{BASE_URL}}", s.cfg.BaseURL))
}

func (s *Server) handleDownload(w http.ResponseWriter, r *http.Request) {
	name := filepath.Base(r.URL.Path)
	path := filepath.Join("dist", name)
	if _, err := os.Stat(path); err != nil {
		http.Error(w, "agent binary not built yet. run: go build -o dist/"+name+" ./cmd/agent", http.StatusNotFound)
		return
	}
	http.ServeFile(w, r, path)
}

func (s *Server) handleFrontendAsset(w http.ResponseWriter, r *http.Request) {
	if !s.frontendAvailable() {
		http.NotFound(w, r)
		return
	}
	s.serveFrontend(w, r)
}

func (s *Server) handleRegisterTokenCheck(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	token := strings.TrimSpace(r.URL.Query().Get("token"))
	if token == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"status":  "missing",
			"message": "register token is required",
		})
		return
	}
	code, message := s.store.CheckRegisterToken(token)
	status := "available"
	if code != http.StatusOK {
		status = "unavailable"
	}
	writeJSON(w, code, map[string]string{
		"status":  status,
		"message": message,
	})
}

func (s *Server) handleAgentRegister(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req struct {
		Token        string `json:"token"`
		Name         string `json:"name"`
		Hostname     string `json:"hostname"`
		OS           string `json:"os"`
		Arch         string `json:"arch"`
		AgentVersion string `json:"agentVersion"`
		GostVersion  string `json:"gostVersion"`
		GostStatus   string `json:"gostStatus"`
	}
	if !decodeJSON(w, r, &req) {
		return
	}
	node, err := s.store.RegisterNode(req.Token, req.Name, store.PublicIPFromAddr(r.RemoteAddr), req.Hostname, req.OS, req.Arch, req.AgentVersion, req.GostVersion, req.GostStatus)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	if _, err := s.store.CreateTaskFromPayload(node.ID, "sync_node_proxy", s.defaultNodeProxyPayload(node)); err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusCreated, map[string]string{
		"nodeId":     node.ID,
		"agentToken": node.AgentToken,
	})
}

func (s *Server) handleAgentHeartbeat(w http.ResponseWriter, r *http.Request) {
	node, ok := s.authorizeAgent(w, r)
	if !ok {
		return
	}
	var req model.Node
	if !decodeJSON(w, r, &req) {
		return
	}
	updated, err := s.store.Heartbeat(node.ID, req)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, http.StatusOK, updated)
}

func (s *Server) handleAgentTasks(w http.ResponseWriter, r *http.Request) {
	node, ok := s.authorizeAgent(w, r)
	if !ok {
		return
	}
	tasks, err := s.store.PendingTasks(node.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, tasks)
}

func (s *Server) handleAgentTaskResult(w http.ResponseWriter, r *http.Request) {
	node, ok := s.authorizeAgent(w, r)
	if !ok {
		return
	}
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	taskID := strings.TrimPrefix(r.URL.Path, "/api/agent/tasks/")
	taskID = strings.TrimSuffix(taskID, "/result")
	var req struct {
		Status string `json:"status"`
		Result string `json:"result"`
		Error  string `json:"error"`
	}
	if !decodeJSON(w, r, &req) {
		return
	}
	if req.Status == "" {
		req.Status = model.TaskStatusSuccess
	}
	if err := s.store.FinishTask(node.ID, taskID, req.Status, req.Result, req.Error); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func (s *Server) handleAgentTraffic(w http.ResponseWriter, r *http.Request) {
	node, ok := s.authorizeAgent(w, r)
	if !ok {
		return
	}
	var req struct {
		UploadBytes   int64 `json:"uploadBytes"`
		DownloadBytes int64 `json:"downloadBytes"`
	}
	if !decodeJSON(w, r, &req) {
		return
	}
	if err := s.store.AddTraffic(node.ID, req.UploadBytes, req.DownloadBytes); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func (s *Server) authorizeAgent(w http.ResponseWriter, r *http.Request) (model.Node, bool) {
	raw := strings.TrimSpace(r.Header.Get("Authorization"))
	raw = strings.TrimPrefix(raw, "Bearer ")
	parts := strings.SplitN(raw, ":", 2)
	if len(parts) != 2 {
		writeErrorText(w, http.StatusUnauthorized, "missing agent token")
		return model.Node{}, false
	}
	node, ok := s.store.AuthenticateAgent(parts[0], parts[1])
	if !ok {
		writeErrorText(w, http.StatusUnauthorized, "invalid agent token")
		return model.Node{}, false
	}
	return node, true
}

func (s *Server) installCommand(token, name string) string {
	cmd := fmt.Sprintf("curl -fsSL %s/install.sh | bash -s -- --server %s --token %s", s.cfg.BaseURL, s.cfg.BaseURL, token)
	if name != "" {
		cmd += " --name " + shellQuote(name)
	}
	return cmd
}

func (s *Server) buildNodeProxyPayload(r *http.Request) (string, error) {
	httpPort, _ := strconv.Atoi(defaultString(r.FormValue("http_port"), "18080"))
	socksPort, _ := strconv.Atoi(defaultString(r.FormValue("socks_port"), "18081"))
	if httpPort <= 0 && socksPort <= 0 {
		return "", fmt.Errorf("HTTP 或 SOCKS5 端口至少需要一个")
	}
	state := s.store.Snapshot()
	username := state.Settings.ProxyUsername
	password := state.Settings.ProxyPassword
	if username == "" || password == "" {
		return "", fmt.Errorf("请先在设置中配置全局出口账号密码")
	}
	egressMode, egressInterface, err := egressFromForm(r)
	if err != nil {
		return "", err
	}
	payload := map[string]any{
		"httpPort":        httpPort,
		"socksPort":       socksPort,
		"username":        username,
		"password":        password,
		"gostVersion":     defaultString(r.FormValue("gost_version"), "3.2.6"),
		"egressMode":      egressMode,
		"egressInterface": egressInterface,
	}
	b, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func (s *Server) defaultNodeProxyPayload(node model.Node) map[string]any {
	state := s.store.Snapshot()
	httpPort := node.HTTPPort
	if httpPort <= 0 {
		httpPort = 18080
	}
	socksPort := node.SocksPort
	if socksPort <= 0 {
		socksPort = 18081
	}
	egressMode := normalizeEgressMode(node.EgressMode)
	return map[string]any{
		"httpPort":        httpPort,
		"socksPort":       socksPort,
		"username":        state.Settings.ProxyUsername,
		"password":        state.Settings.ProxyPassword,
		"gostVersion":     "3.2.6",
		"egressMode":      egressMode,
		"egressInterface": strings.TrimSpace(node.EgressInterface),
	}
}

func (s *Server) syncAllNodeProxyTasks() (int, error) {
	state := s.store.Snapshot()
	count := 0
	for _, n := range state.Nodes {
		if n.GostStatus == "agent uninstalled" {
			continue
		}
		if _, err := s.store.CreateTaskFromPayload(n.ID, "sync_node_proxy", s.defaultNodeProxyPayload(n)); err != nil {
			return count, err
		}
		count++
	}
	return count, nil
}

func (s *Server) restartEnabledPoolRuntimes() {
	state := s.store.Snapshot()
	for _, p := range state.Pools {
		if !p.Enabled {
			continue
		}
		if err := s.restartPoolRuntime(p.ID); err != nil {
			// restartPoolRuntime records the user-visible status on the pool.
			continue
		}
	}
}

func (s *Server) signSession(expires time.Time) string {
	payload := fmt.Sprintf("%s|%d", s.cfg.AdminUser, expires.Unix())
	mac := hmac.New(sha256.New, []byte(s.cfg.Secret))
	mac.Write([]byte(payload))
	sig := hex.EncodeToString(mac.Sum(nil))
	return base64.RawURLEncoding.EncodeToString([]byte(payload + "|" + sig))
}

func (s *Server) validSession(r *http.Request) bool {
	c, err := r.Cookie(sessionCookie)
	if err != nil || c.Value == "" {
		return false
	}
	raw, err := base64.RawURLEncoding.DecodeString(c.Value)
	if err != nil {
		return false
	}
	parts := strings.Split(string(raw), "|")
	if len(parts) != 3 || parts[0] != s.cfg.AdminUser {
		return false
	}
	expUnix, err := strconv.ParseInt(parts[1], 10, 64)
	if err != nil || time.Now().Unix() > expUnix {
		return false
	}
	payload := parts[0] + "|" + parts[1]
	mac := hmac.New(sha256.New, []byte(s.cfg.Secret))
	mac.Write([]byte(payload))
	expected := hex.EncodeToString(mac.Sum(nil))
	return hmac.Equal([]byte(expected), []byte(parts[2]))
}

func decodeJSON(w http.ResponseWriter, r *http.Request, v any) bool {
	defer r.Body.Close()
	if err := json.NewDecoder(r.Body).Decode(v); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return false
	}
	return true
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, err error) {
	writeErrorText(w, status, err.Error())
}

func writeErrorText(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

func defaultString(v, fallback string) string {
	if v == "" {
		return fallback
	}
	return v
}

func shellQuote(v string) string {
	return "'" + strings.ReplaceAll(v, "'", "'\"'\"'") + "'"
}

func proxyAddr(baseURL string, port int) string {
	u, err := url.Parse(baseURL)
	if err != nil || u.Hostname() == "" {
		return "PANEL_IP:" + strconv.Itoa(port)
	}
	return net.JoinHostPort(u.Hostname(), strconv.Itoa(port))
}

func logRequests(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		next.ServeHTTP(w, r)
	})
}

func (s *Server) frontendDistDir() string {
	if v := strings.TrimSpace(os.Getenv("PANEL_FRONTEND_DIST")); v != "" {
		return v
	}
	return filepath.Join("frontend", "dist")
}

func (s *Server) frontendAvailable() bool {
	info, err := os.Stat(filepath.Join(s.frontendDistDir(), "index.html"))
	return err == nil && !info.IsDir()
}

func (s *Server) serveFrontend(w http.ResponseWriter, r *http.Request) {
	dist := s.frontendDistDir()
	indexPath := filepath.Join(dist, "index.html")
	if r.URL.Path == "/" || r.URL.Path == "/login" {
		http.ServeFile(w, r, indexPath)
		return
	}

	requested := filepath.Clean(strings.TrimPrefix(r.URL.Path, "/"))
	if requested == "." || strings.HasPrefix(requested, "..") {
		http.ServeFile(w, r, indexPath)
		return
	}

	path := filepath.Join(dist, requested)
	distAbs, distErr := filepath.Abs(dist)
	pathAbs, pathErr := filepath.Abs(path)
	if distErr != nil || pathErr != nil || (pathAbs != distAbs && !strings.HasPrefix(pathAbs, distAbs+string(os.PathSeparator))) {
		http.NotFound(w, r)
		return
	}
	if info, err := os.Stat(path); err == nil && !info.IsDir() {
		http.ServeFile(w, r, path)
		return
	}
	http.ServeFile(w, r, indexPath)
}

func (s *Server) renderFallbackLogin(w http.ResponseWriter, msg string) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	errorHTML := ""
	if msg != "" {
		errorHTML = `<p style="color:#b42318">` + strings.ReplaceAll(msg, "<", "&lt;") + `</p>`
	}
	fmt.Fprintf(w, `<!doctype html>
<html lang="zh-CN">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>GOST Pool Panel</title>
  <style>
    body { margin: 0; min-height: 100vh; display: grid; place-items: center; font-family: ui-sans-serif, system-ui, -apple-system, BlinkMacSystemFont, "Segoe UI", sans-serif; background: #0f172a; color: #e5e7eb; }
    form { width: min(360px, calc(100vw - 32px)); background: #111827; border: 1px solid #374151; border-radius: 8px; padding: 24px; }
    h1 { margin: 0 0 18px; font-size: 20px; }
    label { display: block; margin: 12px 0 6px; font-size: 13px; color: #9ca3af; }
    input { width: 100%%; box-sizing: border-box; padding: 10px 12px; border-radius: 6px; border: 1px solid #374151; background: #030712; color: #f9fafb; }
    button { margin-top: 16px; width: 100%%; padding: 10px 12px; border: 0; border-radius: 6px; background: #e5e7eb; color: #111827; font-weight: 700; cursor: pointer; }
    p { font-size: 13px; }
  </style>
</head>
<body>
  <form method="post" action="/login">
    <h1>GOST Pool Panel</h1>
    %s
    <label>账号</label>
    <input name="username" value="admin" autocomplete="username">
    <label>密码</label>
    <input name="password" type="password" autocomplete="current-password">
    <button type="submit">登录</button>
    <p>未找到前端构建产物时显示该备用登录页。完整 UI 请运行 npm run build --prefix frontend。</p>
  </form>
</body>
</html>`, errorHTML)
}

func normalizeEgressMode(mode string) string {
	switch strings.ToLower(strings.TrimSpace(mode)) {
	case "ipv4", "ipv6", "custom":
		return strings.ToLower(strings.TrimSpace(mode))
	default:
		return "auto"
	}
}

func egressFromForm(r *http.Request) (string, string, error) {
	mode := normalizeEgressMode(r.FormValue("egress_mode"))
	iface := strings.TrimSpace(r.FormValue("egress_interface"))
	if mode == "custom" && iface == "" {
		return "", "", fmt.Errorf("自定义出口需要填写接口名或本机 IP")
	}
	if mode != "custom" {
		iface = ""
	}
	return mode, iface, nil
}
