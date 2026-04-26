package store

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net"
	"os"
	"path/filepath"
	"sync"
	"time"

	"gost-pool-panel/internal/model"
)

type Store struct {
	mu    sync.Mutex
	path  string
	state model.State
}

func New(path string) (*Store, error) {
	s := &Store{path: path}
	if err := s.load(); err != nil {
		return nil, err
	}
	return s, nil
}

func (s *Store) load() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if err := os.MkdirAll(filepath.Dir(s.path), 0o755); err != nil {
		return err
	}
	b, err := os.ReadFile(s.path)
	if errors.Is(err, os.ErrNotExist) {
		s.state = model.State{
			Settings: model.Settings{
				ProxyUsername: defaultProxyUsername(),
				ProxyPassword: defaultProxyPassword(),
			},
		}
		return s.saveLocked()
	}
	if err != nil {
		return err
	}
	if len(b) == 0 {
		s.state = model.State{}
		s.ensureDefaultsLocked()
		return s.saveLocked()
	}
	if err := json.Unmarshal(b, &s.state); err != nil {
		return err
	}
	s.ensureDefaultsLocked()
	return nil
}

func (s *Store) Save() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.saveLocked()
}

func (s *Store) saveLocked() error {
	if err := os.MkdirAll(filepath.Dir(s.path), 0o755); err != nil {
		return err
	}
	b, err := json.MarshalIndent(s.state, "", "  ")
	if err != nil {
		return err
	}
	tmp := s.path + ".tmp"
	if err := os.WriteFile(tmp, b, 0o600); err != nil {
		return err
	}
	_ = os.Remove(s.path)
	return os.Rename(tmp, s.path)
}

func (s *Store) ensureDefaultsLocked() {
	if s.state.Settings.ProxyUsername == "" {
		s.state.Settings.ProxyUsername = defaultProxyUsername()
	}
	if s.state.Settings.ProxyPassword == "" {
		s.state.Settings.ProxyPassword = defaultProxyPassword()
	}
}

func (s *Store) Snapshot() model.State {
	s.mu.Lock()
	defer s.mu.Unlock()
	return cloneState(s.state)
}

func (s *Store) CreateRegisterToken(name string, ttl time.Duration) (model.RegisterToken, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now().UTC()
	t := model.RegisterToken{
		Token:     randomID("rt"),
		Name:      name,
		ExpiresAt: now.Add(ttl),
		CreatedAt: now,
	}
	s.state.RegisterTokens = append([]model.RegisterToken{t}, s.state.RegisterTokens...)
	return t, s.saveLocked()
}

func (s *Store) CheckRegisterToken(token string) (int, string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now().UTC()
	for i := range s.state.RegisterTokens {
		t := &s.state.RegisterTokens[i]
		if t.Token != token {
			continue
		}
		if t.Used {
			return 409, "register token already used"
		}
		if now.After(t.ExpiresAt) {
			return 410, "register token expired"
		}
		return 200, "register token available"
	}
	return 404, "invalid register token"
}

