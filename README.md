# GOST Pool Panel

GOST Pool Panel 是一个面向多台 VPS 的代理池管理面板。管理端负责 Web 控制台、节点接入、任务下发、分组和代理池入口；节点端运行 Linux agent，自动注册、安装 GOST、同步代理配置并上报状态。

它的目标是把“逐台 VPS SSH 搭代理”的流程收敛成面板操作：生成一键命令，节点上线，分组，创建 HTTP/SOCKS5 代理池，然后复制带认证的代理地址或测试命令直接使用。

## 功能概览

- 中文 Web 管理面板。
- 一次性注册 token 和 Linux 一键安装命令。
- Linux 节点自动注册、心跳、任务轮询和结果回传。
- 节点端自动安装 GOST v3，启动 HTTP/SOCKS5 代理。
- 节点分组，一个节点可加入多个分组。
- 管理端代理池入口，按分组聚合节点出口，支持 HTTP/SOCKS5。
- 全局一组代理认证账号密码。
- 节点直连代理地址和代理池入口地址复制，默认包含认证信息。
- 出口网络模式：自动、强制 IPv4、IPv6 优先、强制 IPv6、自定义接口/IP。
- 远程同步节点代理、重启 GOST、升级 agent、卸载 agent。
- 任务列表、失败重试、任务清理。
- 节点流量统计：今日上传/下载、累计上传/下载。
- Docker Compose 部署，GitHub tag 触发 GHCR 镜像构建。

## 架构

```mermaid
flowchart TD
  Browser["浏览器 / 管理员"]
  Client["AI 中转 / 代理客户端"]

  subgraph Management["管理端机器"]
    Panel["管理端 gost-pool-panel"]
    PoolGost["代理池入口 GOST"]
    Panel --> UI["Web UI / Admin API"]
    Panel --> Install["注册 token / install.sh / agent 下载"]
    Panel --> Store["JSON 状态存储"]
    Panel --> PoolGost
  end

  subgraph Node["Linux 节点 VPS"]
    NodeGost["节点侧 GOST HTTP/SOCKS5"]
    Agent["gost-pool-agent"] --> Tasks["心跳 / 任务轮询 / 结果回传"]
    Agent --> GostOps["安装 / 配置 / 重启 GOST"]
    Agent --> Traffic["流量统计上报"]
    GostOps --> NodeGost
  end

  Browser --> Panel
  Client --> PoolGost
  Client -.-> NodeGost
  Panel <--> Agent
  PoolGost --> NodeGost
```

管理端可运行在 Linux、Windows 或容器中；节点端必须是 Linux，当前内置 `amd64` 和 `arm64` agent。

## 快速部署

推荐生产环境使用 Docker Compose：

```bash
git clone https://github.com/faithererer/gost-pool-panel.git
cd gost-pool-panel
cp .env.example .env
```

编辑 `.env`：

```bash
PANEL_BASE_URL=http://你的管理端公网IP:3000
PANEL_ADMIN_USER=admin
PANEL_ADMIN_PASSWORD=一个强密码
PANEL_SECRET=一段随机字符串
PANEL_PROXY_USERNAME=proxy
PANEL_PROXY_PASSWORD=一个代理入口密码
```

启动：

```bash
docker compose --env-file .env up -d --build
```

Docker Compose 默认使用 `host` 网络，便于管理端同时暴露面板端口和代理池入口端口。请在安全组/防火墙中只放行需要的端口。

HTTPS 不内置。生产环境需要 HTTPS 时，请用 Nginx、Caddy、宝塔等反向代理到面板端口。

## 使用镜像

本仓库的 GitHub Actions 只在推送 `v*` tag 或手动触发 workflow 时构建镜像，普通提交不会触发镜像发布。

tag 镜像示例：

```bash
docker pull ghcr.io/faithererer/gost-pool-panel:v0.3.9
```

使用镜像部署时，把 `docker-compose.yml` 里的 `build: .` 替换成：

```yaml
image: ghcr.io/faithererer/gost-pool-panel:v0.3.9
```

然后启动：

```bash
docker compose --env-file .env up -d
```

## 环境变量

| 名称 | 默认值 | 说明 |
| --- | --- | --- |
| `PANEL_PORT` | `3000` | 管理端端口 |
| `PANEL_LISTEN` | `:3000` | 管理端监听地址 |
| `PANEL_BASE_URL` | `http://127.0.0.1:3000` | 节点访问管理端的公网 URL |
| `PANEL_ADMIN_USER` | `admin` | 管理员账号 |
| `PANEL_ADMIN_PASSWORD` | `admin123` | 管理员密码，生产环境必须修改 |
| `PANEL_SECRET` | `change-me` | 登录 Cookie 签名密钥，生产环境必须修改 |
| `PANEL_PROXY_USERNAME` | `proxy` | 首次初始化时使用的代理账号 |
| `PANEL_PROXY_PASSWORD` | 随机生成 | 首次初始化时使用的代理密码 |
| `PANEL_DATA_PATH` | `data/state.json` | 数据文件路径 |
| `PANEL_FRONTEND_DIST` | 空 | 可选，自定义前端构建产物目录 |

