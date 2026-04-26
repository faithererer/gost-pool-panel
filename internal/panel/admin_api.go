package panel

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"gost-pool-panel/internal/buildinfo"
	"gost-pool-panel/internal/model"
	"gost-pool-panel/internal/store"
)

type adminVersions struct {
	Panel string `json:"panel"`
	Agent string `json:"agent"`
}

type adminSummary struct {
	TotalNodes         int          `json:"totalNodes"`
	OnlineNodes        int          `json:"onlineNodes"`
	GostActiveNodes    int          `json:"gostActiveNodes"`
	RunningPools       int          `json:"runningPools"`
	FailedTasks        int          `json:"failedTasks"`
	OutdatedAgentNodes int          `json:"outdatedAgentNodes"`
	RecentErrorTasks   []model.Task `json:"recentErrorTasks"`
}

type registerTokenView struct {
	model.RegisterToken
	InstallCommand string `json:"installCommand"`
}

type adminStateResponse struct {
	Nodes          []model.Node        `json:"nodes"`
	Groups         []model.Group       `json:"groups"`
	Pools          []model.Pool        `json:"pools"`
	RegisterTokens []registerTokenView `json:"registerTokens"`
	Tasks          []model.Task        `json:"tasks"`
	Settings       model.Settings      `json:"settings"`
	BaseURL        string              `json:"baseURL"`
	Versions       adminVersions       `json:"versions"`
	Summary        adminSummary        `json:"summary"`
}

type loginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type createTokenRequest struct {
	Name     string `json:"name"`
	TTLHours int    `json:"ttlHours"`
}

type nodeGroupsRequest struct {
	GroupIDs []string `json:"groupIds"`
}

type nodeTaskRequest struct {
	Type            string `json:"type"`
	Payload         string `json:"payload"`
	HTTPPort        int    `json:"httpPort"`
	SocksPort       int    `json:"socksPort"`
	GostVersion     string `json:"gostVersion"`
	EgressMode      string `json:"egressMode"`
	EgressInterface string `json:"egressInterface"`
}

type batchNodeTaskRequest struct {
	NodeIDs []string `json:"nodeIds"`
	nodeTaskRequest
}

type groupRequest struct {
	Name   string `json:"name"`
	Remark string `json:"remark"`
}

type poolRequest struct {
	Name      string   `json:"name"`
	GroupIDs  []string `json:"groupIds"`
	HTTPPort  int      `json:"httpPort"`
	SocksPort int      `json:"socksPort"`
	Strategy  string   `json:"strategy"`
	Enabled   *bool    `json:"enabled"`
}

type settingsRequest struct {
	ProxyUsername string `json:"proxyUsername"`
	ProxyPassword string `json:"proxyPassword"`
}

func (s *Server) requireAdminAPIFunc(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !s.validSession(r) {
			writeErrorText(w, http.StatusUnauthorized, "not logged in")
			return
		}
		next(w, r)
	}
}

func (s *Server) handleAdminLogin(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeErrorText(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	var req loginRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	if req.Username != s.cfg.AdminUser || req.Password != s.cfg.AdminPassword {
		writeErrorText(w, http.StatusUnauthorized, "username or password is incorrect")
		return
	}
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookie,
		Value:    s.signSession(time.Now().Add(24 * time.Hour)),
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})
	writeJSON(w, http.StatusOK, map[string]any{
		"authenticated": true,
		"user":          s.cfg.AdminUser,
	})
}

