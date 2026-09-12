# Changelog

All notable changes to this project will be documented in this file.

## [v0.1.0] - 2026-09-12

### 新增

- **DDNS 多域名**：支持多条 DDNS 配置，独立管理根域名与记录
- **证书多域名**：申请/续签支持多域名与通配符（`*.example.com`）
- **ACME 颁发机构**：申请与续签时可选择 Let's Encrypt 正式/测试环境
- **证书导入**：支持粘贴 PEM 或指定文件路径导入自签/已有证书
- **Docker 发版**：推送 `v*` 标签自动构建并推送 `linux/amd64`、`linux/arm64` 镜像到 Docker Hub
- **表格操作规范**：所有列表操作按钮平铺展示（`renderTableRowActions`）

### 改进

- **仪表盘**：全新布局，ECharts 图表，状态卡片与日志预览
- **日志页**：分页（20 条/页），支持 `?tab=` 跳转，统一时间格式
- **证书页**：概览统计、申请弹窗优化、表格自适应宽度
- **布局**：顶栏时间靠左、操作靠右，内容区铺满主区域
- **主题**：深色模式与组件样式统一

### 数据库迁移

- `003_ddns_multi.sql` — DDNS 多配置
- `004_cert_acme_ca.sql` — 证书 ACME CA 字段
- `005_cert_domains.sql` — 证书多域名
