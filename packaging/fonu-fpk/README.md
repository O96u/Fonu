# Fonu 飞牛 fnOS 原生 FPK

真·原生安装包：内置 `fonu`（Go + 嵌入 Vue 管理端）、源码编译的 `Nginx`、`frpc`。安装后目录遵循飞牛约定：

```
/var/apps/fonu/
  target/Nginx     编译进去的 nginx（TRIM_APPDEST）
  target/fonu      管理端
  target/frpc
  var/             运行数据、PID、cache（TRIM_PKGVAR）
  shares/etc/      nginx 默认 conf / modules
```

nginx 使用当前 **stable 1.30.x**（1.28/1.29 已停止维护），configure 路径与官方 Nginx FPK 相同，仅把应用名换成 `fonu`。

模板与脚本在 Git 仓库中；**`.cache/`、`packaging/dist/`** 不提交。

## 构建

[`scripts/build-fpk.sh`](scripts/build-fpk.sh) 在 Ubuntu 24.04 / 24.04-arm 上原生编译 nginx，再打包 `fonu_<version>_<arch>.fpk`。

| 方式 | 说明 |
|------|------|
| **正式发版** | 推送 `v*` tag 后，`.fpk` 会出现在对应 GitHub Release 的 **Assets** |
| **手动试包** | Actions → **Build FPK** → 填 `version`；可选 `nginx_version`（默认最新 1.30.x）；产物在预发布 **`fpk-v<version>`**，若已有 `v<version>` Release 会同时挂到该版本 Assets |
| **本地 Linux** | 安装 gcc/make、libpcre2-dev、libssl-dev、fnpack 后执行 `packaging/fonu-fpk/scripts/build-fpk.sh amd64 1.2.2` |

产物始终是 **`.fpk`**；请只在 Release **Assets** 下载，不要用 Actions **Artifacts**（那是 zip 外壳）。

## 安装与访问

```bash
appcenter-cli install-fpk fonu_1.2.2_amd64.fpk
```

| 入口 | 默认地址（安装时可改） |
|------|------|
| 管理后台 | `http://<NAS-IP>:6893` |
| HTTP 反代 | `http://<NAS-IP>:18080` |
| HTTPS 反代 | `https://<NAS-IP>:9443` |

安装向导中自定义管理员用户名、密码和管理后台 / HTTP / HTTPS 端口。未在向导中设账号时，打开管理后台会进入 `/setup`。共享目录在「应用文件 / fonu」下的 `etc`、`www`、`certs`；NAS 其他文件夹在应用设置「访问权限」中添加。卸载时可选择是否删除全部数据。

## 从 Docker 迁到原生 FPK

Docker 把数据放在 volume 的 `/data`（飞牛上常见为 **`/vol1/@appdata/fonu`** 一类路径）。原生 FPK 的默认数据目录是 **`/var/apps/fonu/var/`**（`TRIM_PKGVAR`），与「应用文件 / fonu」**不是同一路径**。

| 现象 | 原因 | 处理 |
|------|------|------|
| Nginx 报 `mime.types` / `/etc/nginx/...` | 旧数据里生成的 `nginx.conf` 或 `fonu.env` 仍带 Docker 的 `FONU_NGINX_MIME_TYPES=/etc/nginx/mime.types` | 删掉 `fonu.env` 里该行；重启应用后在「反向代理」里点一次保存以重生成配置（或删除 `var/nginx/nginx.conf` 后重启） |
| DDNS「凭据无法解密」 | FPK 在 `etc/session_secret` 生成了新密钥，与 Docker 里加密 Token 用的 `FONU_SESSION_SECRET` 不一致 | **二选一**：在 DDNS 里重新保存各服务商 API Token；或把 Docker 用的 `FONU_SESSION_SECRET` 写入 `/var/apps/fonu/etc/session_secret`（或 `fonu.env`）后重启 |
| 通知 Telegram/SMTP 报「未配置」 | 同上，Bot Token 等也在库里加密保存 | 在 **系统设置 → 通知** 重新保存 Token，或对齐 `session_secret` |
| 设置页 **备份导入**（tar.gz） | 只恢复 `fonu.db` / `certs` / `nginx`，**不含** `etc/session_secret` | 导入后仍需 **重填 DDNS/通知密钥** 或写入 Docker 的 `FONU_SESSION_SECRET` |
| `/vol1/@appshare/fonu` 为空 | 正常：共享目录只用于可选的 `etc` / `www` / `certs`，数据库与 nginx 配置在 **`var/`** | 迁移时把 Docker volume **整包复制到 `var/`**，不要指望出现在应用文件里 |

推荐步骤（SSH）：

