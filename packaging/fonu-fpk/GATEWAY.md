# 统一网关入口（Phase 4 / 未实现）

当前 FPK 使用 **HTTP 6893 + `app/ui/config` iframe**，与 Docker host 模式一致。

若后续支持飞牛 **统一网关** `/app/fonu`，需要：

1. Vite `base: '/app/fonu/'`（或 `VITE_BASE` 构建参数）
2. Vue Router / API 路径适配子路径
3. Go 监听 `${TRIM_APPDEST}/app.sock`（扩展 `FONU_LISTEN` 或专用环境变量）
4. `manifest` 可省略 `service_port`，`app/ui/config` 使用 `gatewayPrefix` + `gatewaySocket`
5. 证书申请 SSE、日志流等在网关后做真机回归

本仓库 Phase 1–3 不包含上述改动。
