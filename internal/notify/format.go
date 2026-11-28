package notify

import (
	"strings"
	"time"
)

func formatFooter(at time.Time) string {
	return "Fonu · " + at.Format("2006-01-02 15:04:05")
}

type eventMeta struct {
	icon     string
	category string
}

type AlertContent struct {
	Event     string
	Subject   string
	PushTitle string
	PlainBody string
	HTMLBody  string
	Category  string
	Icon      string
	Detail    string
	Timestamp time.Time
}

func FormatAlert(event, title, message string) AlertContent {
	return FormatAlertAt(event, title, message, time.Now())
}

func FormatAlertAt(event, title, message string, at time.Time) AlertContent {
	meta := eventMetaFor(event)
	title = strings.TrimSpace(title)
	message = strings.TrimSpace(message)
	pushTitle := strings.TrimSpace(meta.icon + " " + title)
	subject := "[Fonu] " + pushTitle
	footer := formatFooter(at)

	detail := message
	if detail == "" {
		detail = "暂无更多详情"
	}

	status, action := eventGuidance(event)
	body := "类型：" + meta.category + " · " + status + "\n\n" + detail
	if action != "" {body += "\n\n建议：" + action}
	plain := strings.Join([]string{pushTitle, "", body, "", footer}, "\n")
	html := formatTelegramHTML(pushTitle, body, at)

	return AlertContent{
		Event:     event,
		Subject:   subject,
		PushTitle: pushTitle,
		PlainBody: plain,
		HTMLBody:  html,
		Category:  meta.category,
		Icon:      meta.icon,
		Detail:    detail,
		Timestamp: at,
	}
}

// Guidance describes the event, not the delivery outcome.
func eventGuidance(event string) (status, action string) {
	switch event {
	case EventDDNSIPChange:
		return "已同步", "通常无需操作；如服务仍不可访问，请检查 DNS 解析及入口连通性。"
	case EventDDNSFailure:
		return "需要检查", "前往 DDNS 查看任务状态、失败域名及服务商凭证。"
	case EventCertExpiry:
		return "到期提醒", "前往证书管理检查到期时间、自动续签及 DNS 凭证。"
	case EventCertRenewSuccess:
		return "续签成功", "确认需要部署的服务已使用新证书。"
	case EventCertRenewFailure:
		return "需要处理", "前往证书管理查看失败原因，检查 DNS 凭证及验证记录。"
	case EventScan, EventCC, EventIPFrequentAccess:
		return "疑似异常", "前往日志中心的访问日志，按入口及来源 IP 核对流量；确认异常后可通过记录的黑名单操作限制当前规则。分布式高频访问建议结合限流与 CDN 防护。"
	case EventLoginFailure:
		return "登录异常", "核查登录日志；如非本人操作，请检查账号安全。"
	case EventNginxReloadFailure:
		return "需要处理", "查看反向代理发布状态及 Nginx 日志，确认实际运行配置。"
	case "test":
		return "手动测试", ""
	default:
		return "事件提醒", ""
	}
}

func eventMetaFor(event string) eventMeta {
	switch event {
	case EventDDNSIPChange:
		return eventMeta{"🌐", "DDNS"}
	case EventDDNSFailure:
		return eventMeta{"⚠️", "DDNS"}
	case EventCertExpiry:
		return eventMeta{"📅", "证书"}
	case EventCertRenewSuccess:
		return eventMeta{"✅", "证书"}
	case EventCertRenewFailure:
		return eventMeta{"❌", "证书"}
	case EventScan,EventCC:
        return eventMeta{"🛡️","安全"}
	case EventIPFrequentAccess:
		return eventMeta{"🛡️", "安全"}
	case EventLoginFailure:
		return eventMeta{"🔐", "安全"}
	case EventNginxReloadFailure:
		return eventMeta{"⚙️", "系统"}
	case "test":
		return eventMeta{"🔔", "系统"}
	default:
		return eventMeta{"📣", "告警"}
	}
}

func formatTelegramHTML(title, detail string, at time.Time) string {
	lines := []string{
		"<b>" + escapeHTML(title) + "</b>",
		"",
		"<pre>" + escapeHTML(detail) + "</pre>",
		"",
		"<i>" + escapeHTML(formatFooter(at)) + "</i>",
	}
	return strings.Join(lines, "\n")
}

func escapeHTML(s string) string {
	replacer := strings.NewReplacer(
		"&", "&amp;",
		"<", "&lt;",
		">", "&gt;",
	)
	return replacer.Replace(s)
}

func (c AlertContent) webhookPayload() map[string]string {
	return map[string]string{
		"event":     c.Event,
		"category":  c.Category,
		"icon":      c.Icon,
		"title":     strings.TrimPrefix(c.PushTitle, c.Icon+" "),
		"message":   c.Detail,
		"formatted": c.PlainBody,
		"time":      c.Timestamp.UTC().Format(time.RFC3339),
	}
}

func (c AlertContent) barkMessage() string {
	status, action := eventGuidance(c.Event)
	body := []string{"类型：" + c.Category + " · " + status, "", c.Detail}
	if action != "" {body = append(body, "", "建议：" + action)}
	return strings.Join(append(body, "", formatFooter(c.Timestamp)), "\n")
}
