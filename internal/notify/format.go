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

	plain := strings.Join([]string{
		pushTitle,
		strings.Repeat("─", 28),
		"",
		detail,
		"",
		strings.Repeat("─", 28),
		footer,
	}, "\n")

	html := formatTelegramHTML(pushTitle, detail, at)

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
	return strings.Join([]string{c.Detail, "", "────────", formatFooter(c.Timestamp)}, "\n")
}
