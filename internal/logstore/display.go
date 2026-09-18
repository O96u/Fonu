package logstore

import (
	"fmt"
	"strings"
)

func localizeSystemEntry(entry SystemEntry, raw map[string]any) SystemEntry {
	msg := strings.TrimSpace(entry.Message)
	if msg == "" {
		return entry
	}
	entry.Module = localizeSystemModule(entry.Module)
	entry.Message = localizeSystemMessage(msg, raw)
	return entry
}

func localizeSystemModule(module string) string {
	switch strings.ToUpper(strings.TrimSpace(module)) {
	case "SYSTEM":
		return "系统"
	case "NGINX":
		return "Nginx"
	case "DDNS":
		return "域名解析"
	case "ACME", "CERT":
		return "证书"
	case "CHINA_CIDR":
		return "中国 IP 段"
	case "API":
		return "API"
	case "HTTP":
		return "HTTP"
	case "FRP":
		return "内网穿透"
	default:
		if module == "" {
			return "系统"
		}
		return module
	}
}

func localizeSystemMessage(msg string, raw map[string]any) string {
	if looksChinese(msg) {
		return appendSystemAttrs(msg, raw)
	}

	switch msg {
	case "application started":
		if listen := rawString(raw, "listen"); listen != "" {
			return fmt.Sprintf("应用已启动，监听 %s", listen)
		}
		return "应用已启动"
	case "application shutting down":
		return "应用正在关闭"
	case "nginx started":
		return "Nginx 已启动"
	case "nginx stopped":
		return "Nginx 已停止"
	case "nginx reloaded":
		return "Nginx 已重载"
	case "nginx reloaded existing master":
		return "Nginx 已重载（使用现有主进程）"
	case "nginx start skipped duplicate master, reloaded instead":
		return "检测到重复 Nginx 主进程，已改为重载"
	case "nginx reload failed, trying restart":
		return fmt.Sprintf("Nginx 重载失败，正在尝试重启：%s", rawString(raw, "error"))
	case "nginx signal reload failed, falling back to cli":
		return fmt.Sprintf("Nginx 信号重载失败，回退到命令行重载（PID %s）：%s", rawString(raw, "pid"), rawString(raw, "error"))
	case "nginx command failed":
		return fmt.Sprintf("Nginx 命令执行失败（%s）：%s", rawString(raw, "args"), rawString(raw, "error"))
	case "nginx start failed":
		return fmt.Sprintf("Nginx 启动失败（%s）：%s", rawString(raw, "args"), rawString(raw, "error"))
	case "nginx binary not found, applied syntax-only validation":
		return fmt.Sprintf("未找到 Nginx 可执行文件，仅进行配置语法校验（%s）", rawString(raw, "bin"))
	case "nginx binary not found, syntax-only validation":
		return fmt.Sprintf("未找到 Nginx 可执行文件，仅进行语法校验（%s）", rawString(raw, "bin"))
	case "initial nginx apply skipped":
		return fmt.Sprintf("初始 Nginx 配置应用已跳过：%s", rawString(raw, "error"))
	case "initial frpc bootstrap skipped":
		return fmt.Sprintf("初始 frpc 引导已跳过：%s", rawString(raw, "error"))
	case "nginx stop failed":
		return fmt.Sprintf("Nginx 停止失败：%s", rawString(raw, "error"))
	case "frpc stop failed":
		return fmt.Sprintf("frpc 停止失败：%s", rawString(raw, "error"))
	case "frpc started":
		return "frpc 已启动"
	case "frpc stopped":
		return "frpc 已停止"
	case "frpc reloaded":
		return "frpc 已重载"
	case "frpc reload failed, trying restart":
		return fmt.Sprintf("frpc 重载失败，正在尝试重启：%s", rawString(raw, "error"))
	case "frpc binary not found, skipping bootstrap":
		return fmt.Sprintf("未找到 frpc 可执行文件，已跳过引导（%s）", rawString(raw, "bin"))
	case "initial frpc apply skipped":
		return fmt.Sprintf("初始 frpc 配置应用已跳过：%s", rawString(raw, "error"))
	case "frpc exited during start":
		return fmt.Sprintf("frpc 启动过程中退出：%s", rawString(raw, "error"))
	case "china cidr update failed":
		return fmt.Sprintf("中国 IP 段更新失败：%s", rawString(raw, "error"))
	case "china cidr updated":
		return fmt.Sprintf("中国 IP 段已更新（IPv4 %s 条，IPv6 %s 条）", rawString(raw, "v4"), rawString(raw, "v6"))
	case "ddns update failed":
		return fmt.Sprintf("DDNS 更新失败（%s）：%s", rawString(raw, "domain"), rawString(raw, "error"))
	case "ddns ipv4 updated":
		return fmt.Sprintf("DDNS IPv4 已更新：%s → %s", rawString(raw, "domain"), rawString(raw, "ip"))
	case "ddns ipv6 updated":
		return fmt.Sprintf("DDNS IPv6 已更新：%s → %s", rawString(raw, "domain"), rawString(raw, "ip"))
	case "ddns records updated":
		return fmt.Sprintf("DDNS 记录已更新（服务商：%s）", rawString(raw, "provider"))
	case "notify load config failed":
		return fmt.Sprintf("通知配置加载失败：%s", rawString(raw, "error"))
	case "notify send failed":
		return fmt.Sprintf("通知发送失败（%s）：%s", rawString(raw, "event"), rawString(raw, "error"))
	case "notify test send failed":
		return fmt.Sprintf("通知测试发送失败：%s", rawString(raw, "error"))
	case "request failed":
		return fmt.Sprintf("API 请求失败（%s %s，状态 %s）：%s",
			rawString(raw, "method"), rawString(raw, "path"), rawString(raw, "status"), rawString(raw, "message"))
	case "request rejected":
		return fmt.Sprintf("API 请求被拒绝（%s %s，状态 %s）：%s",
			rawString(raw, "method"), rawString(raw, "path"), rawString(raw, "status"), rawString(raw, "message"))
	case "api request completed with client error":
		return fmt.Sprintf("API 请求返回客户端错误（%s %s，状态 %s，耗时 %sms）",
			rawString(raw, "method"), rawString(raw, "path"), rawString(raw, "status"), rawString(raw, "duration_ms"))
	case "api request failed":
		return fmt.Sprintf("API 请求返回服务端错误（%s %s，状态 %s，耗时 %sms）",
			rawString(raw, "method"), rawString(raw, "path"), rawString(raw, "status"), rawString(raw, "duration_ms"))
	case "certificate operation failed":
		return fmt.Sprintf("证书操作失败（%s）：%s", rawString(raw, "domain"), rawString(raw, "error"))
	case "certificate obtained":
		return fmt.Sprintf("证书已获取：%s（CA %s，到期 %s）",
			rawString(raw, "domain"), rawString(raw, "ca"), rawString(raw, "expires_at"))
	case "certificate imported":
		return fmt.Sprintf("证书已导入：%s（到期 %s）", rawString(raw, "domain"), rawString(raw, "expires_at"))
	case "certificate deleted":
		return fmt.Sprintf("证书已删除：%s", rawString(raw, "domain"))
	case "certificate renew failed":
		return fmt.Sprintf("证书续签失败（%s）：%s", rawString(raw, "domain"), rawString(raw, "error"))
	case "save cert error status failed":
		return fmt.Sprintf("保存证书错误状态失败（%s）：%s", rawString(raw, "domain"), rawString(raw, "error"))
	case "nginx reload after cert apply failed":
		return fmt.Sprintf("证书申请后 Nginx 重载失败：%s", rawString(raw, "error"))
	case "nginx reload after cert import failed":
		return fmt.Sprintf("证书导入后 Nginx 重载失败：%s", rawString(raw, "error"))
	case "nginx reload after cert delete failed":
		return fmt.Sprintf("证书删除后 Nginx 重载失败：%s", rawString(raw, "error"))
	case "nginx reload after cert renew failed":
		return fmt.Sprintf("证书续签后 Nginx 重载失败：%s", rawString(raw, "error"))
	case "save acme ca setting failed":
		return fmt.Sprintf("保存 ACME 颁发机构设置失败：%s", rawString(raw, "error"))
	case "save acme email failed":
		return fmt.Sprintf("保存 ACME 邮箱失败：%s", rawString(raw, "error"))
	case "remove cert files failed":
		return fmt.Sprintf("删除证书文件失败（%s）：%s", rawString(raw, "domain"), rawString(raw, "error"))
	default:
		return appendSystemAttrs(msg, raw)
	}
}

func appendSystemAttrs(msg string, raw map[string]any) string {
	if strings.Contains(msg, "首次启动已创建管理员账户") {
		username := rawString(raw, "username")
		password := rawString(raw, "password")
		if username != "" && password != "" {
			return fmt.Sprintf("%s（用户名：%s，密码：%s）", msg, username, password)
		}
	}
	if err := rawString(raw, "error"); err != "" && !strings.Contains(msg, err) {
		return fmt.Sprintf("%s：%s", msg, err)
	}
	return msg
}

func looksChinese(msg string) bool {
	for _, r := range msg {
		if r >= 0x4e00 && r <= 0x9fff {
			return true
		}
	}
	return false
}

func rawString(raw map[string]any, key string) string {
	v, ok := raw[key]
	if !ok || v == nil {
		return ""
	}
	switch t := v.(type) {
	case string:
		return t
	case float64:
		if t == float64(int64(t)) {
			return fmt.Sprintf("%d", int64(t))
		}
		return fmt.Sprintf("%v", t)
	case int:
		return fmt.Sprintf("%d", t)
	case int64:
		return fmt.Sprintf("%d", t)
	case bool:
		if t {
			return "true"
		}
		return "false"
	default:
		return fmt.Sprintf("%v", t)
	}
}
