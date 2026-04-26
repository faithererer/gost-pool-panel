# VPS 调试指南

这份文档用于把管理端部署到 VPS，然后用另一台 Linux VPS 接入为节点。

## 1. 准备管理端 VPS

安装 Docker 和 Docker Compose 插件后，拉取代码：

```bash
git clone https://github.com/YOUR_NAME/gost-pool-panel.git
cd gost-pool-panel
cp .env.example .env
```

编辑 `.env`：

```bash
PANEL_BASE_URL=http://你的管理端公网IP:3000
PANEL_ADMIN_PASSWORD=一个强密码
PANEL_SECRET=一段随机字符串
```

启动：

```bash
docker compose --env-file .env up -d --build
```

打开：

```text
http://你的管理端公网IP:3000
```

HTTPS 第一版不内置。需要 HTTPS 时，用 Nginx、Caddy 或宝塔反代到 `127.0.0.1:3000`。

## 2. 接入节点 VPS

登录管理端：

```text
admin / 你设置的 PANEL_ADMIN_PASSWORD
```

进入“接入命令”，生成 token，然后在 Linux 节点 VPS 上执行生成的命令。

节点脚本会：

- 下载 `gost-pool-agent-linux-amd64` 或 `gost-pool-agent-linux-arm64`。
- 安装到 `/opt/gost-pool-agent`。
- 创建 `gost-pool-agent.service`。
- 启动 agent。

查看节点 agent：

```bash
systemctl status gost-pool-agent
journalctl -u gost-pool-agent -f
```

## 3. 常见问题

### 节点下载 agent 404

确认管理端镜像内存在 agent：

```bash
docker exec -it gost-pool-panel ls -lah /app/dist
```

如果没有，重新构建镜像：

```bash
docker compose --env-file .env up -d --build
```

### 节点注册失败

确认：

- `PANEL_BASE_URL` 是节点 VPS 能访问的公网地址。
- 管理端端口已放行。
- token 没过期且没有被使用过。

### Web 面板能打开，但 agent 连不上

在节点 VPS 测试：

```bash
curl -I http://你的管理端公网IP:3000/install.sh
```

如果访问失败，优先检查防火墙、安全组、反代配置。
