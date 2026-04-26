# GOST Pool Panel 需求草案

## 目标

开发一个类似哪吒面板节点接入体验的 GOST 代理池管理面板：

- 管理端提供 Web 面板。
- 管理端可以生成一条节点安装命令。
- VPS 节点执行安装命令后自动安装 agent、注册到管理端并保持心跳。
- 面板可以查看节点在线状态、系统信息、GOST 运行状态。
- 面板可以下发 GOST 配置，并让节点自动应用和重启服务。
- 最终为 AI 中转服务提供统一、可管理的 HTTP/SOCKS 出口代理池。
- 管理端可以调整节点本身暴露的端口，可以远程卸载
- 节点可以分组，并可基于分组构建代理池

## 设计原则

- 第一版优先可用，不追求复杂架构。
- 节点端尽量一条命令完成安装。
- 面板端尽量减少手工 SSH 操作。
- agent 和 server 通信走 HTTPS/HTTP API，避免依赖公网 SSH。
- 配置变更必须可追踪、可回滚。
- 默认只管理自己拥有或授权使用的 VPS。
- 节点端必须是 Linux。
- 管理端不限制操作系统，本地 Windows 也需要可以运行和调试。
- 节点端不依赖 Go/Node/Python 等运行时，agent 以预编译单文件二进制方式分发。

## 已确认产品决策

- 第一版使用 GOST v3。
- 管理端需要支持 Docker Compose 一键启动。
- 本地 Windows 开发环境需要可以直接运行和调试。
- 面板界面中文优先。
- 节点端只支持 Linux。
- 管理端不限制操作系统。
- HTTPS 不内置，第一版交给 Nginx/Caddy/宝塔等反向代理处理。
- 代理池入口部署在管理端机器。
- AI 中转出口同时支持 HTTP 代理和 SOCKS5 代理。
- 出口代理需要账号密码认证。
- 出口代理账号密码第一版使用全局一组。
- 需要记录每个节点的流量。
- 节点支持分组，并可以基于分组构建代理池。
- 一个节点可以同时属于多个分组。
- 管理端可以调整节点暴露的代理端口。
- 管理端支持远程卸载节点 agent 和 GOST。
- 节点卸载时保留 GOST 配置备份。
- 流量统计第一版采用最方便且合理的方式实现，允许先用 agent 侧近似统计，后续再接入更精确的 GOST 原生能力。
- 每个代理池可以单独配置 HTTP/SOCKS5 入口端口。
- 节点卸载时不停止、不禁用 GOST systemd 服务，只卸载本项目 agent 并保留 GOST 配置备份。
- 管理端第一版只保留一个管理员账号。

## 核心角色

### 管理端 Server

负责：

- Web 管理面板。
- 节点注册令牌生成。
- 节点列表和状态展示。
- 节点心跳接收。
- GOST 配置模板管理。
- 代理池入口管理。
- 下发节点任务。
- 记录任务执行结果。

### 节点端 Agent

负责：

- 一键安装后自动注册。
- 定时上报心跳、系统信息、GOST 状态。
- 拉取待执行任务。
- 写入 GOST 配置文件。
- 安装、启动、停止、重启 GOST systemd 服务。
- 返回任务执行日志。
- 调整节点本机暴露的 HTTP/SOCKS5 端口。
- 执行远程卸载任务。
- 采集节点侧代理流量统计。

### GOST 进程

负责真实代理能力：

- HTTP 代理。
- SOCKS5 代理。
- 转发链。
- 多上游负载均衡。
- 健康检查和故障切换。

## MVP 功能范围

第一版只做以下能力。

### 1. 管理端安装和启动

- 提供单体服务，默认监听 `3000`。
- 默认使用本地 JSON 或 SQLite 存储。
- 支持通过环境变量配置：
  - `PANEL_BASE_URL`
  - `PANEL_ADMIN_USER`
  - `PANEL_ADMIN_PASSWORD`
  - `PANEL_SECRET`
  - `PANEL_PORT`

### 2. 登录

