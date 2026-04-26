package panel

import (
	"fmt"
	"net/http"
	"strings"

	"gost-pool-panel/internal/buildinfo"
	"gost-pool-panel/internal/model"
)

type viewData struct {
	State        model.State
	BaseURL      string
	InstallCmds  map[string]string
	PanelVersion string
	AgentVersion string
}

func (s *Server) viewData() viewData {
	state := s.store.Snapshot()
	cmds := map[string]string{}
	for _, t := range state.RegisterTokens {
		cmds[t.Token] = s.installCommand(t.Token, t.Name)
	}
	return viewData{
		State:        state,
		BaseURL:      s.cfg.BaseURL,
		InstallCmds:  cmds,
		PanelVersion: buildinfo.PanelVersion,
		AgentVersion: buildinfo.AgentVersion,
	}
}

func (s *Server) renderLogin(w http.ResponseWriter, msg string) {
	renderTemplate(w, "登录", map[string]string{"Message": msg}, loginHTML)
}

func (s *Server) renderDashboard(w http.ResponseWriter) {
	renderTemplate(w, "概览", s.viewData(), dashboardHTML)
}

func (s *Server) renderNodes(w http.ResponseWriter) {
	renderTemplate(w, "节点", s.viewData(), nodesHTML)
}

func (s *Server) renderTokens(w http.ResponseWriter) {
	renderTemplate(w, "接入命令", s.viewData(), tokensHTML)
}

func (s *Server) renderGroups(w http.ResponseWriter) {
	renderTemplate(w, "分组", s.viewData(), groupsHTML)
}

func (s *Server) renderPools(w http.ResponseWriter) {
	renderTemplate(w, "代理池", s.viewData(), poolsHTML)
}

func (s *Server) renderTasks(w http.ResponseWriter) {
	renderTemplate(w, "任务", s.viewData(), tasksHTML)
}

func (s *Server) renderSettings(w http.ResponseWriter) {
	renderTemplate(w, "设置", s.viewData(), settingsHTML)
}

func groupNames(groups []model.Group, ids []string) string {
	var names []string
	for _, id := range ids {
		for _, g := range groups {
			if g.ID == id {
				names = append(names, g.Name)
				break
			}
		}
	}
	return strings.Join(names, ", ")
}

func poolSummary(p model.Pool) string {
	return fmt.Sprintf("HTTP:%d SOCKS5:%d %s", p.HTTPPort, p.SocksPort, p.Strategy)
}