func (s *Store) RegisterNode(registerToken, name, publicIP, hostname, osName, arch, agentVersion, gostVersion, gostStatus string) (model.Node, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now().UTC()
	tokenIndex := -1
	for i := range s.state.RegisterTokens {
		t := &s.state.RegisterTokens[i]
		if t.Token == registerToken {
			if t.Used {
				return model.Node{}, errors.New("register token already used")
			}
			if now.After(t.ExpiresAt) {
				return model.Node{}, errors.New("register token expired")
			}
			tokenIndex = i
			break
		}
	}
	if tokenIndex < 0 {
		return model.Node{}, errors.New("invalid register token")
	}
	if name == "" {
		name = s.state.RegisterTokens[tokenIndex].Name
	}
	if name == "" {
		name = hostname
	}
	if name == "" {
		name = "linux-node"
	}
	node := model.Node{
		ID:           randomID("node"),
		Name:         name,
		PublicIP:     publicIP,
		Hostname:     hostname,
		OS:           osName,
		Arch:         arch,
		Status:       model.NodeStatusOnline,
		LastSeenAt:   now,
		AgentToken:   randomID("agent"),
		AgentVersion: agentVersion,
		GostVersion:  gostVersion,
		GostStatus:   gostStatus,
		HTTPPort:     18080,
		SocksPort:    18081,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	s.state.RegisterTokens[tokenIndex].Used = true
	s.state.Nodes = append([]model.Node{node}, s.state.Nodes...)
	return node, s.saveLocked()
}

func (s *Store) CreateTaskFromPayload(nodeID, taskType string, payload any) (model.Task, error) {
	b, err := json.Marshal(payload)
	if err != nil {
		return model.Task{}, err
	}
	return s.CreateTask(nodeID, taskType, string(b))
}

func (s *Store) AuthenticateAgent(nodeID, token string) (model.Node, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, n := range s.state.Nodes {
		if n.ID == nodeID && n.AgentToken == token {
			return n, true
		}
	}
	return model.Node{}, false
}

func (s *Store) Heartbeat(nodeID string, patch model.Node) (model.Node, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now().UTC()
	for i := range s.state.Nodes {
		n := &s.state.Nodes[i]
		if n.ID != nodeID {
			continue
		}
		if patch.PublicIP != "" {
			n.PublicIP = patch.PublicIP
		}
		if patch.Hostname != "" {
			n.Hostname = patch.Hostname
		}
		if patch.OS != "" {
			n.OS = patch.OS
		}
		if patch.Arch != "" {
			n.Arch = patch.Arch
		}
		if patch.AgentVersion != "" {
			n.AgentVersion = patch.AgentVersion
		}
		if patch.GostVersion != "" {
			n.GostVersion = patch.GostVersion
		}
		if patch.GostStatus != "" {
			n.GostStatus = patch.GostStatus
		}
		n.Status = model.NodeStatusOnline
		n.LastSeenAt = now
		n.UpdatedAt = now
		if err := s.saveLocked(); err != nil {
			return model.Node{}, err
		}
		return *n, nil
	}
	return model.Node{}, errors.New("node not found")
}

func (s *Store) AddTraffic(nodeID string, upload, download int64) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i := range s.state.Nodes {
		n := &s.state.Nodes[i]
		if n.ID == nodeID {
			n.TodayUploadBytes += upload
			n.TodayDownloadBytes += download
			n.TotalUploadBytes += upload
			n.TotalDownloadBytes += download
			n.UpdatedAt = time.Now().UTC()
			return s.saveLocked()
		}
	}
	return errors.New("node not found")
}

func (s *Store) CreateGroup(name, remark string) (model.Group, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now().UTC()
	g := model.Group{ID: randomID("group"), Name: name, Remark: remark, CreatedAt: now, UpdatedAt: now}
	s.state.Groups = append(s.state.Groups, g)
	return g, s.saveLocked()
}

func (s *Store) AssignGroups(nodeID string, groupIDs []string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i := range s.state.Nodes {
		if s.state.Nodes[i].ID == nodeID {
			s.state.Nodes[i].GroupIDs = uniqueStrings(groupIDs)
			s.state.Nodes[i].UpdatedAt = time.Now().UTC()
			return s.saveLocked()
		}
	}
	return errors.New("node not found")
}

func (s *Store) UpdateNodePorts(nodeID string, httpPort, socksPort int) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i := range s.state.Nodes {
		if s.state.Nodes[i].ID == nodeID {
			if httpPort > 0 {
				s.state.Nodes[i].HTTPPort = httpPort
			}
			if socksPort > 0 {
				s.state.Nodes[i].SocksPort = socksPort
			}
			s.state.Nodes[i].ConfigVersion++
			s.state.Nodes[i].UpdatedAt = time.Now().UTC()
			return s.saveLocked()
		}
	}
	return errors.New("node not found")
}

func (s *Store) DeleteNode(nodeID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	found := false
	nodes := s.state.Nodes[:0]
	for _, n := range s.state.Nodes {
		if n.ID == nodeID {
			found = true
			continue
		}
		nodes = append(nodes, n)
	}
	if !found {
		return errors.New("node not found")
	}
	s.state.Nodes = nodes
	s.state.Tasks = filterTasks(s.state.Tasks, map[string]bool{nodeID: true})
	return s.saveLocked()
}

func (s *Store) DeleteUninstalledNodes() (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	removed := map[string]bool{}
	nodes := s.state.Nodes[:0]
	for _, n := range s.state.Nodes {
		if n.GostStatus == "agent uninstalled" {
			removed[n.ID] = true
			continue
		}
		nodes = append(nodes, n)
	}
	if len(removed) == 0 {
		return 0, nil
	}
	s.state.Nodes = nodes
	s.state.Tasks = filterTasks(s.state.Tasks, removed)
	return len(removed), s.saveLocked()
}

