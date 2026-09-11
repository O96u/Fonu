# Fonu 产品需求文档

## 1. 产品定义

**产品名称：** Fonu

**产品定位：**

Fonu 是一个面向 NAS 用户的轻量公网访问管理工具，优先适配飞牛 fnOS。

Fonu 专注解决四个核心问题：

1. 公网 IP 变化后自动更新 DNS
2. 自动申请和续签 HTTPS 证书
3. 使用二级域名访问 NAS 内部不同 Web 服务
4. 查看访问日志、错误日志和系统运行状态

Fonu 不追求成为大型网络工具箱。

核心原则：

> 简单、轻量、稳定、自动化、易维护。

---

# 2. 产品边界

Fonu V1 只围绕：

```text
DDNS
HTTPS / ACME
反向代理
日志
```

进行开发。

V1 不包含：

```text
FRP
内网穿透
STUN
P2P
VPN
WAF
WebDAV
文件管理
SSH 管理
端口转发
TCP / UDP 四层代理
负载均衡
多用户权限体系
集群
Kubernetes
Redis
MySQL
PostgreSQL
消息队列
```

如果某项功能不能明显改善：

> NAS 公网访问体验

则默认不加入 Fonu。

---

# 3. 目标环境

主要目标：

```text
飞牛 fnOS
Docker / Docker Compose
家庭公网 IPv4 / IPv6
自有域名
```

典型环境：

```text
Internet
   ↓
家庭公网 IP
   ↓
路由器
   ↓
飞牛 NAS
   ↓
Docker
   ↓
Fonu
```

优先支持：

```text
linux/amd64
linux/arm64
```

---

# 4. 核心技术栈

后端：

```text
Go
```

前端：

```text
Vue 3
Vite
TypeScript
Naive UI
```

数据库：

```text
SQLite
```

反向代理：

```text
独立 Nginx
```

证书：

```text
ACME
lego
Let's Encrypt
DNS-01
```

部署：

```text
Docker
Docker Compose
```

日志：

```text
Go slog
Nginx access log
Nginx error log
```

---

# 5. 开发规范原则

本产品文档负责定义：

> Fonu 要做什么。

项目开发规范负责定义：

> Fonu 应该怎么实现。

开发过程中必须严格遵守项目指定的开发规范。

如果产品文档与开发规范涉及不同层面的要求：

```text
产品功能 / 产品边界
→ PRODUCT.md

编码规范 / 架构规范 / 命名 / 测试 / Git / 安全实现
→ 开发规范
```

不得为了方便实现而绕过开发规范。

不得擅自修改产品边界。

---

# 6. 项目结构原则

PRODUCT.md **不固定仓库目录结构**。

AI / 开发者应根据：

- Go 工程最佳实践
- Vue 工程最佳实践
- 模块职责
- 可测试性
- 可维护性
- 项目开发规范

自行设计合理目录。

要求：

```text
模块职责清晰
避免循环依赖
避免无意义分层
避免过度抽象
保持项目轻量
```

不允许为了“架构完整”引入实际不需要的组件。

---

# 7. 端口设计

Fonu 默认使用：

```text
6893
管理后台 / Go API
```

因此公网用户仍然正常访问：

```text
http://nas.example.com
https://nas.example.com
```

无需携带端口。

---

# 8. fnOS 隔离原则

Fonu 必须使用独立 Nginx。

Fonu：

```text
不读取 fnOS 原生 Nginx 配置
不修改 fnOS 原生 Nginx 配置
不覆盖 fnOS 原生 Nginx 文件
不 reload fnOS 原生 Nginx
不依赖 fnOS Nginx 内部实现
```

Fonu 与 fnOS 原生 Web 服务之间只通过标准 HTTP / HTTPS 通信。

例如：

```text
nas.example.com
        ↓
Fonu Nginx
        ↓
http://192.168.1.10:5666
        ↓
fnOS
```

---

# 9. 数据持久化

所有持久化数据统一放入：

```text
/data
```

具体子目录结构不在产品文档中固定，由实现根据开发规范设计。

必须持久化：

```text
SQLite 数据库
证书和私钥
Nginx 生成配置
应用日志
Nginx access/error log
备份数据
```

删除或升级容器不得导致配置丢失。

---

# 10. Dashboard

首页用于快速判断系统是否正常。

至少显示：

```text
公网 IPv4
公网 IPv6

DDNS 状态

证书状态
证书剩余有效期

反向代理规则数量

今日请求量
错误请求量

Fonu 运行状态
Nginx 运行状态
```