func (s *Server) handleAdminLogout(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeErrorText(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	http.SetCookie(w, &http.Cookie{Name: sessionCookie, Value: "", Path: "/", MaxAge: -1})
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func (s *Server) handleAdminSession(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeErrorText(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	if !s.validSession(r) {
		writeErrorText(w, http.StatusUnauthorized, "not logged in")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"authenticated": true,
		"user":          s.cfg.AdminUser,
	})
}

func (s *Server) handleAdminAPI(w http.ResponseWriter, r *http.Request) {
	parts := adminPathParts(r.URL.Path)
	if len(parts) == 0 {
		writeErrorText(w, http.StatusNotFound, "not found")
		return
	}

	switch parts[0] {
	case "state":
		if r.Method == http.MethodGet && len(parts) == 1 {
			writeJSON(w, http.StatusOK, s.adminState())
			return
		}
	case "version":
		if r.Method == http.MethodGet && len(parts) == 1 {
			writeJSON(w, http.StatusOK, s.adminVersions())
			return
		}
	case "install-command":
		if r.Method == http.MethodGet && len(parts) == 1 {
			token := r.URL.Query().Get("token")
			name := r.URL.Query().Get("name")
			writeJSON(w, http.StatusOK, map[string]string{"command": s.installCommand(token, name)})
			return
		}
	case "register-tokens":
		if r.Method == http.MethodPost && len(parts) == 1 {
			s.handleAdminCreateRegisterToken(w, r)
			return
		}
	case "nodes":
		if s.handleAdminNodesAPI(w, r, parts) {
			return
		}
	case "groups":
		if s.handleAdminGroupsAPI(w, r, parts) {
			return
		}
	case "pools":
		if s.handleAdminPoolsAPI(w, r, parts) {
			return
		}
	case "tasks":
		if s.handleAdminTasksAPI(w, r, parts) {
			return
		}
	case "settings":
		if r.Method == http.MethodPatch && len(parts) == 1 {
			s.handleAdminUpdateSettings(w, r)
			return
		}
	}
	writeErrorText(w, http.StatusNotFound, "not found")
}

func (s *Server) handleAdminCreateRegisterToken(w http.ResponseWriter, r *http.Request) {
	var req createTokenRequest
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
	writeJSON(w, http.StatusCreated, s.registerTokenView(t))
}

func (s *Server) handleAdminNodesAPI(w http.ResponseWriter, r *http.Request, parts []string) bool {
	if r.Method == http.MethodGet && len(parts) == 1 {
		writeJSON(w, http.StatusOK, sanitizedNodes(s.store.Snapshot().Nodes))
		return true
	}
	if r.Method == http.MethodPost && len(parts) == 2 && parts[1] == "tasks" {
		s.handleAdminCreateBatchNodeTask(w, r)
		return true
	}
	if r.Method == http.MethodPost && len(parts) == 2 && parts[1] == "cleanup-uninstalled" {
		count, err := s.store.DeleteUninstalledNodes()
		if err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return true
		}
		writeJSON(w, http.StatusOK, map[string]any{"ok": true, "count": count})
		return true
	}
	if len(parts) < 2 {
		return false
	}
	nodeID := parts[1]
	if r.Method == http.MethodDelete && len(parts) == 2 {
		if err := s.store.DeleteNode(nodeID); err != nil {
			writeError(w, http.StatusBadRequest, err)
			return true
		}
		writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
		return true
	}
	if r.Method == http.MethodPost && len(parts) == 3 && parts[2] == "groups" {
		var req nodeGroupsRequest
		if !decodeJSON(w, r, &req) {
			return true
		}
		if err := s.store.AssignGroups(nodeID, req.GroupIDs); err != nil {
			writeError(w, http.StatusBadRequest, err)
			return true
		}
		node, _ := s.store.Node(nodeID)
		node.AgentToken = ""
		writeJSON(w, http.StatusOK, node)
		return true
	}
	if r.Method == http.MethodPost && len(parts) == 3 && parts[2] == "tasks" {
		var req nodeTaskRequest
		if !decodeJSON(w, r, &req) {
			return true
		}
		task, err := s.createNodeTask(nodeID, req)
		if err != nil {
			writeError(w, http.StatusBadRequest, err)
			return true
		}
		writeJSON(w, http.StatusCreated, task)
		return true
	}
	return false
}

func (s *Server) handleAdminCreateBatchNodeTask(w http.ResponseWriter, r *http.Request) {
	var req batchNodeTaskRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	if len(req.NodeIDs) == 0 {
		writeErrorText(w, http.StatusBadRequest, "nodeIds are required")
		return
	}
	taskType := normalizeTaskType(req.Type)
	if err := validateTaskRequest(taskType, req.Payload); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	if taskType == "sync_node_proxy" || taskType == "update_ports" {
		var tasks []model.Task
		nodeIDs := uniqueRequestStrings(req.NodeIDs)
		if len(nodeIDs) == 0 {
			writeErrorText(w, http.StatusBadRequest, "nodeIds are required")
			return
		}
		for _, nodeID := range nodeIDs {
			task, err := s.createNodeTask(nodeID, req.nodeTaskRequest)
			if err != nil {
				writeError(w, http.StatusBadRequest, err)
				return
			}
			tasks = append(tasks, task)
		}
		writeJSON(w, http.StatusCreated, tasks)
		return
	}
	tasks, err := s.store.CreateTasks(req.NodeIDs, taskType, req.Payload)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, http.StatusCreated, tasks)
}