func (s *Store) CreatePool(name string, groupIDs []string, httpPort, socksPort int, strategy string) (model.Pool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now().UTC()
	if strategy == "" {
		strategy = "round"
	}
	p := model.Pool{
		ID:            randomID("pool"),
		Name:          name,
		GroupIDs:      uniqueStrings(groupIDs),
		HTTPPort:      httpPort,
		SocksPort:     socksPort,
		Strategy:      strategy,
		Enabled:       true,
		RuntimeStatus: "created",
		CreatedAt:     now,
		UpdatedAt:     now,
	}
	s.state.Pools = append(s.state.Pools, p)
	return p, s.saveLocked()
}

func (s *Store) UpdatePoolRuntime(poolID, status, errText string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now().UTC()
	for i := range s.state.Pools {
		if s.state.Pools[i].ID == poolID {
			s.state.Pools[i].RuntimeStatus = status
			s.state.Pools[i].RuntimeError = errText
			if status == "running" {
				s.state.Pools[i].StartedAt = now
			}
			s.state.Pools[i].UpdatedAt = now
			return s.saveLocked()
		}
	}
	return errors.New("pool not found")
}

func (s *Store) CreateTask(nodeID, taskType, payload string) (model.Task, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now().UTC()
	t := model.Task{
		ID:        randomID("task"),
		NodeID:    nodeID,
		Type:      taskType,
		Status:    model.TaskStatusPending,
		Payload:   payload,
		CreatedAt: now,
		UpdatedAt: now,
	}
	s.state.Tasks = append([]model.Task{t}, s.state.Tasks...)
	return t, s.saveLocked()
}

func (s *Store) PendingTasks(nodeID string) ([]model.Task, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now().UTC()
	var tasks []model.Task
	for i := range s.state.Tasks {
		t := &s.state.Tasks[i]
		if t.NodeID == nodeID && t.Status == model.TaskStatusPending {
			t.Status = model.TaskStatusRunning
			t.StartedAt = now
			t.UpdatedAt = now
			tasks = append(tasks, *t)
		}
	}
	if len(tasks) == 0 {
		return tasks, nil
	}
	return tasks, s.saveLocked()
}

func (s *Store) FinishTask(nodeID, taskID, status, result, errText string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now().UTC()
	for i := range s.state.Tasks {
		t := &s.state.Tasks[i]
		if t.ID == taskID && t.NodeID == nodeID {
			t.Status = status
			t.Result = result
			t.Error = errText
			t.FinishedAt = now
			t.UpdatedAt = now
			if t.Type == "uninstall_agent" && status == model.TaskStatusSuccess {
				for j := range s.state.Nodes {
					if s.state.Nodes[j].ID == nodeID {
						s.state.Nodes[j].Status = model.NodeStatusOffline
						s.state.Nodes[j].GostStatus = "agent uninstalled"
						s.state.Nodes[j].UpdatedAt = now
						break
					}
				}
			}
			return s.saveLocked()
		}
	}
	return errors.New("task not found")
}

func (s *Store) UpdateSettings(username, password string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if username != "" {
		s.state.Settings.ProxyUsername = username
	}
	if password != "" {
		s.state.Settings.ProxyPassword = password
	}
	return s.saveLocked()
}

func PublicIPFromAddr(addr string) string {
	host, _, err := net.SplitHostPort(addr)
	if err != nil {
		return addr
	}
	return host
}

func randomID(prefix string) string {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return prefix + "_" + time.Now().UTC().Format("20060102150405")
	}
	return prefix + "_" + hex.EncodeToString(b[:])
}

func defaultProxyUsername() string {
	if v := os.Getenv("PANEL_PROXY_USERNAME"); v != "" {
		return v
	}
	return "proxy"
}

func defaultProxyPassword() string {
	if v := os.Getenv("PANEL_PROXY_PASSWORD"); v != "" {
		return v
	}
	return randomID("pass")
}

func cloneState(in model.State) model.State {
	b, _ := json.Marshal(in)
	var out model.State
	_ = json.Unmarshal(b, &out)
	return out
}

func uniqueStrings(values []string) []string {
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

func filterTasks(tasks []model.Task, removedNodeIDs map[string]bool) []model.Task {
	out := tasks[:0]
	for _, t := range tasks {
		if removedNodeIDs[t.NodeID] {
			continue
		}
		out = append(out, t)
	}
	return out
}
