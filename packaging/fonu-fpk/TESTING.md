# Fonu FPK 真机验证清单

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
