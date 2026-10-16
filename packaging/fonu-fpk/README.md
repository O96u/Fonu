# Fonu 飞牛 fnOS 原生 FPK

真·原生安装包：内置 `fonu`（Go + 嵌入 Vue 管理端）、`nginx`、`frpc`，数据目录为应用 `TRIM_PKGVAR`。

模板与脚本在 Git 仓库中；**`.cache/`、`packaging/dist/`** 不提交。

## 构建

[`scripts/build-fpk.sh`](scripts/build-fpk.sh) 在 Ubuntu CI 中执行，产出 `fonu_<version>_<arch>.fpk`。

| 方式 | 说明 |
|------|------|
| **仅 FPK（推荐试包）** | Actions → **Build FPK** → Run workflow：`ref`（如 `main`）、`version`（如 `1.1.10`）；完成后在 Artifacts 下载 amd64 / arm64 |
| **发版一条龙** | 推送 `v*` tag → **GitHub Release** 工作流：创建 Release + 打 FPK 并上传 |

单独打 FPK 时默认**不**创建 Release、**不**构建 Docker。若勾选 `upload_to_release`，需该 `v*` tag 的 Release 已存在。

## 安装与访问

```bash
appcenter-cli install-fpk fonu_1.1.9_amd64.fpk
```

| 入口 | 地址 |
|------|------|
| 管理后台 | `http://<NAS-IP>:6893` |
| HTTP 反代默认 | `http://<NAS-IP>:18080` |
| HTTPS 反代默认 | `https://<NAS-IP>:9443` |

初始 `admin` 密码见 `TRIM_PKGVAR/fonu.log`。

## 统一网关

未实现，见 [GATEWAY.md](./GATEWAY.md)。