- 简单账号密码登录。
- Session 或 JWT 均可。
- 第一版不做多管理员权限。

### 3. 节点接入

面板提供“添加节点”页面：

- 生成一次性或长期注册 token。
- 生成 Linux 一键安装命令。
- 命令示例：

```bash
curl -fsSL https://panel.example.com/install.sh | bash -s -- --server https://panel.example.com --token xxx --name node-hk-01
```

节点执行后：

- 下载 agent。
- 安装到 `/opt/gost-pool-agent`。
- 写入配置文件。
- 创建 systemd 服务。
- 启动 agent。
- agent 调用管理端注册接口。

### 4. 节点列表

面板展示：

- 节点名称。
- 公网 IP。
- 地区备注。
- 在线/离线状态。
- 最近心跳时间。
- 系统版本。
- CPU/内存/磁盘基础信息。
- GOST 安装状态。
- GOST 运行状态。
- 当前配置版本。
- 所属分组。
- 节点代理端口。
- 今日流量和总流量。

### 5. 节点详情

节点详情展示：

- 节点基础信息。
- 心跳历史摘要。
- 当前 GOST 配置。
- 最近任务列表。
- 最近错误日志。
- 端口配置。
- 流量统计。

### 5.1 节点分组

面板支持：

- 创建节点分组。
- 修改节点所属分组。
- 一个节点可以属于多个分组。
- 按分组筛选节点。
- 基于一个或多个分组创建代理池。
- 分组维度查看在线节点数和总流量。

### 6. GOST 配置下发

面板支持给单个节点下发配置：

- HTTP 代理监听端口。
- SOCKS5 代理监听端口。
- 用户名密码认证。
- 转发规则。
- 上游代理。
- GOST 配置原文编辑。
- 节点暴露端口调整。

agent 收到任务后：

- 备份旧配置。
- 写入新配置。
- 校验配置格式。
- 重启 GOST。
- 回传执行结果。
- 更新本地端口和认证配置。

### 7. 代理池入口

第一版支持一种简单模式：

- 在中心机运行一个 GOST 入口。
- 入口把一个或多个节点分组作为上游来源。
- 支持 round-robin 或 random 策略。
- 支持 HTTP 和 SOCKS5 两种入口。
- 支持全局账号密码认证。

面板展示：

- 入口监听地址。
- 已加入的节点。
- 已加入的节点分组。
- 负载均衡策略。
- 节点可用状态。
- 入口协议类型。
- 入口认证状态。

### 8. 任务系统

管理端创建任务：

- 安装 GOST。
- 升级 GOST。
- 重启 GOST。
- 下发配置。
- 执行健康检查。
- 修改节点端口。
- 卸载节点。
- 上报流量统计。

agent 轮询任务并上报：

- `pending`
- `running`
- `success`
- `failed`

### 9. 安全要求

- 注册 token 需要有过期时间。
- agent 注册成功后使用节点专属 token。
- agent 请求需要鉴权。
- 管理端敏感字段不能明文展示完整值。
- 安装脚本必须明确展示将安装的路径和服务名。
- 不默认开放无认证代理端口。
- 远程卸载需要二次确认。
- 端口变更需要检测端口占用。
- 节点专属 token 需要支持在管理端重置。

### 10. 流量统计

第一版需要记录：

- 每个节点的今日上传流量。
- 每个节点的今日下载流量。
- 每个节点的总上传流量。
- 每个节点的总下载流量。
- 每个代理池的聚合流量。

第一版可以接受分钟级或心跳级统计，不要求实时精确到账单级。

## 暂不做的功能

以下功能不进入 MVP：

- 多租户。
- 用户订阅系统。
- 计费系统。
- Telegram 通知。
- Kubernetes 部署。
- 复杂 RBAC。
- 自动购买 VPS。
- 绕过第三方平台风控的策略。
- 非授权主机管理。
- 账单级流量计费。

## 建议技术方案

### 方案 A：Node.js 单体

适合快速出第一版。

