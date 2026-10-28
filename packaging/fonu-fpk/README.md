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
| **仅 FPK（推荐试包）** | Actions → **Build FPK** → 填 `version`；可选 `nginx_version`（默认最新 1.30.x）；产物在预发布 **`fpk-v<version>`** |
| **本地 Linux** | 安装 gcc/make、libpcre2-dev、libssl-dev、fnpack 后执行 `packaging/fonu-fpk/scripts/build-fpk.sh amd64 1.2.0` |

产物始终是 **`.fpk`**；请只在 Release **Assets** 下载，不要用 Actions **Artifacts**（那是 zip 外壳）。

## 安装与访问

```bash
appcenter-cli install-fpk fonu_1.2.0_amd64.fpk
```

| 入口 | 默认地址（安装时可改） |
|------|------|
| 管理后台 | `http://<NAS-IP>:6893` |
| HTTP 反代 | `http://<NAS-IP>:18080` |
| HTTPS 反代 | `https://<NAS-IP>:9443` |

安装向导中自定义管理员用户名、密码和管理后台 / HTTP / HTTPS 端口。未在向导中设账号时，打开管理后台会进入 `/setup`。共享目录为 `etc`、`www`、`certs`；NAS 其他文件夹在应用设置「访问权限」中添加。

## 统一网关

未实现，见 [GATEWAY.md](./GATEWAY.md)。