1. 停掉 Docker 版 Fonu，安装/启用 FPK。
2. 将原 Docker volume 内容复制到 `/var/apps/fonu/var/`（保留 `fonu.db`、`certs/`、`nginx/` 等）。
3. 若需保留 DDNS/通知里已存的密文：把原 `FONU_SESSION_SECRET` 写入 `/var/apps/fonu/etc/session_secret`（600 权限）。
4. 确认 `/var/apps/fonu/etc/fonu.env` **没有** `FONU_DATA_DIR=...@appdata...` 和 `FONU_NGINX_MIME_TYPES=/etc/nginx/...`（有则删掉对应行）。
5. 应用中心重启 Fonu；管理端打开反向代理任意保存一次，或查看 `var/logs/app.log` 无 Nginx 错误后再测 DDNS。

## 数据与日志位置

| 内容 | 位置 |
|------|------|
| 应用文件共享（etc / www / certs） | 文件管理器 → **应用文件 / fonu/** |
| 数据库、证书、Nginx 配置、程序日志 | **`/var/apps/fonu/var/`**（`TRIM_PKGVAR`，不在应用文件里） |
| 启动脚本日志 | `/var/apps/fonu/var/fonu.log` |
| 后台「日志中心」来源 | `/var/apps/fonu/var/logs/app.log` 等 |

## 证书映射：用户操作指南（飞牛 FPK）

### 什么时候不用映射

- 证书只给 **Fonu 自带反代 HTTPS** 用：不用开「证书映射」，PEM 已在 **`var/certs/`**，保存反代规则即可。

### 什么时候需要映射

- 要在 **文件管理 → 应用文件** 里看到 PEM 副本；
- 要让 **其他程序** 读同目录下的证书（且该目录已对 `fonu` 授权）；
- **不要**把目标填成 `/etc/...`、飞牛系统证书目录（`/usr/trim/...`）或 Docker 时代的 `/etc/app/certs`——FPK 下 `fonu` 用户通常写不进去。

### 方式 A：映射到「应用文件 / fonu / certs」（推荐）

1. 确认 **应用设置 → 访问权限** 里已有 **应用文件/fonu/certs**（安装 FPK 后一般自动有，读写）。
2. （可选）SSH 查真实绝对路径（界面里填的是 Linux 路径，不是「应用文件/…」中文）：
   ```bash
   ls -la /var/apps/fonu/shares/
   # 或应用运行时：
   echo "$TRIM_DATA_SHARE_PATHS"
   ```
   找到名字里带 **`fonu/certs`** 或 **`certs`** 的那一段，例如常见为  
   `/vol1/@appshare/fonu/certs`（以你机器上 `echo` 输出为准）。
3. 管理后台 **证书管理** → 选中证书 → **编辑**。
4. 打开 **证书映射** → **目标目录** 填上一步的目录（只填 **目录**，不要填到某个 `.pem` 文件）。
5. 保存后点 **立即同步**（或等续期后自动同步）。  
   Fonu 会在该目录下再建 **域名子目录**（如 `wildcard.muxui.com/`），里面是 `fullchain.pem` / `privkey.pem`。
6. 在 **文件管理 → 应用文件 → fonu → certs** 下查看是否出现对应文件夹。

**重载命令**：FPK 内置 Nginx 一般 **留空** 即可（反代已读 `var/certs`）。不要填 Docker 的 `docker exec …`。

### 方式 B：映射到其他文件夹（例如自建的 `/vol1/1000/my-certs`）

1. **应用中心 → Fonu → 应用设置 → 访问权限 → 「+ 添加」**，选中目标文件夹并授予 **读写**（不能选整盘 `/`）。
2. 在文件夹 **详细信息** 里 **复制原始路径**（或 SSH 下 `readlink -f`）。
3. 证书 **编辑 → 证书映射 → 目标目录** 粘贴该绝对路径 → 保存 → **立即同步**。
4. 其他程序从该文件夹读取；Fonu 仍保留 `var/certs/` 里的主副本。

### 方式 C：要给飞牛「系统 SSL」用

Fonu **不能**直接写入系统证书库。推荐：

1. 用 **方式 A 或 B** 把 PEM 同步到你有权目录；
2. 再用 **管理员 SSH / 计划任务** 把文件 **复制或软链** 到 `network_cert_all.conf` 里对应路径，并重载系统服务（社区常见做法，需自行承担风险）。

### 界面上的「证书源目录」

- 显示的是 Fonu **当前运行时** 的 `var/certs/.../域名目录`，不是 Docker 时代的 `/data/...`。
- 新版管理端会在映射区域提示 **建议路径**（来自 `TRIM_DATA_SHARE_PATHS`）；旧包可手动按方式 A 第 2 步查询。

### 导入 Docker 备份后

映射配置若在库里会保留，但 **DDNS/通知密钥** 可能需重填；映射目标若仍写 Docker 路径，请按上面改为 **方式 A/B** 的路径。

卸载时选「删除所有数据」会清理 `var/`、`etc/` 以及 `TRIM_DATA_SHARE_PATHS` 指向的共享目录；若未生效，查看卸载过程日志（`TRIM_TEMP_LOGFILE` 或应用中心任务详情）。