func (s *Server) handleAdminGroupsAPI(w http.ResponseWriter, r *http.Request, parts []string) bool {
	if r.Method == http.MethodGet && len(parts) == 1 {
		writeJSON(w, http.StatusOK, s.store.Snapshot().Groups)
		return true
	}
	if r.Method == http.MethodPost && len(parts) == 1 {
		var req groupRequest
		if !decodeJSON(w, r, &req) {
			return true
		}
		g, err := s.store.CreateGroup(req.Name, req.Remark)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return true
		}
		writeJSON(w, http.StatusCreated, g)
		return true
	}
	if len(parts) != 2 {
		return false
	}
	groupID := parts[1]
	switch r.Method {
	case http.MethodPatch:
		var req groupRequest
		if !decodeJSON(w, r, &req) {
			return true
		}
		g, err := s.store.UpdateGroup(groupID, req.Name, req.Remark)
		if err != nil {
			writeError(w, http.StatusBadRequest, err)
			return true
		}
		writeJSON(w, http.StatusOK, g)
		return true
	case http.MethodDelete:
		if err := s.store.DeleteGroup(groupID); err != nil {
			writeError(w, http.StatusBadRequest, err)
			return true
		}
		s.restartEnabledPoolRuntimes()
		writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
		return true
	}
	return false
}

func (s *Server) handleAdminPoolsAPI(w http.ResponseWriter, r *http.Request, parts []string) bool {
	if r.Method == http.MethodGet && len(parts) == 1 {
		writeJSON(w, http.StatusOK, s.store.Snapshot().Pools)
		return true
	}
	if r.Method == http.MethodPost && len(parts) == 1 {
		var req poolRequest
		if !decodeJSON(w, r, &req) {
			return true
		}
		pool, err := s.store.CreatePool(req.Name, req.GroupIDs, req.HTTPPort, req.SocksPort, req.Strategy)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return true
		}
		_ = s.restartPoolRuntime(pool.ID)
		pool, _ = s.store.Pool(pool.ID)
		writeJSON(w, http.StatusCreated, pool)
		return true
	}
	if len(parts) < 2 {
		return false
	}
	poolID := parts[1]
	if r.Method == http.MethodPost && len(parts) == 3 && parts[2] == "restart" {
		if err := s.restartPoolRuntime(poolID); err != nil {
			writeError(w, http.StatusBadRequest, err)
			return true
		}
		pool, _ := s.store.Pool(poolID)
		writeJSON(w, http.StatusOK, pool)
		return true
	}
	if len(parts) != 2 {
		return false
	}
	switch r.Method {
	case http.MethodPatch:
		pool, err := s.updatePoolFromJSON(w, r, poolID)
		if err != nil {
			writeError(w, http.StatusBadRequest, err)
			return true
		}
		writeJSON(w, http.StatusOK, pool)
		return true
	case http.MethodDelete:
		s.stopPoolRuntime(poolID)
		if err := s.store.DeletePool(poolID); err != nil {
			writeError(w, http.StatusBadRequest, err)
			return true
		}
		writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
		return true
	}
	return false
}

func (s *Server) handleAdminTasksAPI(w http.ResponseWriter, r *http.Request, parts []string) bool {
	if r.Method == http.MethodGet && len(parts) == 1 {
		writeJSON(w, http.StatusOK, s.store.Snapshot().Tasks)
		return true
	}
	if r.Method == http.MethodPost && len(parts) == 3 && parts[2] == "retry" {
		task, err := s.store.RetryTask(parts[1])
		if err != nil {
			writeError(w, http.StatusBadRequest, err)
			return true
		}
		writeJSON(w, http.StatusCreated, task)
		return true
	}
	return false
}

