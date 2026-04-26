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
- 节点端可以通过任务自动安装 GOST v3，并启动带账号密码认证的 HTTP/SOCKS5 代理。
- 管理端可以基于节点分组生成中心 GOST 入口并启动代理池。
- 节点代理出口支持自动、强制 IPv4、强制 IPv6、自定义接口/IP，适合双栈 VPS 选择住宅 IPv6 出口。
- 精确流量统计仍在后续实现中。

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

Docker Compose 默认使用 host network，方便管理端同时暴露面板端口和动态代理池端口。生产环境请在 VPS 防火墙/安全组中只放行你需要的端口。

环境变量：

| 名称 | 默认值 | 说明 |
| --- | --- | --- |
| `PANEL_PORT` | `3000` | 管理端端口 |
| `PANEL_BASE_URL` | `http://127.0.0.1:3000` | 节点访问管理端的公网地址 |
| `PANEL_ADMIN_USER` | `admin` | 管理员账号 |
| `PANEL_ADMIN_PASSWORD` | `admin123` | 管理员密码 |
| `PANEL_SECRET` | `change-me` | 登录 Cookie 签名密钥 |
| `PANEL_PROXY_USERNAME` | `proxy` | 首次初始化时使用的代理入口账号 |
| `PANEL_PROXY_PASSWORD` | 随机生成 | 首次初始化时使用的代理入口密码 |
| `PANEL_DATA_PATH` | `data/state.json` | 数据文件路径 |

HTTPS 不内置，生产环境请使用 Nginx、Caddy、宝塔等反向代理。

如果管理端使用 IPv6 地址，`PANEL_BASE_URL` 需要写成带中括号的 URL，例如：

```text
http://[2600:1700:abcd::1234]:3000
```

## 发布镜像

仓库内置 GitHub Actions。为避免普通提交频繁触发镜像构建，只有推送 `v*` tag 或手动触发 workflow 时才会构建并发布 GHCR 镜像：

```text
ghcr.io/YOUR_NAME/gost-pool-panel:v0.1.0
```

发布和 VPS 调试文档：

- [发布指南](docs/RELEASE.md)
- [VPS 调试指南](docs/VPS_DEBUG.md)
- [前端 API 交接文档](docs/FRONTEND_API.md)

## 节点接入流程

1. 登录管理端。
2. 进入“接入命令”。
3. 生成 token。
4. 在 Linux VPS 上执行面板给出的命令。
5. 节点自动注册上线。
6. 进入“设置”，确认全局出口代理账号密码。
7. 进入“节点”，给节点下发“同步节点代理”任务。
8. 进入“分组”，把节点加入分组。
9. 进入“代理池”，选择分组并配置 HTTP/SOCKS5 入口端口。

面板登录账号和代理入口账号是两套凭据。代理测试请使用“设置”页里的全局出口认证，代理池页面会直接展示当前可用的 `curl` 测试命令，默认使用 `api64.ipify.org` 便于验证双栈出口。

双栈节点可以在“节点”页下发“同步节点代理”任务时选择出口网络：

- `自动`：交给系统路由和 GOST 默认行为。
- `强制 IPv4`：agent 自动选择本机 IPv4 源地址作为 GOST 出口。
- `强制 IPv6`：agent 自动选择本机 IPv6 源地址作为 GOST 出口，并让 GOST 只使用 AAAA 解析结果。
- `自定义接口/IP`：手动填写网卡名或本机 IP，例如 `eth0` 或 `2600:...`。

节点端安装脚本会：

- 检查当前系统是否为 Linux。
- 下载对应架构的 `gost-pool-agent`。
- 安装到 `/opt/gost-pool-agent`。
- 写入 systemd 服务。
- 启动 agent。

节点端不需要安装 Go、Node.js 或 Python。

重复执行一键安装命令会覆盖 agent 二进制并重启 `gost-pool-agent.service`，可用于升级节点端 agent。

## 当前功能

- 管理端登录。
- 节点注册 token。
- 一键安装命令生成。
- agent 注册、心跳、任务轮询、任务结果回传。
- 节点列表。
- 节点分组。
- 节点端 GOST 自动安装和 HTTP/SOCKS5 代理配置。
- 管理端中心 GOST 代理池入口。
- 全局出口代理账号密码。
- 节点流量字段和 API。
- 下发任务骨架。
- 管理端远程升级 agent：agent `0.3.3` 起可以通过“节点”页下发“升级 agent”任务。
- agent `0.3.4` 修复 GOST 安装时 `/tmp` 与 `/usr/local/bin` 跨文件系统导致的 `invalid cross-device link`。
- 远程卸载 agent：任务回报成功后，节点端会删除 `gost-pool-agent.service` 和 `/opt/gost-pool-agent`，不会停止或删除 GOST。
- 节点记录删除：面板支持删除单个节点记录，也支持清理所有 `agent uninstalled` 节点及其任务记录。

## 下一步

- 代理健康检查和自动剔除故障节点。
- 代理池配置编辑/删除。
- 更准确的流量统计。
- 前端框架化和 UI 优化。
