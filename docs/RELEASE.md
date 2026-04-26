# 发布指南

## GitHub 仓库

首次发布：

```bash
git remote add origin https://github.com/YOUR_NAME/gost-pool-panel.git
git push -u origin main
```

推送到 `main` 或 `master` 后，GitHub Actions 会构建并发布镜像到 GHCR：

```text
ghcr.io/YOUR_NAME/gost-pool-panel:main
```

打 tag：

```bash
git tag v0.1.0
git push origin v0.1.0
```

会生成：

```text
ghcr.io/YOUR_NAME/gost-pool-panel:v0.1.0
```

## 使用 GHCR 镜像

把 `docker-compose.yml` 中的 `build: .` 替换为：

```yaml
image: ghcr.io/YOUR_NAME/gost-pool-panel:main
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