- Server：Node.js 原生 HTTP 或 Fastify。
- 前端：服务端静态页面 + 原生 JS。
- 存储：SQLite。
- Agent：Shell + Node.js，或 Go 单文件二进制。
- 部署：systemd + Docker 二选一。

优点：

- 开发快。
- 面板和 API 可以一个进程完成。
- 依赖少。

缺点：

- agent 如果用 Node.js，节点端需要安装运行时。

### 方案 B：Go 单体

适合后续正式维护。

- Server：Go + SQLite + HTML template。
- Agent：Go 单文件二进制。
- 前端：少量原生 JS。
- 本地开发：Windows 直接运行管理端。
- 生产部署：Docker Compose 或 systemd。

优点：

- agent 分发简单。
- 适合 VPS 环境。
- 单文件部署。
- 节点端不需要安装 Go 环境。

缺点：

- 初始开发比 Node.js 慢一点。

## 推荐实现路线

建议先用 Go 做，因为节点端体验最接近哪吒：

- `gost-pool-panel`：中心端二进制。
- `gost-pool-agent`：节点端二进制。
- `install.sh`：节点一键安装脚本。
- 节点端只下载预编译 agent，不安装 Go 运行时。
- 开发阶段先支持 Windows 本地运行管理端。
- 生产阶段提供 Docker Compose。

项目结构建议：

```text
gost-pool-panel/
  cmd/
    panel/
    agent/
  internal/
    api/
    auth/
    store/
    task/
    gost/
    agent/
    traffic/
  web/
    static/
    templates/
  scripts/
    install-agent.sh
  data/
  README.md
  REQUIREMENTS.md
```

## API 草案

### 管理端 API

```text
POST /api/admin/login
GET  /api/admin/nodes
GET  /api/admin/nodes/:id
POST /api/admin/register-tokens
GET  /api/admin/install-command
POST /api/admin/nodes/:id/tasks
POST /api/admin/nodes/:id/uninstall
POST /api/admin/nodes/:id/ports
GET  /api/admin/groups
POST /api/admin/groups
POST /api/admin/pools
GET  /api/admin/tasks
```

### 节点端 API

```text
POST /api/agent/register
POST /api/agent/heartbeat
GET  /api/agent/tasks
POST /api/agent/tasks/:id/result
POST /api/agent/traffic
```

## 节点状态字段

```json
{
  "id": "node_xxx",
  "name": "hk-01",
  "publicIp": "1.2.3.4",
  "hostname": "vps-hk-01",
  "os": "ubuntu 22.04",
  "arch": "amd64",
  "status": "online",
  "lastSeenAt": "2026-04-26T10:00:00Z",
  "agentVersion": "0.1.0",
  "gostVersion": "3.x",
  "gostStatus": "running",
  "configVersion": 3,
  "groupIds": ["group_hk"],
  "httpPort": 18080,
  "socksPort": 18081,
  "todayUploadBytes": 123456,
  "todayDownloadBytes": 654321,
  "totalUploadBytes": 12345678,
  "totalDownloadBytes": 87654321
}
```

## 第一版验收标准

- 管理端能启动并打开 Web 面板。
- 面板能生成节点安装命令。
- Linux VPS 执行命令后节点自动出现在面板。
- 节点能每 10-30 秒上报心跳。
- 面板能给节点下发一个 GOST HTTP 代理配置。
- 面板能给节点下发一个 GOST SOCKS5 代理配置。
- 节点能安装/启动/restart GOST。
- 面板能看到任务成功或失败。
- 通过中心入口能按节点分组轮询多个节点作为出口。
- 面板能修改节点暴露端口。
- 面板能远程卸载节点。
- 面板能展示节点流量统计。

## 后续增强

- 节点分组。
- 批量下发配置。
- 节点标签。
- 健康检查。
- 出口 IP 检测。
- 代理可用性检测。
- 流量统计。
- 配置版本回滚。
- 操作审计。
- Docker Compose 部署。
- HTTPS 自动证书。
- WebSocket 实时日志。

## 待确认问题

暂无。当前需求已经足够进入第一版开发。
