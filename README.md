# GOST Pool Panel

一个面向多 VPS 的 GOST 代理池管理面板，目标是提供类似哪吒面板的节点接入体验：

- 管理端生成节点一键安装命令。
- Linux 节点执行命令后自动注册上线。
- 面板查看节点、分组、任务和流量。
- 基于节点分组构建 HTTP/SOCKS5 代理池。
- 管理端跨平台运行，节点端仅支持 Linux。

当前是第一版骨架，已经包含管理端、agent、Web 面板和核心 API。

## 当前状态

这个仓库还处在 MVP 阶段，适合上 VPS 调试接入链路：

- 管理端登录、token、节点注册、心跳已经可用。
- 中文面板、节点、分组、代理池配置页已经可用。
- Docker 镜像会内置 Linux `amd64` 和 `arm64` agent，节点一键安装命令可以直接下载。
- GOST v3 的完整配置生成、中心入口进程管理、精确流量统计仍在后续实现中。

## 本地运行

Windows 本地开发：

```powershell
cd F:\WorkSpace\opensource\gost-pool-panel
$env:PANEL_PORT="3000"
$env:PANEL_BASE_URL="http://127.0.0.1:3000"
$env:PANEL_ADMIN_USER="admin"
$env:PANEL_ADMIN_PASSWORD="admin123"
$env:PANEL_SECRET="dev-secret"
go run ./cmd/panel
```

打开：

```text
http://127.0.0.1:3000
```

默认账号：

```text
admin / admin123
```

生产环境必须通过环境变量修改默认密码和 `PANEL_SECRET`。

## 构建

构建管理端：

```powershell
go build -o dist/gost-pool-panel.exe ./cmd/panel
```

构建 Linux agent：

```powershell
$env:GOOS="linux"
$env:GOARCH="amd64"
go build -o dist/gost-pool-agent-linux-amd64 ./cmd/agent

$env:GOARCH="arm64"
go build -o dist/gost-pool-agent-linux-arm64 ./cmd/agent
```

也可以直接运行：

```powershell
.\scripts\build.ps1
```

Linux/macOS：

```bash
sh ./scripts/build.sh
```

## Docker Compose

```bash
cp .env.example .env
docker compose up -d --build
```

环境变量：

| 名称 | 默认值 | 说明 |
| --- | --- | --- |
| `PANEL_PORT` | `3000` | 管理端端口 |
| `PANEL_BASE_URL` | `http://127.0.0.1:3000` | 节点访问管理端的公网地址 |
| `PANEL_ADMIN_USER` | `admin` | 管理员账号 |
| `PANEL_ADMIN_PASSWORD` | `admin123` | 管理员密码 |
| `PANEL_SECRET` | `change-me` | 登录 Cookie 签名密钥 |
| `PANEL_DATA_PATH` | `data/state.json` | 数据文件路径 |

HTTPS 不内置，生产环境请使用 Nginx、Caddy、宝塔等反向代理。

## 发布镜像

仓库内置 GitHub Actions，会在推送 `main`、`master` 或 `v*` tag 时构建并发布 GHCR 镜像：

```text
ghcr.io/YOUR_NAME/gost-pool-panel:main
ghcr.io/YOUR_NAME/gost-pool-panel:v0.1.0
```

发布和 VPS 调试文档：

- [发布指南](docs/RELEASE.md)
- [VPS 调试指南](docs/VPS_DEBUG.md)

## 节点接入流程

1. 登录管理端。
2. 进入“接入命令”。
3. 生成 token。
4. 在 Linux VPS 上执行面板给出的命令。
5. 节点自动注册上线。

节点端安装脚本会：

- 检查当前系统是否为 Linux。
- 下载对应架构的 `gost-pool-agent`。
- 安装到 `/opt/gost-pool-agent`。
- 写入 systemd 服务。
- 启动 agent。

节点端不需要安装 Go、Node.js 或 Python。

## 当前功能

- 管理端登录。
- 节点注册 token。
- 一键安装命令生成。
- agent 注册、心跳、任务轮询、任务结果回传。
- 节点列表。
- 节点分组。
- 代理池配置草案。
- 全局出口代理账号密码。
- 节点流量字段和 API。
- 下发任务骨架。

## 下一步

- 生成真实 GOST v3 配置。
- 管理端中心 GOST 入口进程管理。
- 节点端安装/升级 GOST。
- 节点端端口变更任务落地。
- 远程卸载 agent 的确认和执行流程。
- 更准确的流量统计。