首页保持简洁。

V1 不做复杂监控平台。

---

# 11. DDNS

## 11.1 目标

当家庭公网 IP 变化后，自动修改 DNS A / AAAA 记录。

典型：

```text
*.example.com
      ↓
当前家庭公网 IP
```

公网 IP：

```text
123.45.67.89
```

变化为：

```text
111.22.33.44
```

Fonu 自动更新 DNS。

---

## 11.2 支持

V1：

```text
A
AAAA
```

支持：

```text
example.com
*.example.com
home.example.com
```

推荐默认：

```text
*.example.com
```

---

## 11.3 Provider

首个完整实现：

```text
Cloudflare
```

架构必须允许后续增加：

```text
DNSPod
AliDNS
其他 Provider
```

但 V1 不要求一次实现大量 DNS Provider。

---

## 11.4 更新策略

默认：

```text
5 分钟检测一次
```

流程：

```text
获取公网 IP
↓
读取当前 DNS 记录
↓
比较 IP
↓
相同
→ 不做操作

不同
→ 更新 DNS
↓
记录日志
```

必须避免没有变化时重复调用 DNS API。

---

# 12. HTTPS / ACME

默认 CA：

```text
Let's Encrypt
```

ACME 实现：

```text
lego
```

验证方式：

```text
DNS-01
```

推荐默认申请：

```text
example.com
*.example.com
```

原因：

```text
支持 wildcard
不依赖公网 80 完成验证
适合 NAS 场景
```

---

# 13. 证书续签

Fonu 自动维护证书。

每天检查证书状态。

默认：

```text
剩余 ≤ 30 天
```

触发续签。

续签成功：

```text
写入新证书
↓
验证 Nginx 配置
↓
平滑 reload Nginx
```

续签失败：

```text
保留当前有效证书
记录 ERROR
Dashboard 显示异常状态
```

用户可以手动执行：

```text
立即申请
立即续签
```

---

# 14. 反向代理

这是 Fonu 的核心功能。

用户新增规则时只需要理解：

```text
访问域名
目标地址
```

例如：

```text
nas.example.com
→
http://192.168.1.10:5666
```

```text
alist.example.com
→
http://192.168.1.10:5244
```

```text
jellyfin.example.com
→
http://192.168.1.10:8096
```

---

# 15. 反向代理规则

基础字段：

```text
域名
目标地址
是否启用 HTTPS
是否启用规则
```

目标地址 V1 支持：

```text
http://
https://
```

高级 Nginx 参数不向普通用户暴露。

---

# 16. HTTPS 自动匹配

如果 Fonu 已拥有：

```text
*.example.com
```

证书，那么：

```text
nas.example.com
alist.example.com
jellyfin.example.com
```

应自动使用该证书。

用户不需要手动为每个反向代理规则选择证书。

---

# 17. HTTP → HTTPS

默认开启。

```text
http://nas.example.com
        ↓
https://nas.example.com
```

允许用户关闭。

---

# 18. WebSocket / SSE

反向代理必须默认兼容：

```text
WebSocket
SSE
```

普通用户无需看到相关技术配置。

---

# 19. Nginx 配置安全

任何可能影响 Nginx 的配置变更都必须遵循：

```text
生成候选配置
↓
执行 nginx 配置测试
↓
测试成功
→ 应用配置
→ graceful reload

测试失败
→ 保留当前工作配置
→ 返回可读错误
```

必须满足：

> 一条错误规则不能导致现有所有代理失效。

禁止保存错误配置后直接 reload。

---

# 20. 日志

Fonu 提供三种日志。

## Access Log

来源：

```text
Nginx access log
```

展示至少包括：

```text
时间
域名
方法
路径
HTTP 状态码
响应耗时
来源 IP
上游地址
```

## Error Log

来源：

```text
Nginx error log
```

## Application Log

来源：

```text
Go
```

至少包含模块：

```text
SYSTEM
DDNS
ACME
NGINX
CONFIG
AUTH
```

---

# 21. 实时日志

Web UI 支持实时查看日志。

推荐使用：

```text
SSE
```

而不是为了日志功能额外引入 WebSocket 系统。

---

# 22. 日志存储

访问日志不写入 SQLite。

日志使用文件存储并轮转。

默认策略建议：

```text
单文件最大 20 MB
最多 10 个备份
最长保留 30 天
```

具体实现应遵循开发规范。

---

# 23. SQLite

SQLite 用于保存：

```text
反向代理规则
DDNS 配置
证书元数据
系统设置
管理员账户
必要运行状态
```