如果管理端使用 IPv6 公网地址，`PANEL_BASE_URL` 需要写成带中括号的 URL：

```text
http://[2600:1700:abcd::1234]:3000
```

## 节点接入

1. 登录管理端。
2. 进入“接入命令”，创建注册 token。
3. 在 Linux VPS 上执行页面生成的一键安装命令。
4. 节点自动注册上线。
5. 进入“节点”，下发“同步节点代理”。
6. 进入“分组”，把节点加入分组。
7. 进入“代理池”，选择分组并配置 HTTP/SOCKS5 入口端口。
8. 复制代理池入口地址或测试命令。

注册 token 对新节点是一次性的。已注册节点重复执行安装命令时，脚本会优先读取本机保存的节点身份，用于原地升级 agent，不会再次消耗 token。

## 出口网络模式

在“同步节点代理”任务里可以选择出口网络：

- `自动`：交给系统路由和 GOST 默认行为。
- `强制 IPv4`：agent 自动选择本机 IPv4 源地址作为 GOST 出口。
- `IPv6 优先`：同步节点代理时先探测节点 IPv6 出口；探测通过则优先使用 AAAA/IPv6，探测失败则自动下发 IPv4-only 配置。IPv4-only 目标也会走 IPv4。
- `强制 IPv6`：agent 自动选择本机 IPv6 源地址作为 GOST 出口，并让 GOST 只使用 AAAA 解析结果；目标没有 AAAA 记录时会失败。
- `自定义接口/IP`：手动填写网卡名或本机 IP，例如 `eth0`、`ens3` 或 `2600:...`。

如果某些应用使用强制 IPv6 时返回 503，通常是目标服务 IPv4-only、应用直接访问 IPv4 地址，或本机 IPv6 出口不稳定。优先改用 `IPv6 优先`。

## 代理地址

面板登录账号和代理入口账号是两套凭据。代理认证账号密码在“设置”页维护。

节点直连代理和代理池入口都会展示带认证的完整 URL：

```text
http://proxy:password@8.209.236.102:18080
socks5h://proxy:password@8.209.236.102:18081
```

测试命令示例：

```bash
curl -x "http://proxy:password@管理端IP:28080" https://api64.ipify.org
curl -x "socks5h://proxy:password@管理端IP:28081" https://api64.ipify.org
```

## 本地开发

Windows 本地运行管理端：

```powershell
cd F:\WorkSpace\opensource\gost-pool-panel
$env:PANEL_PORT="3000"
$env:PANEL_BASE_URL="http://127.0.0.1:3000"
$env:PANEL_ADMIN_USER="admin"
$env:PANEL_ADMIN_PASSWORD="admin123"
$env:PANEL_SECRET="dev-secret"
go run ./cmd/panel
```

前端开发：

```powershell
cd frontend
npm install
npm run dev
```

完整构建：

```powershell
.\scripts\build.ps1
```

Linux/macOS：

```bash
sh ./scripts/build.sh
```

构建产物：

```text
dist/gost-pool-panel.exe
dist/gost-pool-agent-linux-amd64
dist/gost-pool-agent-linux-arm64
frontend/dist/
```

## 运维说明

- 管理端状态默认保存在 `./data/state.json`，容器内路径为 `/data/state.json`。
- 节点 agent 安装在 `/opt/gost-pool-agent`。
- 节点 GOST 配置写入 `/etc/gost/gost.json`。
- 节点 GOST 二进制安装到 `/usr/local/bin/gost`。
- 远程卸载 agent 只删除本项目 agent 和 systemd 服务，不停止、不禁用、不删除 GOST。
- 流量统计依赖节点侧 `iptables` / `ip6tables` 计数规则。

## 文档

- [发布指南](docs/RELEASE.md)
- [VPS 调试指南](docs/VPS_DEBUG.md)
- [前端 API 文档](docs/FRONTEND_API.md)
- [产品规格](REQUIREMENTS.md)

## 当前限制

- 仅支持一个管理员账号。
- HTTPS 不内置。
- 代理认证为全局一组账号密码。
- 数据存储为本地 JSON 文件。
- 节点端只支持 Linux。
- 尚未实现主动健康检查和自动剔除故障节点。
- 流量统计适合面板展示，不作为账单级计费依据。
