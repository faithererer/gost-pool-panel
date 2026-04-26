package panel

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"gost-pool-panel/internal/model"
	"gost-pool-panel/internal/store"
)

func newAdminTestServer(t *testing.T) (*httptest.Server, *store.Store) {
	t.Helper()
	st, err := store.New(filepath.Join(t.TempDir(), "state.json"))
	if err != nil {
		t.Fatal(err)
	}
	srv := NewServer(Config{
		BaseURL:       "http://127.0.0.1:3000",
		AdminUser:     "admin",
		AdminPassword: "admin123",
		Secret:        "test-secret",
		DataPath:      filepath.Join(t.TempDir(), "state.json"),
	}, st)
	return httptest.NewServer(srv.Routes()), st
}

func newAdminClient(t *testing.T) *http.Client {
	t.Helper()
	jar, err := cookiejar.New(nil)
	if err != nil {
		t.Fatal(err)
	}
	return &http.Client{Jar: jar}
}

func doJSON(t *testing.T, client *http.Client, method, url, body string, out any) int {
	t.Helper()
	req, err := http.NewRequest(method, url, bytes.NewBufferString(body))
	if err != nil {
		t.Fatal(err)
	}
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if out != nil {
		if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
			t.Fatal(err)
		}
	}
	return resp.StatusCode
}

func loginAdmin(t *testing.T, client *http.Client, baseURL string) {
	t.Helper()
	var resp map[string]any
	code := doJSON(t, client, http.MethodPost, baseURL+"/api/admin/login", `{"username":"admin","password":"admin123"}`, &resp)
	if code != http.StatusOK {
		t.Fatalf("login status = %d, want 200", code)
	}
}

func TestAdminAPIRequiresJSONAuth(t *testing.T) {
	ts, _ := newAdminTestServer(t)
	defer ts.Close()

	client := newAdminClient(t)
	var resp map[string]string
	code := doJSON(t, client, http.MethodGet, ts.URL+"/api/admin/state", "", &resp)
	if code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", code)
	}
	if resp["error"] == "" {
		t.Fatalf("missing error response: %#v", resp)
	}
}

