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

# 本地开发（Windows PowerShell，使用本机 Nginx）
. .\scripts\dev.ps1
go run ./cmd/fonu
# 首次启动会在系统日志中输出 admin 初始密码，访问 http://localhost:6893/login 登录后在设置中修改

# 清空本地数据（勿用 seed，直接录入真实配置）
. .\scripts\clean-data.ps1

# 可选：写入模拟数据（仅调试 UI）
# go run ./cmd/seed

# 测试
make test
```

## Docker

### 本地构建

```bash
docker compose up -d --build
```

### 从 Docker Hub 拉取（发版后）

GitHub 推送 `v*` 标签（如 `v0.1.0`）后，Actions 会分架构推送到 Docker Hub（镜像名：`{DOCKERHUB_USERNAME}/fonu`）。

| 标签 | 架构 | 说明 |
|------|------|------|
| `latest` | amd64 | 最新版（x86） |
| `0.1.0` | amd64 | 指定版本（x86） |
| `0.1.0-amd64` | amd64 | 显式架构标签 |
| `0.1.0-arm64` | arm64 | ARM 设备专用 |

```bash
# x86 / amd64（默认 latest）
FONU_SESSION_SECRET=your-secret docker compose -f docker-compose.hub.yml up -d

# 指定版本（amd64）
FONU_VERSION=0.1.0 FONU_SESSION_SECRET=your-secret docker compose -f docker-compose.hub.yml up -d

# ARM64 设备
FONU_VERSION=0.1.0-arm64 FONU_SESSION_SECRET=your-secret docker compose -f docker-compose.hub.yml up -d

# 若 Docker Hub 用户名不是默认值
DOCKERHUB_USERNAME=your-dockerhub-username FONU_VERSION=0.1.0-arm64 FONU_SESSION_SECRET=your-secret docker compose -f docker-compose.hub.yml up -d
```

### 发版

在 **GitHub 仓库 Secrets** 配置（仅 CI 构建推送时使用，不写死在代码里）：

- `DOCKERHUB_USERNAME`：Docker Hub 用户名
- `DOCKERHUB_TOKEN`：Docker Hub Access Token

本地 `docker compose` 拉镜像时，通过环境变量 `DOCKERHUB_USERNAME` 指定用户名（必填）。

本地打标签并推送（由你本人提交，不要用 Cursor 自动提交）：

```bash
git tag v0.1.0
git push origin v0.1.0
```

推送标签后 GitHub Actions 会自动构建镜像并创建 [GitHub Release](https://github.com/O96u/Fonu/releases)。Release 说明从 [CHANGELOG.md](./CHANGELOG.md) 对应版本章节自动提取。

管理后台：`http://NAS-IP:6893`

反向代理入口：`http://NAS-IP:18080` / `https://NAS-IP:9443`
