# Fonu FPK 真机验证清单

## 安装报「应用包不符合系统要求」

安装提示本身不足以确定根因。请记录 NAS 的 `uname -m`、系统完整版本，以及 `appcenter-cli install-fpk /path/to/package.fpk` 的完整输出。

新构建分为原生 payload 编译和独立 package 阶段。后者从干净目录重新装配模板，校验 fonu、Nginx、frpc 的 ELF 架构后调用官方 fnpack，不从其他目录拾取旧 FPK。此前关于 CRLF、LICENSE 必然引发该提示的判断没有实机日志支持。

在飞牛 fnOS 上安装对应架构的 `.fpk` 后逐项确认。

## 安装与生命周期

- [ ] `appcenter-cli install-fpk` 无报错
- [ ] 应用中心显示 **运行中**，停止后再启动状态正常（`cmd/main status` 退出码 0/3）
- [ ] `fonu.log` 有 `application started`，无反复崩溃

## 管理端（6893）

- [ ] 浏览器或桌面图标可打开登录页
- [ ] 安装向导标题不重复「安装 Fonu」；可填写自定义管理员用户名、密码（含确认）和管理后台 / HTTP / HTTPS 端口
- [ ] 向导里设置的账号可直接登录；卸载重装后新密码立即生效
- [ ] 未在向导中设账号时，打开管理后台会进入 `/setup` 创建管理员
- [ ] 「应用文件 / fonu」下可见 `etc` / `www` / `certs`；访问权限可添加 NAS 文件夹
- [ ] 应用窗口打开后很快出现登录页，不应长时间空白
- [ ] 卸载向导询问是否删除所有数据；选择保留时重装可恢复
- [ ] 设置中可修改管理员用户名和密码；应用设置中可再改端口（需重启）

## 反向代理与 nginx

- [ ] 新建 HTTP 规则默认监听 **18080**（非 80）
- [ ] 新建 HTTPS 规则默认监听 **9443**（非 443）
- [ ] 保存后 `TRIM_PKGVAR/nginx/` 下生成配置，`TRIM_APPDEST/Nginx` 进程可访问内网目标
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
