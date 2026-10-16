# Fonu FPK 真机验证清单

## 安装报「应用包不符合系统要求」

1. **只装 Release Assets 里的 `.fpk`**（约 37MB，文件头 `1F 8B`），不要装 Actions 的 zip（文件头 `PK`）。
2. **架构**：NAS 上执行 `uname -m` → `x86_64` 用 `_amd64.fpk`；`aarch64` 用 `_arm64.fpk`。
3. **系统版本**：manifest 要求 **fnOS ≥ 1.1.3100**（与 [nginx-web-fnos](https://github.com/chenpingonline/nginx-web-fnos) 一致）。低于此版本请先升级系统。
4. **对照**：在同一台 NAS 试装官方第三方 [nginx-web x86_64.fpk](https://github.com/chenpingonline/nginx-web-fnos/releases)。若也失败，多为系统/权限问题；若只有 Fonu 失败，把 `appcenter-cli install-fpk` 完整输出发维护者。
5. **SSH 安装**（看真实错误）：`appcenter-cli install-fpk /path/to/fonu_1.1.9_amd64.fpk`
4. **用最新构建**：重新跑 **Build FPK**（`ref=main`），不要装旧 run 里打的包。
5. **看详细错误**（SSH）：`appcenter-cli install-fpk /path/to/fonu_xxx_amd64.fpk`

在飞牛 fnOS 上安装对应架构的 `.fpk` 后逐项确认。

## 安装与生命周期

- [ ] `appcenter-cli install-fpk` 无报错
- [ ] 应用中心显示 **运行中**，停止后再启动状态正常（`cmd/main status` 退出码 0/3）
- [ ] `fonu.log` 有 `application started`，无反复崩溃

## 管理端（6893）

- [ ] 浏览器或桌面图标可打开登录页
- [ ] 使用日志中的初始 `admin` 密码登录成功
- [ ] 设置中可修改管理员密码

## 反向代理与 nginx

- [ ] 新建 HTTP 规则默认监听 **18080**（非 80）
- [ ] 新建 HTTPS 规则默认监听 **9443**（非 443）
- [ ] 保存后 `TRIM_PKGVAR/nginx/` 下生成配置，反代可访问内网目标
- [ ] 停用/删除规则后 nginx 重载无报错

## 证书与 DDNS

- [ ] 已配置 DNS 凭证时可申请测试证书（或 dry-run 流程到验证步骤）
- [ ] DDNS 任务可保存（不要求公网环境必成功）

## FRP

- [ ] 填写无效 frps 时应用仍可运行；保存合法配置后 `frpc` 进程可按需启停
- [ ] `TRIM_PKGVAR/frp/` 下生成 `frpc.toml`

## 持久化

- [ ] 应用停止并再次启动后，反代规则与证书列表仍在
- [ ] 卸载前导出/备份 `TRIM_PKGVAR`（含 `fonu.db`）

## 架构

- [ ] x86_64 设备仅安装 `*_amd64.fpk`；ARM 设备仅安装 `*_arm64.fpk`
