package panel

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
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
	cfg   Config
	store *store.Store
}

func NewServer(cfg Config, st *store.Store) *Server {
	return &Server{cfg: cfg, store: st}
}

func (s *Server) Routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/login", s.handleLogin)
	mux.HandleFunc("/logout", s.handleLogout)
	mux.HandleFunc("/install.sh", s.handleInstallScript)
	mux.HandleFunc("/downloads/", s.handleDownload)

	mux.HandleFunc("/api/agent/register", s.handleAgentRegister)
	mux.HandleFunc("/api/agent/heartbeat", s.handleAgentHeartbeat)
	mux.HandleFunc("/api/agent/tasks", s.handleAgentTasks)
	mux.HandleFunc("/api/agent/tasks/", s.handleAgentTaskResult)
	mux.HandleFunc("/api/agent/traffic", s.handleAgentTraffic)

	mux.HandleFunc("/api/admin/", s.requireLoginFunc(s.handleAdminAPI))
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
		s.renderLogin(w, "")
		return
	}
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if err := r.ParseForm(); err != nil {
		s.renderLogin(w, "表单解析失败")
		return
	}
	if r.FormValue("username") != s.cfg.AdminUser || r.FormValue("password") != s.cfg.AdminPassword {
		s.renderLogin(w, "账号或密码错误")
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
	if r.Method == http.MethodPost {
		s.handleFormPost(w, r)
		return
	}
	switch r.URL.Path {
	case "/":
		s.renderDashboard(w)
	case "/nodes":
		s.renderNodes(w)
	case "/tokens":
		s.renderTokens(w)
	case "/groups":
		s.renderGroups(w)
	case "/pools":
		s.renderPools(w)
	case "/tasks":
		s.renderTasks(w)
	case "/settings":
		s.renderSettings(w)
	default:
		http.NotFound(w, r)
	}
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
		if _, err := s.store.CreateTask(r.FormValue("node_id"), r.FormValue("task_type"), r.FormValue("payload")); err != nil {
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
		if _, err := s.store.CreatePool(r.FormValue("name"), r.Form["group_id"], httpPort, socksPort, r.FormValue("strategy")); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		http.Redirect(w, r, "/pools", http.StatusFound)
	case "/settings":
		if err := s.store.UpdateSettings(r.FormValue("proxy_username"), r.FormValue("proxy_password")); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
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

func (s *Server) handleAdminAPI(w http.ResponseWriter, r *http.Request) {
	switch {
	case r.Method == http.MethodGet && r.URL.Path == "/api/admin/nodes":
		writeJSON(w, http.StatusOK, s.store.Snapshot().Nodes)
	case r.Method == http.MethodGet && r.URL.Path == "/api/admin/groups":
		writeJSON(w, http.StatusOK, s.store.Snapshot().Groups)
	case r.Method == http.MethodGet && r.URL.Path == "/api/admin/tasks":
		writeJSON(w, http.StatusOK, s.store.Snapshot().Tasks)
	case r.Method == http.MethodGet && r.URL.Path == "/api/admin/install-command":
		token := r.URL.Query().Get("token")
		name := r.URL.Query().Get("name")
		writeJSON(w, http.StatusOK, map[string]string{"command": s.installCommand(token, name)})
	case r.Method == http.MethodPost && r.URL.Path == "/api/admin/register-tokens":
		var req struct {
			Name     string `json:"name"`
			TTLHours int    `json:"ttlHours"`
		}
		if !decodeJSON(w, r, &req) {
			return
		}
		if req.TTLHours <= 0 {
			req.TTLHours = 24
		}
		t, err := s.store.CreateRegisterToken(req.Name, time.Duration(req.TTLHours)*time.Hour)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}
		writeJSON(w, http.StatusCreated, t)
	default:
		http.NotFound(w, r)
	}
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

func logRequests(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		next.ServeHTTP(w, r)
	})
}

func renderTemplate(w http.ResponseWriter, title string, data any, body string) {
	funcs := template.FuncMap{
		"formatTime":  formatTime,
		"formatBytes": formatBytes,
		"contains":    contains,
		"join":        strings.Join,
		"gostText":    gostText,
	}
	t := template.Must(template.New("page").Funcs(funcs).Parse(baseHTML + body))
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := t.Execute(w, map[string]any{"Title": title, "Data": data}); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func formatTime(t time.Time) string {
	if t.IsZero() {
		return "-"
	}
	return t.Local().Format("2006-01-02 15:04:05")
}

func formatBytes(n int64) string {
	const unit = 1024
	if n < unit {
		return fmt.Sprintf("%d B", n)
	}
	div, exp := int64(unit), 0
	for n/div >= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %ciB", float64(n)/float64(div), "KMGTPE"[exp])
}

func contains(values []string, needle string) bool {
	for _, v := range values {
		if v == needle {
			return true
		}
	}
	return false
}

func gostText(version, status string) string {
	version = strings.TrimSpace(version)
	status = strings.TrimSpace(status)
	if version == "" {
		version = "not installed"
	}
	if status == "" {
		status = "unknown"
	}
	if version == "not installed" && status == "not installed" {
		return "not installed"
	}
	if version == status {
		return version
	}
	return strings.TrimSpace(version + " " + status)
}