func (s *Server) handleAdminUpdateSettings(w http.ResponseWriter, r *http.Request) {
	var req settingsRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	if err := s.store.UpdateSettings(req.ProxyUsername, req.ProxyPassword); err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	if _, err := s.syncAllNodeProxyTasks(); err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	s.restartEnabledPoolRuntimes()
	writeJSON(w, http.StatusOK, s.store.Snapshot().Settings)
}

func (s *Server) updatePoolFromJSON(w http.ResponseWriter, r *http.Request, poolID string) (model.Pool, error) {
	defer r.Body.Close()
	var raw map[string]json.RawMessage
	if err := json.NewDecoder(r.Body).Decode(&raw); err != nil {
		return model.Pool{}, err
	}
	var req poolRequest
	b, _ := json.Marshal(raw)
	if err := json.Unmarshal(b, &req); err != nil {
		return model.Pool{}, err
	}
	patch := store.PoolPatch{}
	if _, ok := raw["name"]; ok {
		patch.Name = &req.Name
	}
	if _, ok := raw["groupIds"]; ok {
		patch.GroupIDs = req.GroupIDs
		patch.GroupIDsSet = true
	}
	if _, ok := raw["httpPort"]; ok {
		patch.HTTPPort = &req.HTTPPort
	}
	if _, ok := raw["socksPort"]; ok {
		patch.SocksPort = &req.SocksPort
	}
	if _, ok := raw["strategy"]; ok {
		patch.Strategy = &req.Strategy
	}
	if _, ok := raw["enabled"]; ok {
		patch.Enabled = req.Enabled
	}
	pool, err := s.store.UpdatePool(poolID, patch)
	if err != nil {
		return model.Pool{}, err
	}
	if pool.Enabled {
		_ = s.restartPoolRuntime(pool.ID)
	} else {
		s.stopPoolRuntime(pool.ID)
		_ = s.store.UpdatePoolRuntime(pool.ID, "disabled", "")
	}
	pool, _ = s.store.Pool(pool.ID)
	return pool, nil
}

func (s *Server) createNodeTask(nodeID string, req nodeTaskRequest) (model.Task, error) {
	taskType := normalizeTaskType(req.Type)
	if err := validateTaskRequest(taskType, req.Payload); err != nil {
		return model.Task{}, err
	}
	payload := req.Payload
	if taskType == "sync_node_proxy" || taskType == "update_ports" {
		node, ok := s.store.Node(nodeID)
		if !ok {
			return model.Task{}, fmt.Errorf("node not found")
		}
		httpPort := defaultPositiveInt(req.HTTPPort, defaultPositiveInt(node.HTTPPort, 18080))
		socksPort := defaultPositiveInt(req.SocksPort, defaultPositiveInt(node.SocksPort, 18081))
		egressMode := req.EgressMode
		if egressMode == "" {
			egressMode = node.EgressMode
		}
		egressInterface := req.EgressInterface
		if egressInterface == "" {
			egressInterface = node.EgressInterface
		}
		mode, iface, err := normalizeEgressInput(egressMode, egressInterface)
		if err != nil {
			return model.Task{}, err
		}
		payload, err = s.nodeProxyPayloadJSON(httpPort, socksPort, req.GostVersion, mode, iface)
		if err != nil {
			return model.Task{}, err
		}
		if err := s.store.UpdateNodeProxyConfig(nodeID, httpPort, socksPort, mode, iface); err != nil {
			return model.Task{}, err
		}
		taskType = "sync_node_proxy"
	}
	return s.store.CreateTask(nodeID, taskType, payload)
}

