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

默认 `docker-compose.yml` 使用 host network，这样代理池入口端口可以直接监听在管理端 VPS 上。请确认安全组/防火墙放行：

- 面板端口：默认 `3000`
- 你在代理池里配置的 HTTP/SOCKS5 入口端口

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

重复执行一键安装命令会覆盖 agent 并重启 `gost-pool-agent.service`，可以用来升级节点端 agent。升级后面板里应能看到新的 agent 版本。

查看节点 agent：

```bash
systemctl status gost-pool-agent
journalctl -u gost-pool-agent -f
```

远程卸载 agent 后，面板会先收到任务成功结果，然后节点端会在后台删除：

```text
/etc/systemd/system/gost-pool-agent.service
/opt/gost-pool-agent
```

卸载不会停止、禁用或删除 GOST 服务，也不会删除 `/etc/gost`。卸载日志在：

```bash
/tmp/gost-pool-agent-uninstall.log
```

面板的“节点”页支持删除单个节点记录，也支持清理所有 `agent uninstalled` 节点及其任务记录。这个删除只清理面板数据，不会再操作 VPS。

如果误删了仍在运行的节点记录，在面板重新生成 token，并在该 VPS 上重新执行一键安装命令。新版 agent 收到 401 后会使用当前安装命令里的 token 重新注册。

## 3. 跑通代理池

### 3.1 同步节点代理

节点上线后，进入“设置”确认全局出口账号密码，然后进入“节点”，对节点下发“同步节点代理”任务。

agent 会在节点 VPS 上执行：

- 下载并安装 GOST v3 到 `/usr/local/bin/gost`
- 写入 `/etc/gost/gost.json`
- 创建并重启 `gost.service`
- 暴露节点侧 HTTP/SOCKS5 代理端口

节点 VPS 的安全组/防火墙需要允许管理端 VPS 访问节点侧 HTTP 代理端口，默认是 `18080`。建议只放行管理端 VPS 的 IP。

任务成功后，节点下一次心跳应显示：

```text
gost 3.x active
```

### 3.2 创建代理池

1. 进入“分组”，创建分组。
2. 进入“节点”，把已同步 GOST 的节点加入分组。
3. 进入“代理池”，选择分组并配置 HTTP/SOCKS5 入口端口。
4. 如果节点还没 ready，稍后点击“重启入口”。

测试 HTTP 入口：

```bash
curl -x http://用户名:密码@管理端IP:HTTP入口端口 https://api.ipify.org
```

测试 SOCKS5 入口：

```bash
curl --socks5 用户名:密码@管理端IP:SOCKS5入口端口 https://api.ipify.org
```

### GOST 显示 not installed

如果节点上还没有执行“同步节点代理”任务，面板会显示：

```text
gost not installed
```

这表示 agent 正常，但节点侧 GOST 尚未安装。

## 4. 常见问题

### 节点下载 agent 404

确认管理端镜像内存在 agent：

```bash
docker exec -it gost-pool-panel ls -lah /app/dist
```

如果没有，重新构建镜像：

```bash
docker compose --env-file .env up -d --build
```

### curl: (23) Failure writing output to destination

这通常是重复安装/升级时，旧的 `gost-pool-agent` 进程还在运行，脚本直接覆盖正在执行的二进制导致写入失败。

新版安装脚本会先下载到临时文件，再原子替换 agent 二进制。更新管理端镜像后重新执行一键安装命令即可。

如果你仍在旧镜像上，可以先手动停掉旧 agent，再重新执行安装命令：

```bash
systemctl stop gost-pool-agent
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
