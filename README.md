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

```bash
docker compose up -d --build
```

管理后台：`http://NAS-IP:6893`

反向代理入口：`http://NAS-IP:18080` / `https://NAS-IP:9443`
