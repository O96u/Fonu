package logstore

import "testing"

func TestLocalizeSystemMessage(t *testing.T) {
	raw := map[string]any{
		"msg":    "application started",
		"listen": ":6893",
	}
	entry := localizeSystemEntry(SystemEntry{
		Level:   "INFO",
		Module:  "SYSTEM",
		Message: "application started",
	}, raw)
	if entry.Message != "应用已启动，监听 :6893" {
		t.Fatalf("unexpected message: %q", entry.Message)
	}
	if entry.Module != "系统" {
		t.Fatalf("unexpected module: %q", entry.Module)
	}
}

func TestLocalizeSystemMessageAdminBootstrap(t *testing.T) {
	raw := map[string]any{
		"msg":      "首次启动已创建管理员账户，请使用以下凭据登录，并在「设置」中尽快修改密码",
		"username": "admin",
		"password": "secret",
	}
	entry := localizeSystemEntry(SystemEntry{
		Message: "首次启动已创建管理员账户，请使用以下凭据登录，并在「设置」中尽快修改密码",
		Module:  "SYSTEM",
	}, raw)
	want := "首次启动已创建管理员账户，请使用以下凭据登录，并在「设置」中尽快修改密码（用户名：admin，密码：secret）"
	if entry.Message != want {
		t.Fatalf("unexpected message: %q", entry.Message)
	}
}

func TestLocalizeSystemMessageAPIRejected(t *testing.T) {
	raw := map[string]any{
		"msg":     "request rejected",
		"method":  "POST",
		"path":    "/api/settings/notify/test",
		"status":  400,
		"message": "Telegram Bot Token 未配置",
	}
	entry := localizeSystemEntry(SystemEntry{Message: "request rejected", Module: "API"}, raw)
	if entry.Message != "API 请求被拒绝（POST /api/settings/notify/test，状态 400）：Telegram Bot Token 未配置" {
		t.Fatalf("unexpected message: %q", entry.Message)
	}
	if entry.Module != "API" {
		t.Fatalf("unexpected module: %q", entry.Module)
	}
}