const baseHTML = `{{define "base"}}
<!doctype html>
<html lang="zh-CN">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>{{.Title}} - GOST Pool Panel</title>
  <style>
    * { box-sizing: border-box; }
    body { margin: 0; font-family: ui-sans-serif, system-ui, -apple-system, BlinkMacSystemFont, "Segoe UI", sans-serif; background: #f6f7f9; color: #1f2937; }
    a { color: #155eef; text-decoration: none; }
    .layout { display: grid; grid-template-columns: 220px minmax(0, 1fr); min-height: 100vh; }
    .sidebar { background: #101828; color: #fff; padding: 20px 14px; }
    .brand { font-size: 18px; font-weight: 700; margin: 0 0 24px; }
    .nav a { display: block; color: #d0d5dd; padding: 10px 12px; border-radius: 8px; margin-bottom: 4px; }
    .nav a:hover { background: #1d2939; color: #fff; }
    .main { padding: 22px; }
    .topbar { display: flex; justify-content: space-between; align-items: center; margin-bottom: 18px; }
    h1 { margin: 0; font-size: 24px; }
    h2 { font-size: 18px; margin: 0 0 12px; }
    .grid { display: grid; grid-template-columns: repeat(auto-fit, minmax(180px, 1fr)); gap: 12px; margin-bottom: 16px; }
    .card { background: #fff; border: 1px solid #e5e7eb; border-radius: 8px; padding: 16px; }
    .metric { font-size: 28px; font-weight: 700; margin-top: 8px; }
    table { width: 100%; border-collapse: collapse; background: #fff; border: 1px solid #e5e7eb; border-radius: 8px; overflow: hidden; }
    th, td { text-align: left; border-bottom: 1px solid #eef0f3; padding: 10px; vertical-align: top; font-size: 14px; }
    th { background: #f9fafb; color: #475467; font-weight: 600; }
    tr:last-child td { border-bottom: 0; }
    input, select, textarea { width: 100%; padding: 9px 10px; border: 1px solid #d0d5dd; border-radius: 6px; font: inherit; background: #fff; }
    textarea { min-height: 82px; resize: vertical; }
    label { display: block; font-size: 13px; color: #475467; margin: 10px 0 6px; }
    .row { display: grid; grid-template-columns: repeat(auto-fit, minmax(180px, 1fr)); gap: 12px; }
    button, .button { display: inline-flex; border: 0; background: #155eef; color: #fff; padding: 9px 12px; border-radius: 6px; font: inherit; cursor: pointer; align-items: center; justify-content: center; }
    .button.secondary, button.secondary { background: #344054; }
    button.danger { background: #b42318; }
    .muted { color: #667085; font-size: 13px; }
    .pill { display: inline-flex; padding: 3px 8px; border-radius: 999px; font-size: 12px; background: #eef4ff; color: #155eef; }
    .online { background: #ecfdf3; color: #067647; }
    .offline { background: #f2f4f7; color: #667085; }
    code, pre { font-family: ui-monospace, SFMono-Regular, Consolas, monospace; }
    pre { white-space: pre-wrap; word-break: break-all; background: #101828; color: #f9fafb; padding: 12px; border-radius: 8px; }
    .checks { display: flex; flex-wrap: wrap; gap: 8px; }
    .checks label { display: inline-flex; gap: 6px; align-items: center; margin: 0; }
    .checks input { width: auto; }
    @media (max-width: 760px) { .layout { grid-template-columns: 1fr; } .sidebar { position: static; } .main { padding: 14px; } }
  </style>
</head>
<body>
  <div class="layout">
    <aside class="sidebar">
      <div class="brand">GOST Pool Panel</div>
      <nav class="nav">
        <a href="/">概览</a>
        <a href="/nodes">节点</a>
        <a href="/tokens">接入命令</a>
        <a href="/groups">分组</a>
        <a href="/pools">代理池</a>
        <a href="/tasks">任务</a>
        <a href="/settings">设置</a>
        <a href="/logout">退出</a>
      </nav>
    </aside>
    <main class="main">
      <div class="topbar"><h1>{{.Title}}</h1><span class="muted">中心地址：{{.Data.BaseURL}} · panel {{.Data.PanelVersion}} · agent {{.Data.AgentVersion}}</span></div>
      {{template "content" .Data}}
    </main>
  </div>
</body>
</html>
{{end}}
{{template "base" .}}`

const loginHTML = `{{define "content"}}
<div class="card" style="max-width:420px;margin:8vh auto;">
  <h2>登录管理端</h2>
  {{with .Message}}<p style="color:#b42318">{{.}}</p>{{end}}
  <form method="post" action="/login">
    <label>账号</label>
    <input name="username" value="admin" autocomplete="username">
    <label>密码</label>
    <input name="password" type="password" autocomplete="current-password">
    <div style="margin-top:14px;"><button type="submit">登录</button></div>
  </form>
  <p class="muted">默认账号 admin，默认密码 admin123。生产环境请用环境变量修改。</p>
</div>
{{end}}`

const dashboardHTML = `{{define "content"}}
<div class="grid">
  <div class="card"><div class="muted">节点数</div><div class="metric">{{len .State.Nodes}}</div></div>
  <div class="card"><div class="muted">分组数</div><div class="metric">{{len .State.Groups}}</div></div>
  <div class="card"><div class="muted">代理池</div><div class="metric">{{len .State.Pools}}</div></div>
  <div class="card"><div class="muted">任务数</div><div class="metric">{{len .State.Tasks}}</div></div>
</div>
<div class="card">
  <h2>下一步</h2>
  <p>先到“接入命令”生成 token，把命令复制到 Linux VPS 执行。节点上线后，到“分组”创建分组，再到“节点”把节点加入分组，最后到“代理池”配置入口端口。</p>
</div>
{{end}}`

const tokensHTML = `{{define "content"}}
<div class="card">
  <h2>生成节点接入命令</h2>
  <form method="post" action="/tokens">
    <div class="row">
      <div><label>节点名称</label><input name="name" placeholder="hk-01"></div>
      <div><label>有效期小时</label><input name="ttl_hours" value="24" type="number"></div>
    </div>
    <div style="margin-top:12px;"><button type="submit">生成 token</button></div>
  </form>
</div>
<div style="height:14px"></div>
<table>
  <thead><tr><th>名称</th><th>状态</th><th>过期时间</th><th>一键命令</th></tr></thead>
  <tbody>
  {{range .State.RegisterTokens}}
    <tr>
      <td>{{.Name}}</td>
      <td>{{if .Used}}已使用{{else}}可用{{end}}</td>
      <td>{{formatTime .ExpiresAt}}</td>
      <td><pre>{{index $.InstallCmds .Token}}</pre></td>
    </tr>
  {{else}}
    <tr><td colspan="4">暂无 token。</td></tr>
  {{end}}
  </tbody>
</table>
{{end}}`