func (s *Server) nodeProxyPayloadJSON(httpPort, socksPort int, gostVersion, egressMode, egressInterface string) (string, error) {
	if httpPort <= 0 && socksPort <= 0 {
		return "", fmt.Errorf("HTTP or SOCKS5 port is required")
	}
	state := s.store.Snapshot()
	if state.Settings.ProxyUsername == "" || state.Settings.ProxyPassword == "" {
		return "", fmt.Errorf("global proxy username/password are required")
	}
	payload := map[string]any{
		"httpPort":        httpPort,
		"socksPort":       socksPort,
		"username":        state.Settings.ProxyUsername,
		"password":        state.Settings.ProxyPassword,
		"gostVersion":     defaultString(gostVersion, "3.2.6"),
		"egressMode":      egressMode,
		"egressInterface": egressInterface,
	}
	b, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func (s *Server) adminState() adminStateResponse {
	state := s.store.Snapshot()
	tokens := make([]registerTokenView, 0, len(state.RegisterTokens))
	for _, t := range state.RegisterTokens {
		tokens = append(tokens, s.registerTokenView(t))
	}
	return adminStateResponse{
		Nodes:          sanitizedNodes(state.Nodes),
		Groups:         state.Groups,
		Pools:          state.Pools,
		RegisterTokens: tokens,
		Tasks:          state.Tasks,
		Settings:       state.Settings,
		BaseURL:        s.cfg.BaseURL,
		Versions:       s.adminVersions(),
		Summary:        summarizeState(state),
	}
}

func (s *Server) registerTokenView(t model.RegisterToken) registerTokenView {
	return registerTokenView{RegisterToken: t, InstallCommand: s.installCommand(t.Token, t.Name)}
}

func (s *Server) adminVersions() adminVersions {
	return adminVersions{Panel: buildinfo.PanelVersion, Agent: buildinfo.AgentVersion}
}

func summarizeState(state model.State) adminSummary {
	summary := adminSummary{TotalNodes: len(state.Nodes)}
	for _, n := range state.Nodes {
		if n.Status == model.NodeStatusOnline {
			summary.OnlineNodes++
		}
		if strings.Contains(strings.ToLower(n.GostStatus), "active") {
			summary.GostActiveNodes++
		}
		if n.GostStatus != "agent uninstalled" && n.AgentVersion != buildinfo.AgentVersion {
			summary.OutdatedAgentNodes++
		}
	}
	for _, p := range state.Pools {
		if p.RuntimeStatus == "running" {
			summary.RunningPools++
		}
	}
	for _, t := range state.Tasks {
		if t.Status == model.TaskStatusFailed {
			summary.FailedTasks++
			if len(summary.RecentErrorTasks) < 5 {
				summary.RecentErrorTasks = append(summary.RecentErrorTasks, t)
			}
		}
	}
	return summary
}

func sanitizedNodes(nodes []model.Node) []model.Node {
	out := make([]model.Node, len(nodes))
	copy(out, nodes)
	for i := range out {
		out[i].AgentToken = ""
	}
	return out
}

func adminPathParts(path string) []string {
	path = strings.TrimPrefix(path, "/api/admin/")
	path = strings.Trim(path, "/")
	if path == "" {
		return nil
	}
	return strings.Split(path, "/")
}

func normalizeTaskType(taskType string) string {
	switch taskType {
	case "update_ports":
		return "update_ports"
	default:
		return strings.TrimSpace(taskType)
	}
}

func validateTaskRequest(taskType, payload string) error {
	switch taskType {
	case "sync_node_proxy", "restart_gost", "update_ports", "upgrade_agent", "uninstall_agent":
		return nil
	case "apply_config":
		if strings.TrimSpace(payload) == "" {
			return fmt.Errorf("payload is required for apply_config")
		}
		return nil
	default:
		return fmt.Errorf("unsupported task type: %s", taskType)
	}
}

func normalizeEgressInput(mode, iface string) (string, string, error) {
	mode = normalizeEgressMode(mode)
	iface = strings.TrimSpace(iface)
	if mode == "custom" && iface == "" {
		return "", "", fmt.Errorf("custom egress requires interface or local IP")
	}
	if mode != "custom" {
		iface = ""
	}
	return mode, iface, nil
}

func defaultPositiveInt(v, fallback int) int {
	if v > 0 {
		return v
	}
	return fallback
}

func uniqueRequestStrings(values []string) []string {
	seen := map[string]bool{}
	var out []string
	for _, v := range values {
		if v == "" || seen[v] {
			continue
		}
		seen[v] = true
		out = append(out, v)
	}
	return out
}