func TestAdminLoginAndState(t *testing.T) {
	ts, st := newAdminTestServer(t)
	defer ts.Close()
	token, err := st.CreateRegisterToken("node-a", time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	node, err := st.RegisterNode(token.Token, "node-a", "203.0.113.10", "host-a", "linux", "amd64", "0.1.0", "3.2.6", "active")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := st.CreateTask(node.ID, "restart_gost", ""); err != nil {
		t.Fatal(err)
	}

	client := newAdminClient(t)
	loginAdmin(t, client, ts.URL)

	var state adminStateResponse
	code := doJSON(t, client, http.MethodGet, ts.URL+"/api/admin/state", "", &state)
	if code != http.StatusOK {
		t.Fatalf("state status = %d, want 200", code)
	}
	if state.Summary.TotalNodes != 1 || state.Summary.OutdatedAgentNodes != 1 || state.Summary.GostActiveNodes != 1 {
		t.Fatalf("summary = %#v", state.Summary)
	}
	if len(state.Nodes) != 1 || state.Nodes[0].AgentToken != "" {
		t.Fatalf("nodes not sanitized: %#v", state.Nodes)
	}
}

func TestAdminCreateTokenIncludesInstallCommand(t *testing.T) {
	ts, _ := newAdminTestServer(t)
	defer ts.Close()
	client := newAdminClient(t)
	loginAdmin(t, client, ts.URL)

	var token registerTokenView
	code := doJSON(t, client, http.MethodPost, ts.URL+"/api/admin/register-tokens", `{"name":"hk-01","ttlHours":2}`, &token)
	if code != http.StatusCreated {
		t.Fatalf("status = %d, want 201", code)
	}
	if token.Token == "" || token.InstallCommand == "" {
		t.Fatalf("token response = %#v", token)
	}
	var ok map[string]bool
	code = doJSON(t, client, http.MethodDelete, ts.URL+"/api/admin/register-tokens/"+token.Token, "", &ok)
	if code != http.StatusOK || !ok["ok"] {
		t.Fatalf("delete token status = %d body=%#v", code, ok)
	}
}

func TestAdminBatchTaskAndRetry(t *testing.T) {
	ts, st := newAdminTestServer(t)
	defer ts.Close()
	nodeA := registerPanelTestNode(t, st, "node-a")
	nodeB := registerPanelTestNode(t, st, "node-b")
	client := newAdminClient(t)
	loginAdmin(t, client, ts.URL)

	body := `{"nodeIds":["` + nodeA.ID + `","` + nodeB.ID + `"],"type":"upgrade_agent"}`
	var tasks []model.Task
	code := doJSON(t, client, http.MethodPost, ts.URL+"/api/admin/nodes/tasks", body, &tasks)
	if code != http.StatusCreated {
		t.Fatalf("status = %d, want 201", code)
	}
	if len(tasks) != 2 {
		t.Fatalf("tasks = %d, want 2", len(tasks))
	}
	var retry model.Task
	code = doJSON(t, client, http.MethodPost, ts.URL+"/api/admin/tasks/"+tasks[0].ID+"/retry", `{}`, &retry)
	if code != http.StatusCreated {
		t.Fatalf("retry status = %d, want 201", code)
	}
	if retry.Type != "upgrade_agent" || retry.NodeID != tasks[0].NodeID {
		t.Fatalf("retry = %#v", retry)
	}
	var ok map[string]bool
	code = doJSON(t, client, http.MethodDelete, ts.URL+"/api/admin/tasks/"+tasks[1].ID, "", &ok)
	if code != http.StatusOK || !ok["ok"] {
		t.Fatalf("delete task status = %d body=%#v", code, ok)
	}
	if err := st.FinishTask(tasks[0].NodeID, tasks[0].ID, model.TaskStatusSuccess, "ok", ""); err != nil {
		t.Fatal(err)
	}
	var cleanup map[string]any
	code = doJSON(t, client, http.MethodPost, ts.URL+"/api/admin/tasks/cleanup", `{"statuses":["success"]}`, &cleanup)
	if code != http.StatusOK || cleanup["ok"] != true || cleanup["count"].(float64) != 1 {
		t.Fatalf("cleanup status = %d body=%#v", code, cleanup)
	}
}

func TestAdminSyncNodeProxyAcceptsPreferIPv6(t *testing.T) {
	ts, st := newAdminTestServer(t)
	defer ts.Close()
	node := registerPanelTestNode(t, st, "node-a")
	client := newAdminClient(t)
	loginAdmin(t, client, ts.URL)

	body := `{"type":"sync_node_proxy","httpPort":18080,"socksPort":18081,"gostVersion":"3.2.6","egressMode":"prefer_ipv6"}`
	var task model.Task
	code := doJSON(t, client, http.MethodPost, ts.URL+"/api/admin/nodes/"+node.ID+"/tasks", body, &task)
	if code != http.StatusCreated {
		t.Fatalf("status = %d, want 201", code)
	}
	updated, ok := st.Node(node.ID)
	if !ok {
		t.Fatal("node missing")
	}
	if updated.EgressMode != "prefer_ipv6" || updated.EgressInterface != "" {
		t.Fatalf("node egress = %q/%q", updated.EgressMode, updated.EgressInterface)
	}
	var payload map[string]any
	if err := json.Unmarshal([]byte(task.Payload), &payload); err != nil {
		t.Fatal(err)
	}
	if payload["egressMode"] != "prefer_ipv6" {
		t.Fatalf("payload = %#v", payload)
	}
}

func TestAdminGroupPoolAndSettingsAPIs(t *testing.T) {
	ts, st := newAdminTestServer(t)
	defer ts.Close()
	registerPanelTestNode(t, st, "node-a")
	client := newAdminClient(t)
	loginAdmin(t, client, ts.URL)

	var group model.Group
	code := doJSON(t, client, http.MethodPost, ts.URL+"/api/admin/groups", `{"name":"US","remark":"old"}`, &group)
	if code != http.StatusCreated {
		t.Fatalf("create group status = %d, want 201", code)
	}
	var updatedGroup model.Group
	code = doJSON(t, client, http.MethodPatch, ts.URL+"/api/admin/groups/"+group.ID, `{"name":"US Home","remark":"new"}`, &updatedGroup)
	if code != http.StatusOK {
		t.Fatalf("patch group status = %d, want 200", code)
	}
	if updatedGroup.Name != "US Home" || updatedGroup.Remark != "new" {
		t.Fatalf("updated group = %#v", updatedGroup)
	}

	var pool model.Pool
	poolBody := `{"name":"pool","groupIds":["` + group.ID + `"],"httpPort":28080,"socksPort":28081,"strategy":"round"}`
	code = doJSON(t, client, http.MethodPost, ts.URL+"/api/admin/pools", poolBody, &pool)
	if code != http.StatusCreated {
		t.Fatalf("create pool status = %d, want 201", code)
	}
	var disabledPool model.Pool
	code = doJSON(t, client, http.MethodPatch, ts.URL+"/api/admin/pools/"+pool.ID, `{"enabled":false}`, &disabledPool)
	if code != http.StatusOK {
		t.Fatalf("patch pool status = %d, want 200", code)
	}
	if disabledPool.Enabled || disabledPool.RuntimeStatus != "disabled" {
		t.Fatalf("disabled pool = %#v", disabledPool)
	}
	var ok map[string]bool
	code = doJSON(t, client, http.MethodDelete, ts.URL+"/api/admin/pools/"+pool.ID, "", &ok)
	if code != http.StatusOK || !ok["ok"] {
		t.Fatalf("delete pool status = %d body=%#v", code, ok)
	}

	var settings model.Settings
	code = doJSON(t, client, http.MethodPatch, ts.URL+"/api/admin/settings", `{"proxyUsername":"proxy2","proxyPassword":"secret2"}`, &settings)
	if code != http.StatusOK {
		t.Fatalf("settings status = %d, want 200", code)
	}
	if settings.ProxyUsername != "proxy2" || settings.ProxyPassword != "secret2" {
		t.Fatalf("settings = %#v", settings)
	}
	state := st.Snapshot()
	foundSync := false
	for _, task := range state.Tasks {
		if task.Type == "sync_node_proxy" {
			foundSync = true
			break
		}
	}
	if !foundSync {
		t.Fatalf("sync_node_proxy task not created after settings update: %#v", state.Tasks)
	}
}

func registerPanelTestNode(t *testing.T, st *store.Store, name string) model.Node {
	t.Helper()
	token, err := st.CreateRegisterToken(name, time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	node, err := st.RegisterNode(token.Token, name, "203.0.113.20", "host-"+name, "linux", "amd64", "0.3.3", "3.2.6", "active")
	if err != nil {
		t.Fatal(err)
	}
	return node
}
