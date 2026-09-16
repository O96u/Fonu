<p align="center">
  <img src="web/src/assets/brand/logo.png" alt="Fonu Logo" width="96" />
</p>

<h1 align="center">Fonu</h1>

<p align="center"><strong>让 NAS 访问更简单</strong></p>

<p align="center">
  面向 <strong>飞牛 fnOS</strong> 的轻量公网访问管理工具<br />
  反向代理 · DDNS · HTTPS 证书 · 一站式 Web 管理
</p>

<p align="center">
  <a href="https://hub.docker.com/r/muxui/fonu"><img src="https://img.shields.io/docker/v/muxui/fonu?sort=semver&style=flat-square&logo=docker&logoColor=white" alt="Docker Hub" /></a>
  <img src="https://img.shields.io/badge/目标平台-飞牛_fnOS-22c55e?style=flat-square" alt="飞牛 fnOS" />
  <img src="https://img.shields.io/badge/其他环境-未测试-94a3b8?style=flat-square" alt="其他环境未测试" />
  <img src="https://img.shields.io/badge/Go-1.23-00ADD8?style=flat-square&logo=go&logoColor=white" alt="Go" />
  <img src="https://img.shields.io/badge/Vue-3-4FC08D?style=flat-square&logo=vuedotjs&logoColor=white" alt="Vue 3" />
  <img src="https://img.shields.io/badge/Nginx-内置-009639?style=flat-square&logo=nginx&logoColor=white" alt="Nginx" />
  <a href="https://www.gnu.org/licenses/gpl-3.0.html"><img src="https://img.shields.io/badge/license-GPL--3.0-blue?style=flat-square" alt="GPL-3.0" /></a>
</p>

---

## 简介

**Fonu** 是一款主要为 **飞牛 fnOS** 编写的反代与管理工具，帮助你在 NAS 上快速搭建稳定的公网访问能力：通过 Web 界面管理 Nginx 反向代理、自动同步 DDNS、申请与续期 Let's Encrypt 证书，并在一个仪表盘里查看服务状态、访问日志与流量统计。

无需手写 Nginx 配置，也无需在多个工具之间切换。当前开发与测试环境以飞牛 fnOS 为主，**群晖、威联通、自建 Linux 等其他环境尚未充分测试**，部署前请自行验证。

## 功能特性

| 模块           | 说明                                                                        |
| -------------- | --------------------------------------------------------------------------- |
| **反向代理**   | 多域名、多端口、HTTP/HTTPS、自动重定向；实时访问日志、流量统计、当前连接 IP |
| **DDNS**       | 支持 Cloudflare、DNSPod、阿里云、腾讯云 DNS；单任务多根域名，自动同步公网 IP |
| **HTTPS 证书** | ACME 自动申请与续期；证书/私钥/ZIP 下载；申请进度实时日志                   |
| **仪表盘**     | 公网 IP、域名、证书、服务状态一览；请求趋势与运行健康度                     |
| **日志**       | Nginx 访问/错误日志、系统运行日志；分页与实时 SSE 推送                      |
| **内网穿透**   | FRP 客户端（Nginx 网关穿透）：无公网 IP 时经 VPS 中转，反代与安全策略不变   |
| **设置**       | 管理员密码、配置备份与恢复、服务发现（如飞牛 fnOS :5666）                   |
| **其他**       | 深色模式、配置导出/导入、GitHub Release 更新提示                            |

## 界面展示

<p align="center"><strong>仪表盘</strong> — 服务状态、请求趋势、实时流量与运行健康度</p>
<p align="center">
  <img src="docs/screenshots/dashboard.png" alt="仪表盘" width="92%" />
</p>

<p align="center"><strong>DDNS</strong> — 多服务商、多根域名、解析记录与同步状态</p>
<p align="center">
  <img src="docs/screenshots/ddns.png" alt="DDNS" width="92%" />
</p>

<p align="center"><strong>HTTPS 证书</strong> — ACME 申请、自动续期、导入与下载</p>
<p align="center">
  <img src="docs/screenshots/certificates.png" alt="HTTPS 证书" width="92%" />
</p>

<p align="center"><strong>反向代理</strong> — 多域名多端口、流量统计与连接详情</p>
<p align="center">
  <img src="docs/screenshots/proxy.png" alt="反向代理" width="92%" />
</p>

<p align="center"><strong>设置</strong> — 外观、自动任务、安全、通知与配置备份</p>
<p align="center">
  <img src="docs/screenshots/settings.png" alt="设置" width="92%" />
</p>

## 技术栈

| 层级         | 技术                                                  |
| ------------ | ----------------------------------------------------- |
| **后端**     | Go 1.23、标准库 HTTP、SQLite（modernc.org/sqlite）    |
| **反向代理** | 内置 Nginx（动态生成配置、热重载）                    |
| **证书**     | go-acme/lego（Let's Encrypt）                         |
| **DDNS**     | Cloudflare / DNSPod / 阿里云 / 腾讯云 DNS API         |
| **内网穿透** | 内置 frpc（v0.71.x），固定穿透 Fonu Nginx 端口        |
| **前端**     | Vue 3、TypeScript、Vite、Naive UI、ECharts            |
| **部署**     | Docker 多架构镜像（amd64 / arm64）、GitHub Actions CI |

## 部署