const nodesHTML = `{{define "content"}}
<div class="card" style="margin-bottom:14px;">
  <h2>节点清理</h2>
  <p class="muted">远程卸载成功后的节点会标记为 agent uninstalled。这里可以清掉这些历史节点和它们关联的任务记录。</p>
  <form method="post" action="/nodes/cleanup-uninstalled" onsubmit="return confirm('确认清理所有已卸载节点及其任务记录？');">
    <button class="secondary" type="submit">清理已卸载节点</button>
  </form>
</div>
<table>
  <thead><tr><th>节点</th><th>状态</th><th>系统</th><th>端口</th><th>分组</th><th>流量</th><th>操作</th></tr></thead>
  <tbody>
  {{range .State.Nodes}}
    {{$node := .}}
    <tr>
      <td><strong>{{.Name}}</strong><div class="muted">{{.ID}}<br>{{.PublicIP}} {{.Hostname}}</div></td>
      <td><span class="pill {{.Status}}">{{.Status}}</span><div class="muted">{{formatTime .LastSeenAt}}</div></td>
      <td>{{.OS}}<div class="muted">{{.Arch}} agent {{.AgentVersion}}<br>gost {{gostText .GostVersion .GostStatus}}</div></td>
      <td>HTTP {{.HTTPPort}}<br>SOCKS5 {{.SocksPort}}<div class="muted">出口 {{if .EgressMode}}{{.EgressMode}}{{else}}auto{{end}} {{.EgressInterface}}</div></td>
      <td>
        <form method="post" action="/nodes/groups">
          <input type="hidden" name="node_id" value="{{.ID}}">
          <div class="checks">
          {{range $.State.Groups}}
            <label><input type="checkbox" name="group_id" value="{{.ID}}" {{if contains $node.GroupIDs .ID}}checked{{end}}> {{.Name}}</label>
          {{end}}
          </div>
          <button style="margin-top:8px" class="secondary" type="submit">保存分组</button>
        </form>
      </td>
      <td>今日 {{formatBytes .TodayDownloadBytes}} / {{formatBytes .TodayUploadBytes}}<br><span class="muted">累计 {{formatBytes .TotalDownloadBytes}} / {{formatBytes .TotalUploadBytes}}</span></td>
      <td>
        <form method="post" action="/nodes/tasks">
          <input type="hidden" name="node_id" value="{{.ID}}">
          <label>任务</label>
          <select name="task_type">
            <option value="sync_node_proxy">同步节点代理</option>
            <option value="restart_gost">重启 GOST</option>
            <option value="apply_config">下发配置</option>
            <option value="update_ports">修改端口</option>
            <option value="uninstall_agent">远程卸载 agent</option>
          </select>
          <div class="row">
            <div><label>HTTP 端口</label><input name="http_port" value="{{.HTTPPort}}" type="number"></div>
            <div><label>SOCKS5 端口</label><input name="socks_port" value="{{.SocksPort}}" type="number"></div>
          </div>
          <div class="row">
            <div>
              <label>出口网络</label>
              <select name="egress_mode">
                <option value="auto" {{if or (eq .EgressMode "") (eq .EgressMode "auto")}}selected{{end}}>自动</option>
                <option value="ipv4" {{if eq .EgressMode "ipv4"}}selected{{end}}>强制 IPv4</option>
                <option value="ipv6" {{if eq .EgressMode "ipv6"}}selected{{end}}>强制 IPv6</option>
                <option value="custom" {{if eq .EgressMode "custom"}}selected{{end}}>自定义接口/IP</option>
              </select>
            </div>
            <div><label>自定义接口/IP</label><input name="egress_interface" value="{{.EgressInterface}}" placeholder="eth0 或 2600:..."></div>
          </div>
          <label>GOST 版本</label>
          <input name="gost_version" value="3.2.6">
          <label>Payload</label>
          <textarea name="payload" placeholder="下发配置任务可填写 GOST JSON；同步节点代理任务会自动生成"></textarea>
          <button type="submit">下发</button>
        </form>
        <form method="post" action="/nodes/delete" style="margin-top:10px;" onsubmit="return confirm('确认删除该节点及其任务记录？这不会操作 VPS。');">
          <input type="hidden" name="node_id" value="{{.ID}}">
          <button class="danger" type="submit">删除记录</button>
        </form>
      </td>
    </tr>
  {{else}}
    <tr><td colspan="7">暂无节点，先生成接入命令。</td></tr>
  {{end}}
  </tbody>
</table>
{{end}}`

