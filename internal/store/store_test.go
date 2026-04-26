package store

import (
	"path/filepath"
	"testing"
	"time"

	"gost-pool-panel/internal/model"
)

func newTestStore(t *testing.T) *Store {
	t.Helper()
	st, err := New(filepath.Join(t.TempDir(), "state.json"))
	if err != nil {
		t.Fatal(err)
	}
	return st
}

func registerTestNode(t *testing.T, st *Store, name string) model.Node {
	t.Helper()
	token, err := st.CreateRegisterToken(name, time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	node, err := st.RegisterNode(token.Token, name, "203.0.113.10", "host-"+name, "linux", "amd64", "0.3.3", "3.2.6", "active")
	if err != nil {
		t.Fatal(err)
	}
	return node
}

func TestDeleteGroupRemovesReferences(t *testing.T) {
	st := newTestStore(t)
	node := registerTestNode(t, st, "node-a")
	group, err := st.CreateGroup("us", "remark")
	if err != nil {
		t.Fatal(err)
	}
	if err := st.AssignGroups(node.ID, []string{group.ID}); err != nil {
		t.Fatal(err)
	}
	pool, err := st.CreatePool("pool", []string{group.ID}, 28080, 28081, "round")
	if err != nil {
		t.Fatal(err)
	}
	if err := st.DeleteGroup(group.ID); err != nil {
		t.Fatal(err)
	}
	updatedNode, ok := st.Node(node.ID)
	if !ok {
		t.Fatal("node missing")
	}
	if len(updatedNode.GroupIDs) != 0 {
		t.Fatalf("node group refs = %#v, want empty", updatedNode.GroupIDs)
	}
	updatedPool, ok := st.Pool(pool.ID)
	if !ok {
		t.Fatal("pool missing")
	}
	if len(updatedPool.GroupIDs) != 0 {
		t.Fatalf("pool group refs = %#v, want empty", updatedPool.GroupIDs)
	}
}

func TestUpdateAndDeletePool(t *testing.T) {
	st := newTestStore(t)
	pool, err := st.CreatePool("pool", nil, 28080, 28081, "round")
	if err != nil {
		t.Fatal(err)
	}
	name := "pool-new"
	httpPort := 38080
	enabled := false
	updated, err := st.UpdatePool(pool.ID, PoolPatch{Name: &name, HTTPPort: &httpPort, Enabled: &enabled})
	if err != nil {
		t.Fatal(err)
	}
	if updated.Name != name || updated.HTTPPort != httpPort || updated.Enabled {
		t.Fatalf("updated pool = %#v", updated)
	}
	if err := st.DeletePool(pool.ID); err != nil {
		t.Fatal(err)
	}
	if _, ok := st.Pool(pool.ID); ok {
		t.Fatal("pool still exists after delete")
	}
}

func TestCreateTasksAndRetryTask(t *testing.T) {
	st := newTestStore(t)
	nodeA := registerTestNode(t, st, "node-a")
	nodeB := registerTestNode(t, st, "node-b")
	tasks, err := st.CreateTasks([]string{nodeA.ID, nodeB.ID}, "upgrade_agent", "")
	if err != nil {
		t.Fatal(err)
	}
	if len(tasks) != 2 {
		t.Fatalf("tasks = %d, want 2", len(tasks))
	}
	retry, err := st.RetryTask(tasks[0].ID)
	if err != nil {
		t.Fatal(err)
	}
	if retry.Type != tasks[0].Type || retry.NodeID != tasks[0].NodeID || retry.Status != model.TaskStatusPending {
		t.Fatalf("retry task = %#v", retry)
	}
}

func TestDeleteRegisterTokenAndTasks(t *testing.T) {
	st := newTestStore(t)
	token, err := st.CreateRegisterToken("node-a", time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	if err := st.DeleteRegisterToken(token.Token); err != nil {
		t.Fatal(err)
	}
	if got := st.Snapshot().RegisterTokens; len(got) != 0 {
		t.Fatalf("tokens = %#v, want empty", got)
	}

	node := registerTestNode(t, st, "node-b")
	task, err := st.CreateTask(node.ID, "restart_gost", "")
	if err != nil {
		t.Fatal(err)
	}
	if err := st.DeleteTask(task.ID); err != nil {
		t.Fatal(err)
	}
	if got := st.Snapshot().Tasks; len(got) != 0 {
		t.Fatalf("tasks = %#v, want empty", got)
	}

	task, err = st.CreateTask(node.ID, "restart_gost", "")
	if err != nil {
		t.Fatal(err)
	}
	if err := st.FinishTask(node.ID, task.ID, model.TaskStatusSuccess, "ok", ""); err != nil {
		t.Fatal(err)
	}
	count, err := st.DeleteTasksByStatus([]string{model.TaskStatusSuccess})
	if err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Fatalf("deleted tasks = %d, want 1", count)
	}
}

func TestAddTrafficAccumulatesAndRejectsNegative(t *testing.T) {
	st := newTestStore(t)
	node := registerTestNode(t, st, "node-a")

	if err := st.AddTraffic(node.ID, 100, 200); err != nil {
		t.Fatal(err)
	}
	if err := st.AddTraffic(node.ID, 50, 25); err != nil {
		t.Fatal(err)
	}
	updated, ok := st.Node(node.ID)
	if !ok {
		t.Fatal("node missing")
	}
	if updated.TodayUploadBytes != 150 || updated.TodayDownloadBytes != 225 {
		t.Fatalf("today traffic = %d/%d", updated.TodayUploadBytes, updated.TodayDownloadBytes)
	}
	if updated.TotalUploadBytes != 150 || updated.TotalDownloadBytes != 225 {
		t.Fatalf("total traffic = %d/%d", updated.TotalUploadBytes, updated.TotalDownloadBytes)
	}
	if updated.TrafficDate != time.Now().UTC().Format("2006-01-02") {
		t.Fatalf("traffic date = %q", updated.TrafficDate)
	}

	st.mu.Lock()
	for i := range st.state.Nodes {
		if st.state.Nodes[i].ID == node.ID {
			st.state.Nodes[i].TrafficDate = "2000-01-01"
			st.state.Nodes[i].TodayUploadBytes = 999
			st.state.Nodes[i].TodayDownloadBytes = 999
		}
	}
	st.mu.Unlock()
	if err := st.AddTraffic(node.ID, 10, 20); err != nil {
		t.Fatal(err)
	}
	updated, ok = st.Node(node.ID)
	if !ok {
		t.Fatal("node missing")
	}
	if updated.TodayUploadBytes != 10 || updated.TodayDownloadBytes != 20 {
		t.Fatalf("reset today traffic = %d/%d", updated.TodayUploadBytes, updated.TodayDownloadBytes)
	}
	if updated.TotalUploadBytes != 160 || updated.TotalDownloadBytes != 245 {
		t.Fatalf("total after reset = %d/%d", updated.TotalUploadBytes, updated.TotalDownloadBytes)
	}
	if err := st.AddTraffic(node.ID, -1, 0); err == nil {
		t.Fatal("negative upload accepted")
	}
}
