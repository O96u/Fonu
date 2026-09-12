# Fonu

面向 NAS 的轻量公网访问管理工具。

Phase 1：反向代理基础  
Phase 2：DDNS、ACME 证书、日志、Dashboard、设置、深色模式

## 开发

```bash
# 前端
cd web && npm ci && npm run dev

# 后端（需要先构建前端到 cmd/fonu/web/dist）
make build

# 本地模拟数据（反向代理 / DDNS / 证书 / 日志样本）
# Windows PowerShell:
$env:FONU_DATA_DIR = ".\.data"
$env:FONU_SESSION_SECRET = "dev-secret-change-me"
go run ./cmd/seed

# 测试
make test
```

## Docker

### 本地构建

```bash
docker compose up -d --build
```

### 从 Docker Hub 拉取（发版后）

GitHub 推送 `v*` 标签（如 `v0.1.0`）后，Actions 会用 Secrets 里的账号推送到 Docker Hub（镜像名：`{DOCKERHUB_USERNAME}/fonu`）。

```bash
# 使用最新版（默认拉取 muxui/fonu，可用环境变量覆盖）
FONU_SESSION_SECRET=your-secret docker compose -f docker-compose.hub.yml up -d

# 指定版本
FONU_VERSION=v0.1.0 FONU_SESSION_SECRET=your-secret docker compose -f docker-compose.hub.yml up -d

# 若 Docker Hub 用户名不是默认值
DOCKERHUB_USERNAME=muxui FONU_SESSION_SECRET=your-secret docker compose -f docker-compose.hub.yml up -d
```

### 发版

在 **GitHub 仓库 Secrets** 配置（仅 CI 构建推送时使用，不写死在代码里）：

- `DOCKERHUB_USERNAME`：Docker Hub 用户名（如 `muxui`）
- `DOCKERHUB_TOKEN`：Docker Hub Access Token

本地 `docker compose` 拉镜像时，通过环境变量 `DOCKERHUB_USERNAME` 指定用户名（默认 `muxui`）。

本地打标签并推送（由你本人提交，不要用 Cursor 自动提交）：

```bash
git tag v0.1.0
git push origin v0.1.0
```

推送标签后 GitHub Actions 会自动构建镜像并创建 [GitHub Release](https://github.com/O96u/Fonu/releases)。Release 说明从 [CHANGELOG.md](./CHANGELOG.md) 对应版本章节自动提取。

管理后台：`http://NAS-IP:6893`

反向代理入口：`http://NAS-IP:18080` / `https://NAS-IP:9443`
