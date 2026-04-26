package panel

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"

	"gost-pool-panel/internal/gostcfg"
	"gost-pool-panel/internal/model"
)

type poolRuntimeManager struct {
	mu    sync.Mutex
	procs map[string]*exec.Cmd
}

func newPoolRuntimeManager() *poolRuntimeManager {
	return &poolRuntimeManager{procs: map[string]*exec.Cmd{}}
}

func (s *Server) StartPoolRuntimes() {
	state := s.store.Snapshot()
	for _, p := range state.Pools {
		if !p.Enabled {
			continue
		}
		if err := s.restartPoolRuntime(p.ID); err != nil {
			log.Printf("start pool %s failed: %v", p.Name, err)
		}
	}
}

func (s *Server) restartPoolRuntime(poolID string) error {
	state := s.store.Snapshot()
	var pool model.Pool
	found := false
	for _, p := range state.Pools {
		if p.ID == poolID {
			pool = p
			found = true
			break
		}
	}
	if !found {
		return errors.New("pool not found")
	}
	nodes := activePoolNodes(state.Nodes, pool.GroupIDs)
	if len(nodes) == 0 {
		s.stopPoolRuntime(pool.ID)
		_ = s.store.UpdatePoolRuntime(pool.ID, "no active nodes", "proxy pool has no online node with active GOST")
		return errors.New("proxy pool has no online node with active GOST")
	}
	if _, err := exec.LookPath("gost"); err != nil {
		s.stopPoolRuntime(pool.ID)
		_ = s.store.UpdatePoolRuntime(pool.ID, "gost not found", "gost binary not found on panel host/container")
		return err
	}
	path, err := s.writePoolConfig(pool, nodes, state.Settings)
	if err != nil {
		_ = s.store.UpdatePoolRuntime(pool.ID, "config failed", err.Error())
		return err
	}
	s.stopPoolRuntime(pool.ID)
	logPath := filepath.Join(filepath.Dir(s.cfg.DataPath), "pools", pool.ID+".log")
	logFile, err := os.OpenFile(logPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		_ = s.store.UpdatePoolRuntime(pool.ID, "log failed", err.Error())
		return err
	}
	cmd := exec.Command("gost", "-C", path)
	cmd.Stdout = logFile
	cmd.Stderr = logFile
	if err := cmd.Start(); err != nil {
		_ = logFile.Close()
		_ = s.store.UpdatePoolRuntime(pool.ID, "start failed", err.Error())
		return err
	}
	s.poolRuntime.mu.Lock()
	s.poolRuntime.procs[pool.ID] = cmd
	s.poolRuntime.mu.Unlock()
	_ = s.store.UpdatePoolRuntime(pool.ID, "running", "")
	go func() {
		err := cmd.Wait()
		_ = logFile.Close()
		s.poolRuntime.mu.Lock()
		if s.poolRuntime.procs[pool.ID] == cmd {
			delete(s.poolRuntime.procs, pool.ID)
			if err != nil {
				_ = s.store.UpdatePoolRuntime(pool.ID, "exited", err.Error())
			}
		}
		s.poolRuntime.mu.Unlock()
	}()
	return nil
}

func (s *Server) stopPoolRuntime(poolID string) {
	s.poolRuntime.mu.Lock()
	cmd := s.poolRuntime.procs[poolID]
	delete(s.poolRuntime.procs, poolID)
	s.poolRuntime.mu.Unlock()
	if cmd != nil && cmd.Process != nil {
		_ = cmd.Process.Kill()
	}
}

func (s *Server) writePoolConfig(pool model.Pool, nodes []model.Node, settings model.Settings) (string, error) {
	if settings.ProxyUsername == "" || settings.ProxyPassword == "" {
		return "", errors.New("global proxy username/password are required")
	}
	chainName := "chain-" + pool.ID
	auth := &gostcfg.Auth{Username: settings.ProxyUsername, Password: settings.ProxyPassword}
	services := make([]gostcfg.Service, 0, 2)
	if pool.HTTPPort > 0 {
		services = append(services, gostcfg.Service{
			Name: "pool-" + pool.ID + "-http",
			Addr: ":" + fmt.Sprint(pool.HTTPPort),
			Handler: gostcfg.Handler{
				Type:  "http",
				Auth:  auth,
				Chain: chainName,
			},
			Listener: gostcfg.Listener{Type: "tcp"},
		})
	}
	if pool.SocksPort > 0 {
		services = append(services, gostcfg.Service{
			Name: "pool-" + pool.ID + "-socks5",
			Addr: ":" + fmt.Sprint(pool.SocksPort),
			Handler: gostcfg.Handler{
				Type:  "socks5",
				Auth:  auth,
				Chain: chainName,
			},
			Listener: gostcfg.Listener{Type: "tcp"},
		})
	}
	if len(services) == 0 {
		return "", errors.New("pool must expose at least one port")
	}
	gostNodes := make([]gostcfg.Node, 0, len(nodes))
	for _, n := range nodes {
		gostNodes = append(gostNodes, gostcfg.Node{
			Name: "node-" + n.ID,
			Addr: n.PublicIP + ":" + fmt.Sprint(n.HTTPPort),
			Connector: gostcfg.Connector{
				Type: "http",
				Auth: auth,
			},
			Dialer: gostcfg.Dialer{Type: "tcp"},
		})
	}
	selector := gostcfg.Selector{
		Strategy:    normalizeStrategy(pool.Strategy),
		MaxFails:    1,
		FailTimeout: "30s",
	}
	cfg := gostcfg.Config{
		Services: services,
		Chains: []gostcfg.Chain{{
			Name:     chainName,
			Selector: selector,
			Hops: []gostcfg.Hop{{
				Name:     "hop-" + pool.ID,
				Selector: selector,
				Nodes:    gostNodes,
			}},
		}},
	}
	dir := filepath.Join(filepath.Dir(s.cfg.DataPath), "pools")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	path := filepath.Join(dir, pool.ID+".json")
	b, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return "", err
	}
	return path, os.WriteFile(path, b, 0o600)
}

func activePoolNodes(nodes []model.Node, groupIDs []string) []model.Node {
	groupSet := map[string]bool{}
	for _, id := range groupIDs {
		groupSet[id] = true
	}
	var out []model.Node
	for _, n := range nodes {
		if n.Status != model.NodeStatusOnline || n.PublicIP == "" || n.HTTPPort <= 0 {
			continue
		}
		if !strings.Contains(strings.ToLower(n.GostStatus), "active") {
			continue
		}
		for _, gid := range n.GroupIDs {
			if groupSet[gid] {
				out = append(out, n)
				break
			}
		}
	}
	return out
}

func normalizeStrategy(strategy string) string {
	switch strings.ToLower(strategy) {
	case "random", "rand":
		return "rand"
	default:
		return "round"
	}
}