const groupsHTML = `{{define "content"}}
<div class="card">
  <h2>创建分组</h2>
  <form method="post" action="/groups">
    <div class="row">
      <div><label>名称</label><input name="name" placeholder="香港"></div>
      <div><label>备注</label><input name="remark" placeholder="低延迟出口"></div>
    </div>
    <div style="margin-top:12px;"><button type="submit">创建</button></div>
  </form>
</div>
<div style="height:14px"></div>
<table>
  <thead><tr><th>名称</th><th>备注</th><th>创建时间</th></tr></thead>
  <tbody>{{range .State.Groups}}<tr><td>{{.Name}}</td><td>{{.Remark}}</td><td>{{formatTime .CreatedAt}}</td></tr>{{else}}<tr><td colspan="3">暂无分组。</td></tr>{{end}}</tbody>
</table>
{{end}}`

const poolsHTML = `{{define "content"}}
<div class="card">
  <h2>创建代理池</h2>
  <form method="post" action="/pools">
    <div class="row">
      <div><label>名称</label><input name="name" placeholder="ai-pool"></div>
      <div><label>HTTP 入口端口</label><input name="http_port" value="28080" type="number"></div>
      <div><label>SOCKS5 入口端口</label><input name="socks_port" value="28081" type="number"></div>
      <div><label>策略</label><select name="strategy"><option value="round">轮询</option><option value="random">随机</option></select></div>
    </div>
    <label>包含分组</label>
    <div class="checks">{{range .State.Groups}}<label><input type="checkbox" name="group_id" value="{{.ID}}"> {{.Name}}</label>{{else}}<span class="muted">还没有分组。</span>{{end}}</div>
    <div style="margin-top:12px;"><button type="submit">创建代理池</button></div>
  </form>
</div>
<div style="height:14px"></div>
<table>
  <thead><tr><th>名称</th><th>端口</th><th>分组 ID</th><th>状态</th><th>测试命令</th><th>操作</th></tr></thead>
  <tbody>
  {{$auth := shellQuote (userPass .State.Settings)}}
  {{range .State.Pools}}
    <tr>
      <td>{{.Name}}</td>
      <td>HTTP {{.HTTPPort}}<br>SOCKS5 {{.SocksPort}}</td>
      <td>{{join .GroupIDs ", "}}</td>
      <td>{{if .Enabled}}启用{{else}}停用{{end}}<div class="muted">{{.RuntimeStatus}} {{.RuntimeError}}</div></td>
      <td>
        {{if gt .HTTPPort 0}}<pre>curl -x http://{{proxyAddr $.BaseURL .HTTPPort}} -U {{$auth}} https://api.ipify.org</pre>{{end}}
        {{if gt .SocksPort 0}}<pre>curl -x socks5h://{{proxyAddr $.BaseURL .SocksPort}} -U {{$auth}} https://api.ipify.org</pre>{{end}}
      </td>
      <td><form method="post" action="/pools/restart"><input type="hidden" name="pool_id" value="{{.ID}}"><button type="submit">重启入口</button></form></td>
    </tr>
  {{else}}
    <tr><td colspan="6">暂无代理池。</td></tr>
  {{end}}
  </tbody>
</table>
{{end}}`

const tasksHTML = `{{define "content"}}
<table>
  <thead><tr><th>任务</th><th>节点</th><th>状态</th><th>Payload</th><th>结果</th><th>时间</th></tr></thead>
  <tbody>
  {{range .State.Tasks}}
    <tr><td>{{.Type}}<div class="muted">{{.ID}}</div></td><td>{{.NodeID}}</td><td>{{.Status}}</td><td><pre>{{.Payload}}</pre></td><td>{{.Result}}<div style="color:#b42318">{{.Error}}</div></td><td>{{formatTime .CreatedAt}}<br>{{formatTime .FinishedAt}}</td></tr>
  {{else}}
    <tr><td colspan="6">暂无任务。</td></tr>
  {{end}}
  </tbody>
</table>
{{end}}`