Docker 镜像：[hub.docker.com/r/muxui/fonu](https://hub.docker.com/r/muxui/fonu)

### 快速部署（推荐 · 飞牛 fnOS · Host 模式）

飞牛系统已占用 **80/443**，使用 **host 网络** + **高位反代端口**，避免与系统 Nginx 冲突。

```bash
docker run -d \
  --name fonu \
  --net=host \
  -v ./data:/data \
  --restart unless-stopped \
  muxui/fonu:latest
```

| 入口               | 地址                    |
| ------------------ | ----------------------- |
| 管理后台           | `http://<NAS-IP>:6893`  |
| HTTP 反代（默认）  | `http://<NAS-IP>:18080` |
| HTTPS 反代（默认） | `https://<NAS-IP>:9443` |

- `./data` 挂载后自动作为数据目录，存放数据库、Nginx 配置、证书与日志
- 飞牛系统已占用 80/443，容器默认反代端口为 `18080`、`9443`，可通过环境变量修改

### 其他部署方式

<details>
<summary>Docker Compose</summary>

```bash
FONU_SESSION_SECRET=your-secret docker compose -f docker-compose.host.yml up -d
```

</details>

<details>
<summary>Bridge 网络（端口映射，非飞牛环境可自行尝试）</summary>

```bash
docker pull muxui/fonu
FONU_SESSION_SECRET=your-secret docker compose -f docker-compose.hub.yml up -d
```

</details>

<details>
<summary>本地构建</summary>

```bash
docker compose up -d --build
```

</details>

### 首次登录

1. 启动后查看容器/运行日志，获取自动生成的 **admin 初始密码**
2. 访问 `http://<NAS-IP>:6893/login` 登录
3. 在 **设置** 中立即修改管理员密码

### FRP 内网穿透（无公网 IP）

适用于无法端口映射、无公网 IPv4 的场景。Fonu 内置 **frpc**，仅做 **Nginx 网关穿透**：外网流量经 VPS 上的 frps 中转，再转发到本地 Nginx（默认 `18080` / `9443`）。域名分流、HTTPS 证书、访问控制、日志仍全部由 Fonu Nginx 处理。

**流量路径**：用户 → DNS（解析到 VPS）→ frps → frpc → Fonu Nginx → 内网服务

#### 1. VPS 部署 frps

在具有公网 IP 的 VPS 上安装 [frp](https://github.com/fatedier/frp)（建议 v0.71.x，与 Fonu 内置 frpc 同版本），创建 `frps.toml`：

```toml
bindAddr = "0.0.0.0"
bindPort = 7000

auth.method = "token"
auth.token = "your-secret-token"

vhostHTTPPort = 80
vhostHTTPSPort = 443
```

启动：`frps -c frps.toml`。安全组/防火墙需放行 **7000**（控制连接）及 **80/443**（HTTP/HTTPS 虚拟主机）。

#### 2. Fonu 配置 frpc

在侧栏 **内网穿透** 页面填写：

- FRP 服务器地址 / 端口（默认 7000）
- 与 frps 一致的认证 Token
- 穿透域名列表（与反代规则中的域名一致，支持 `*.example.com`）

保存后 Fonu 自动生成 `frpc.toml` 并启动 frpc，固定创建 HTTP → Nginx HTTP 端口、HTTPS → Nginx HTTPS 端口两条隧道。

#### 3. DNS 与证书

- **DDNS**：启用 FRP 后，域名应解析到 **VPS 公网 IP**，而非 NAS IP（DDNS 页会有提示）
- **HTTPS 证书**：推荐继续使用 **DNS-01** 验证（Cloudflare 等已支持）；证书仍在 NAS 侧 Nginx 终结

## 环境变量

| 变量                    | 默认值                    | 说明                                       |
| ----------------------- | ------------------------- | ------------------------------------------ |
| `FONU_LISTEN`           | `:6893`                   | 管理后台监听地址                           |
| `FONU_DATA_DIR`         | `/data`                   | 数据目录（数据库、Nginx 配置、证书、日志） |
| `FONU_SESSION_SECRET`   | `change-me-in-production` | 会话加密密钥，生产环境请自行修改           |
| `FONU_NGINX_HTTP_PORT`  | `80`                      | 新建 HTTP 反代规则的默认监听端口           |
| `FONU_NGINX_HTTPS_PORT` | `443`                     | 新建 HTTPS 反代规则的默认监听端口          |
| `FONU_NGINX_BIN`        | `nginx`                   | Nginx 可执行文件路径（本地开发可指定）     |
| `FONU_FRPC_BIN`         | `frpc`                    | frpc 可执行文件路径（Docker 镜像内置）     |
| `FONU_FRP_PID`          | `{DATA_DIR}/frp/frpc.pid` | frpc 进程 PID 文件路径                     |

## 注意事项

1. **生产环境务必设置** `FONU_SESSION_SECRET`，不要使用默认值。
2. **端口冲突**：宿主机已有 Nginx（如飞牛系统）占用 80/443 时，不要让 Fonu 反代规则监听 80/443；使用 `18080/9443` 或 host 模式 + 环境变量。
3. **已有反代规则**：修改 `FONU_NGINX_HTTP_PORT` / `FONU_NGINX_HTTPS_PORT` 后，**不会自动更新**数据库中已有规则的端口，需在反代页手动修改并保存。
4. **访问日志**：仅统计经 Fonu Nginx 反代的域名流量，不包含管理后台 `:6893` 的请求。
5. **证书与 DNS**：HTTPS 申请前请确保域名 DNS 已解析到本机，且 DDNS 凭证能管理对应根域。
6. **数据持久化**：Docker 部署请挂载 volume 到 `FONU_DATA_DIR`，避免容器重建后配置丢失。
7. **备份**：可通过设置页导出配置；重要证书与数据库建议定期备份 `FONU_DATA_DIR` 目录。

## 本地开发

```bash
# 前端
cd web && npm ci && npm run dev

# 后端（需先构建前端到 cmd/fonu/web/dist）
make build
go run ./cmd/fonu

# 测试
make test
```

## 开源协议

[GPL-3.0](https://www.gnu.org/licenses/gpl-3.0.html)
