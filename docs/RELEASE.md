# 发布指南

本项目的镜像发布策略是：普通分支推送不构建镜像，只有推送 `v*` tag 或手动触发 GitHub Actions workflow 才构建并推送 GHCR 镜像。

## 1. 发布前检查

在本地先跑完整构建：

```powershell
.\scripts\build.ps1
```

至少确认：

- 前端 `npm run build` 通过。
- Go 编译通过。
- `dist/` 中包含 Linux 节点 agent：
  - `gost-pool-agent-linux-amd64`
  - `gost-pool-agent-linux-arm64`

建议同时跑测试：

```powershell
$env:GOTMPDIR=(Join-Path (Get-Location) '.gotmp')
$env:GOCACHE=(Join-Path (Get-Location) '.gocache')
go test ./...
```

## 2. 推送代码

```bash
git push origin main
```

普通 `main` 推送不会触发镜像构建。

## 3. 创建 tag

示例：

```bash
git tag v0.3.9
git push origin v0.3.9
```

推送 tag 后，GitHub Actions 会构建并发布：

```text
ghcr.io/faithererer/gost-pool-panel:v0.3.9
```

如果 fork 到其他账号，镜像名为：

```text
ghcr.io/<owner>/<repo>:<tag>
```

## 4. 手动触发发布

GitHub 仓库页面进入：

```text
Actions -> Docker Image -> Run workflow
```

手动触发也会构建镜像，但推荐正式发布仍使用 tag，方便 VPS 侧固定版本。

## 5. 使用发布镜像

把 `docker-compose.yml` 中的：

```yaml
build: .
```

替换为：

```yaml
image: ghcr.io/faithererer/gost-pool-panel:v0.3.9
```

然后启动：

```bash
docker compose --env-file .env up -d
```

如果 GHCR 包是私有的，需要先登录：

```bash
docker login ghcr.io
```

开源发布时，建议在 GitHub 仓库的 Packages 页面把镜像设为公开。

## 6. 升级管理端

更新 `docker-compose.yml` 的镜像 tag 后：

```bash
docker compose --env-file .env pull
docker compose --env-file .env up -d
```

确认版本：

```bash
docker exec gost-pool-panel /app/gost-pool-panel --version
docker exec gost-pool-panel /app/dist/gost-pool-agent-linux-amd64 --version
```

## 7. 升级节点 Agent

管理端镜像更新后，进入面板“节点”页：

- 单节点：点击“升级 Agent”。
- 多节点：点击“批量升级 Agent”。

节点会从管理端 `/downloads/` 下载当前架构的新 agent，任务成功后自动重启 `gost-pool-agent.service`。

如果节点 agent 太旧，不支持远程升级，可以在 VPS 上重新执行该节点的一键安装命令。已注册节点会读取本机保存的身份信息，按原地升级处理。

## 8. 回滚

回滚管理端只需要把 compose 里的镜像 tag 改回旧版本：

```yaml
image: ghcr.io/faithererer/gost-pool-panel:v0.3.6
```

然后：

```bash
docker compose --env-file .env up -d
```

注意：回滚二进制不会自动回滚 `data/state.json`。当前状态文件是 JSON，重大版本升级前建议先备份：

```bash
cp -a data data.bak.$(date +%Y%m%d%H%M%S)
```