const settingsHTML = `{{define "content"}}
<div class="card">
  <h2>全局出口认证</h2>
  <form method="post" action="/settings">
    <div class="row">
      <div><label>用户名</label><input name="proxy_username" value="{{.State.Settings.ProxyUsername}}"></div>
      <div><label>密码</label><input name="proxy_password" value="{{.State.Settings.ProxyPassword}}"></div>
    </div>
    <p class="muted">这是代理入口账号密码，不是面板登录账号。保存后会自动给节点下发同步任务，并重启已启用代理池入口。</p>
    <button type="submit">保存</button>
  </form>
</div>
{{end}}`

const installScript = `#!/usr/bin/env bash
set -euo pipefail

SERVER="{{BASE_URL}}"
TOKEN=""
NAME=""
INSTALL_DIR="/opt/gost-pool-agent"
SERVICE_NAME="gost-pool-agent"

while [[ $# -gt 0 ]]; do
  case "$1" in
    --server) SERVER="$2"; shift 2 ;;
    --token) TOKEN="$2"; shift 2 ;;
    --name) NAME="$2"; shift 2 ;;
    *) echo "Unknown argument: $1"; exit 1 ;;
  esac
done

if [[ "$(uname -s)" != "Linux" ]]; then
  echo "This agent only supports Linux nodes."
  exit 1
fi

ARCH="$(uname -m)"
case "$ARCH" in
  x86_64|amd64) BIN="gost-pool-agent-linux-amd64" ;;
  aarch64|arm64) BIN="gost-pool-agent-linux-arm64" ;;
  *) echo "Unsupported arch: $ARCH"; exit 1 ;;
esac

if [[ "$EUID" -ne 0 ]]; then
  echo "Please run as root."
  exit 1
fi

mkdir -p "$INSTALL_DIR"
EXISTING_AGENT=0
if [[ -s "$INSTALL_DIR/agent.json" ]] && grep -q '"nodeId"' "$INSTALL_DIR/agent.json" && grep -q '"agentToken"' "$INSTALL_DIR/agent.json"; then
  EXISTING_AGENT=1
fi

if [[ -z "$TOKEN" && "$EXISTING_AGENT" != "1" ]]; then
  echo "--token is required"
  exit 1
fi

if [[ "$EXISTING_AGENT" == "1" ]]; then
  echo "Existing agent identity found; skipping register token check for in-place upgrade."
else
  echo "Checking register token"
  TOKEN_CHECK="$(curl -sS -w '\n%{http_code}' "$SERVER/api/agent/register-token/check?token=$TOKEN")"
  TOKEN_CODE="$(printf '%s\n' "$TOKEN_CHECK" | tail -n 1)"
  TOKEN_BODY="$(printf '%s\n' "$TOKEN_CHECK" | sed '$d')"
  if [[ "$TOKEN_CODE" != "200" ]]; then
    echo "Register token is not available. Generate a new token in the panel."
    echo "$TOKEN_BODY"
    exit 1
  fi
fi

echo "Downloading agent from $SERVER/downloads/$BIN"
TMP_AGENT="$(mktemp "$INSTALL_DIR/gost-pool-agent.XXXXXX")"
cleanup_tmp_agent() {
  rm -f "$TMP_AGENT"
}
trap cleanup_tmp_agent EXIT
curl -fsSL "$SERVER/downloads/$BIN" -o "$TMP_AGENT"
chmod +x "$TMP_AGENT"
echo "Downloaded agent version: $("$TMP_AGENT" --version 2>/dev/null || echo unknown)"
mv -f "$TMP_AGENT" "$INSTALL_DIR/gost-pool-agent"
trap - EXIT

cat > "$INSTALL_DIR/agent.env" <<EOF
GPP_SERVER=$SERVER
GPP_REGISTER_TOKEN=$TOKEN
GPP_NODE_NAME=$NAME
GPP_CONFIG=$INSTALL_DIR/agent.json
EOF

cat > "/etc/systemd/system/$SERVICE_NAME.service" <<EOF
[Unit]
Description=GOST Pool Agent
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
EnvironmentFile=$INSTALL_DIR/agent.env
ExecStart=$INSTALL_DIR/gost-pool-agent --server \$GPP_SERVER --token \$GPP_REGISTER_TOKEN --name "\$GPP_NODE_NAME" --config \$GPP_CONFIG
Restart=always
RestartSec=5

[Install]
WantedBy=multi-user.target
EOF

systemctl daemon-reload
systemctl enable "$SERVICE_NAME"
systemctl restart "$SERVICE_NAME"
echo "Agent installed. Check status: systemctl status $SERVICE_NAME"
`
