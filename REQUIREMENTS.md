# GOST Pool Panel 产品规格

本文记录当前版本的产品边界、已实现能力和后续增强方向。README 面向使用者，本文面向维护者和贡献者。

## 产品目标

GOST Pool Panel 用于管理多台 VPS 作为代理节点，减少逐台 SSH 安装和维护代理的工作量。

核心目标：

- 管理端提供中文 Web 控制台。
- 管理端生成 Linux 节点一键安装命令。
- 节点执行命令后自动安装 agent、注册到管理端并保持心跳。
- 面板可下发 GOST 代理配置，节点自动安装/重启 GOST。
- 面板可按分组创建 HTTP/SOCKS5 代理池入口。
- 支持直连单节点代理，也支持管理端聚合代理池。
- 支持双栈 VPS 的 IPv4/IPv6 出口选择。

## 设计原则

- 节点端必须是 Linux，管理端不限制操作系统。
- 节点端不依赖 Go、Node.js、Python 等运行时，agent 以预编译单文件二进制分发。
- 常规操作不依赖 SSH：注册、同步代理、升级、卸载都通过面板任务完成。
- HTTPS 不内置，交给 Nginx、Caddy、宝塔等反向代理。
- 第一阶段只保留一个管理员账号。
- 代理认证使用全局一组账号密码。
- 默认只管理自己拥有或授权使用的 VPS。

## 当前架构

```text
cmd/panel
  管理端 Web UI、Admin API、Agent API、代理池运行时

cmd/agent
  Linux 节点端 agent，负责注册、心跳、任务执行、流量统计

internal/store
  JSON 文件状态存储

internal/gostcfg
  GOST v3 配置生成

frontend
  React / Vite / Tailwind 中文管理面板
```

## 已实现能力

### 管理端

- 管理员登录和 session。
- 中文 React Web 面板。
- 节点、分组、代理池、任务、设置页面。
- 全局状态 API：`/api/admin/state`。
- JSON 状态存储：节点、分组、代理池、token、任务、设置。
- Docker Compose 部署。
- GitHub tag 触发 GHCR 镜像构建。

### 节点接入

- 创建注册 token。
- 生成一键安装命令。
- Linux agent 自动安装到 `/opt/gost-pool-agent`。
- systemd 服务：`gost-pool-agent.service`。
- 已注册节点可重复执行安装命令进行原地升级。
- 注册 token 对新节点一次性使用。

### 节点管理

- 节点心跳。
- 节点在线/离线状态。
- 系统信息、架构、agent 版本、GOST 版本和状态。
- 节点代理端口展示。
- 节点所属分组。
- 节点直连代理地址复制。
- 节点直连测试命令复制。
- 删除节点记录。
- 清理已卸载节点记录。

### GOST 配置

- 节点端自动安装 GOST v3。
- 写入 `/etc/gost/gost.json`。
- 创建并重启 `gost.service`。
- 节点侧 HTTP 代理端口。
- 节点侧 SOCKS5 代理端口。
- 代理账号密码认证。
- 同步节点代理任务。
- 重启 GOST 任务。

### 出口网络

支持以下模式：

- `auto`：系统路由和 GOST 默认行为。
- `ipv4`：强制 IPv4 源地址。
- `prefer_ipv6`：DNS 优先 IPv6，允许 IPv4-only 目标回退系统 IPv4。
- `ipv6`：强制 IPv6 源地址，并只解析 AAAA。
- `custom`：自定义接口名或本机 IP。

### 分组

- 创建、编辑、删除分组。
- 节点可加入多个分组。
- 删除分组时自动移除节点和代理池中的引用。

### 代理池

- 基于一个或多个分组创建代理池。
- 管理端运行 GOST 入口。
- 支持 HTTP 入口端口。
- 支持 SOCKS5 入口端口。
- 支持 `round` / `random` 策略。
- 支持启用/禁用。
- 支持编辑、重启、删除。
- 复制带认证的代理池入口地址。
- 复制测试命令。

### 任务系统

支持任务：

- `sync_node_proxy`
- `restart_gost`
- `update_ports`
- `upgrade_agent`
- `uninstall_agent`
- `apply_config`

任务能力：

- 单节点任务。
- 批量任务。
- 任务结果回传。
- 失败任务重试。
- 按状态清理任务。

### 远程升级和卸载

- 管理端远程升级 agent。
- 批量升级过期 agent。
- 远程卸载 agent。
- 卸载只删除本项目 agent 和 systemd 服务，不停止、不禁用、不删除 GOST。

### 流量统计

- agent 基于节点侧 GOST HTTP/SOCKS5 监听端口统计流量。
- 上报今日上传/下载。
- 上报累计上传/下载。
- 今日流量按 UTC 日期切换。
- 统计依赖 Linux `iptables` / `ip6tables` 计数规则。

## 当前限制

- 单管理员账号。
- 无 RBAC。
- 无多租户。
- 无订阅系统。
- 无计费系统。
- 无 HTTPS 自动证书。
- 无主动健康检查和自动剔除故障节点。
- 流量统计用于运维展示，不适合作账单级计费。
- 数据存储为本地 JSON 文件，适合中小规模节点池。

## 安全约束

- 代理入口必须启用账号密码认证。
- 面板登录密码和代理账号密码是两套凭据。
- 生产环境必须修改 `PANEL_ADMIN_PASSWORD` 和 `PANEL_SECRET`。
- 建议安全组只放行必要端口。
- 节点侧代理端口建议只允许管理端 IP 访问，除非你明确要开放直连节点代理。
- HTTPS 需要由外部反向代理提供。

## 验收标准

- 管理端可通过 Docker Compose 启动。
- Web 面板可登录。
- 可创建注册 token，并复制一键安装命令。
- Linux VPS 执行命令后自动出现在面板。
- 节点可自动同步 GOST 代理配置。
- 节点直连 HTTP/SOCKS5 代理可用。
- 可创建分组并把节点加入分组。
- 可创建代理池并通过管理端入口代理出站。
- 可切换 IPv4 / IPv6 优先 / 强制 IPv6 出口模式。
- 可远程升级 agent。
- 可远程卸载 agent。
- 可查看任务结果和失败错误。
- 可查看节点流量统计。

## 后续增强

- 主动健康检查。
- 故障节点自动剔除。
- 代理可用性检测和出口 IP 检测。
- 操作审计。
- 配置版本回滚。
- 更精确的 GOST 原生流量统计。
- SQLite 或其他数据库后端。
- 多管理员和权限控制。
- WebSocket 实时日志。
- 通知渠道，例如 Telegram、Webhook。