不用于保存完整访问日志。

不要求使用 ORM。

具体 Schema 和 migration 设计由实现根据开发规范确定。

---

# 24. 管理后台

默认：

```text
http://NAS-IP:6893
```

管理后台优先面向 LAN 使用。

不主动建议用户将：

```text
6893
```

暴露至公网。

---

# 25. 前端

技术栈：

```text
Vue 3
Vite
TypeScript
Naive UI
```

是否使用：

```text
Pinia
Axios
```

由实际复杂度和开发规范决定。

如果原生 Vue 状态与 fetch 已经足够，则不要为了技术栈完整度强行加入依赖。

---

# 26. UI

一级导航：

```text
仪表盘
反向代理
DDNS
证书
日志
设置
```

不设计复杂多层导航。

设计原则：

```text
简单
现代
低信息噪音
NAS 管理工具风格
```

支持：

```text
浅色
深色
跟随系统
```

桌面端优先，同时保证移动设备基本可用。

---

# 27. 首次使用向导

第一次打开 Fonu：

```text
http://NAS-IP:6893
```

流程：

### 1. 创建管理员

设置管理员密码。

### 2. 配置主域名

例如：

```text
example.com
```

### 3. 配置 DNS Provider

例如：

```text
Cloudflare
API Token
```

### 4. 测试 DNS API

验证：

```text
Token 是否有效
Zone 是否可访问
DNS 是否可修改
```

### 5. 配置 DDNS

推荐：

```text
*.example.com
```

### 6. 申请证书

自动申请：

```text
example.com
*.example.com
```

### 7. 创建第一条代理

例如：

```text
nas.example.com
→
http://192.168.1.10:5666
```

完成。

---

# 28. 安全要求

Fonu 管理后台属于高权限服务。

至少要求：

```text
管理员认证
安全密码哈希
HttpOnly Cookie
输入校验
敏感字段保护
```

DNS API Token：

```text
UI 不返回明文
UI 默认掩码显示
只允许重新填写
```

禁止：

```text
将 Token 返回给前端用于展示
将密码明文保存
将敏感认证信息写入普通日志
```

---

# 29. 输入安全

反向代理输入必须严格校验。

域名：

```text
必须是合法 hostname
```

upstream：

```text
仅允许明确支持的 http:// 或 https:// URL
```

用户输入不得作为：

```text
任意 Nginx 指令
shell command
文件路径
```

直接执行。

---

# 30. Docker

V1 使用：

```text
Docker Compose
```

推荐默认映射：

```text
6893:6893

18080:80

9443:443
```

即：

```text
6893
管理后台

18080
HTTP 反代入口

9443
HTTPS 反代入口
```

---

# 31. 容器原则

V1 优先提供简单部署体验。

允许采用单容器方案：

```text
Go
Vue build
Nginx
```

但具体进程管理方式由开发规范和工程设计决定。

不要因为追求“一个容器”而采用不可靠的进程管理方案。

---

# 32. V1 优先级

## P0

必须完成：

```text
管理后台
管理员认证

SQLite

反向代理 CRUD
Nginx 配置生成
Nginx 配置验证
Nginx 安全 reload

Cloudflare DDNS IPv4

ACME DNS-01
Wildcard Certificate
证书自动续签

访问日志
错误日志
系统日志

Docker Compose
amd64 / arm64
```

## P1

后续：

```text
IPv6 DDNS
DNSPod
AliDNS

实时日志
Dashboard 请求统计

备份恢复
深色模式完善
```

## P2

再以后：

```text
服务自动发现
fnOS 服务识别
通知
更多 DNS Provider
fnOS FPK
```

---

# 33. 产品体验原则

用户应该只需要理解：

```text
我的域名
我的 DNS API Token
我的 NAS 服务地址
```

用户不应该被要求理解：

```text
nginx.conf
server block
SNI
ACME Challenge
Certificate Chain
WebSocket Header
DNS TTL
```

这些应该由 Fonu 自动处理。

---

# 34. 产品最终原则

开发 Fonu 时始终遵循：

```text
少即是多

默认配置优先

自动化优先

高级参数隐藏

安全失败优先于危险成功

错误可恢复

不修改 fnOS 系统配置

不重复造不必要的基础设施

不为了“以后可能需要”提前增加功能

不成为第二个 Lucky
```

---

# 35. 一句话介绍

> Fonu 是一个为 NAS 用户设计的轻量公网访问管理工具，让域名、HTTPS 和服务访问变得简单。

英文：

> Fonu makes secure NAS access simple.
