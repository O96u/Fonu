package nginx

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/fonu/fonu/internal/config"
)

func EnsureErrorPages(cfg config.Config) error {
	dir := cfg.ErrorsDir()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	ensureNginxReadable(dir, true)

	pages := []struct {
		file, title, message, hint string
		code int
	}{
		{"403.html", "访问受限", "当前请求未通过此服务的访问策略。", "请检查访问权限、IP 白名单或联系服务管理员。", 403},
		{"404.html", "页面不存在", "当前地址不存在，或资源已被移动。", "请确认访问地址和路径是否正确。", 404},
		{"429.html", "请求过于频繁", "请求速率或连接数可能已达到限制。", "请稍后重试，避免频繁刷新页面。", 429},
		{"500.html", "服务发生异常", "服务处理请求时遇到了意外错误。", "请稍后重试，或联系服务管理员。", 500},
		{"502.html", "上游服务响应异常", "反向代理未能从目标服务获得有效响应。", "请检查目标服务是否启动及网络是否连通。", 502},
		{"503.html", "服务暂不可用", "服务当前无法处理请求。", "服务可能正在维护或过载，请稍后重试。", 503},
	}
	for _, p := range pages {
		html := errorPageHTML(p.code, p.title, p.message, p.hint)
		target := filepath.Join(dir, p.file)
		if err := os.WriteFile(target, []byte(html), 0o644); err != nil {
			return err
		}
		ensureNginxReadable(target, false)
	}

	for _, name := range []string{"error.png", "429.png"} {
		if err := ensureErrorAsset(dir, name); err != nil {
			return err
		}
	}
	return nil
}

func ensureErrorAsset(dir, name string) error {
	target := filepath.Join(dir, name)
	if src := findErrorAsset(name); src != "" {
		if err := copyFileIfChanged(src, target); err != nil {
			return err
		}
	} else if err := writeEmbeddedErrorAsset(name, target); err != nil {
		return err
	}
	if err := verifyErrorAsset(target); err != nil {
		return err
	}
	ensureNginxReadable(target, false)
	return nil
}

func verifyErrorAsset(path string) error {
	info, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("error asset missing after write: %s: %w", path, err)
	}
	if info.Size() == 0 {
		return fmt.Errorf("error asset is empty: %s", path)
	}
	return nil
}

func writeEmbeddedErrorAsset(name, target string) error {
	data, err := errorAssetFS.ReadFile("assets/" + name)
	if err != nil {
		return fmt.Errorf("embedded error asset %s: %w", name, err)
	}
	return os.WriteFile(target, data, 0o644)
}

func findErrorAsset(name string) string {
	candidates := []string{
		filepath.Join("web", "assets", "image", name),
		filepath.Join("/app", "web", "assets", "image", name),
		filepath.Join("internal", "nginx", "assets", name),
	}
	for _, c := range candidates {
		if _, err := os.Stat(c); err == nil {
			return c
		}
	}
	return ""
}

func copyFileIfChanged(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, in)
	return err
}

// Self-contained, no remote fonts, images or JavaScript; works even when upstream is down.
func errorPageHTML(code int, title, message, hint string) string {
 return fmt.Sprintf(`<!doctype html>
<html lang="zh-CN">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width,initial-scale=1">
<meta name="robots" content="noindex,nofollow">
<title>%d · %s | Fonu</title>
<style>
:root{color-scheme:light dark;--bg:#f8fafc;--panel:#fff;--text:#111827;--muted:#6b7280;--border:#e5e7eb;--brand:#047857;--soft:#ecfdf5;--line:#10b981}
*{box-sizing:border-box}body{margin:0;min-height:100vh;min-height:100dvh;display:grid;place-items:center;padding:24px;background:var(--bg);color:var(--text);font:15px/1.65 -apple-system,BlinkMacSystemFont,"Segoe UI","PingFang SC","Microsoft YaHei",sans-serif}
main{width:min(100%%,560px);border:1px solid var(--border);border-radius:20px;background:var(--panel);padding:32px;box-shadow:0 12px 36px rgba(15,23,42,.05)}
.brand{display:flex;align-items:center;gap:9px;font-size:16px;font-weight:700}.mark{display:grid;place-items:center;width:34px;height:34px;border-radius:10px;background:var(--soft);color:var(--brand)}.label{margin-left:auto;color:var(--muted);font-size:12px;font-weight:400}
.status{display:flex;align-items:center;gap:14px;margin:36px 0 8px}.code{font-size:64px;font-weight:750;line-height:1;letter-spacing:-.06em;color:var(--brand);font-variant-numeric:tabular-nums}.status span:last-child{color:var(--muted);font-size:12px;border-left:1px solid var(--border);padding-left:14px;letter-spacing:.08em}
h1{font-size:25px;line-height:1.4;margin:16px 0 10px;font-weight:700}p{color:var(--muted);margin:0;overflow-wrap:anywhere}.hint{margin:24px 0 0;padding:14px 16px;border-left:3px solid var(--line);border-radius:6px;background:var(--soft);color:var(--text);font-size:13px}.hint strong{display:block;margin-bottom:3px;color:var(--brand)}footer{display:flex;justify-content:space-between;gap:10px;flex-wrap:wrap;border-top:1px solid var(--border);margin-top:28px;padding-top:16px;color:var(--muted);font-size:12px}
@media(prefers-color-scheme:dark){:root{--bg:#0f172a;--panel:#1e293b;--text:#f1f5f9;--muted:#94a3b8;--border:#334155;--brand:#6ee7b7;--soft:#17392f;--line:#10b981}}
@media(max-width:420px){body{padding:16px}main{padding:24px}.code{font-size:54px}h1{font-size:22px}.label{display:none}}
</style>
</head>
<body>
<main>
<div class="brand"><span class="mark" aria-hidden="true"><svg width="22" height="22" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.7" stroke-linecap="round" stroke-linejoin="round"><rect x="3" y="4" width="7" height="7" rx="2"/><rect x="14" y="13" width="7" height="7" rx="2"/><path d="M10 7.5h5a3 3 0 0 1 3 3V13M14 16.5H9a3 3 0 0 1-3-3V11"/></svg></span>Fonu<span class="label">反向代理</span></div>
<div class="status"><span class="code">%d</span><span>HTTP<br>STATUS</span></div>
<h1>%s</h1>
<p>%s</p>
<div class="hint"><strong>你可以尝试</strong>%s</div>
<footer><span>Fonu · 服务访问提示</span><span>HTTP %d</span></footer>
</main>
</body>
</html>`,code,title,code,title,message,hint,code)
}
