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
		file, sprite, bgPos, title, msg string
		code                            int
	}{
		{"403.html", "s403", "0% 0%", "访问被拒绝", "你没有权限访问此资源。", 403},
		{"404.html", "s404", "100% 0%", "页面不存在", "请求的地址不存在或已被移除。", 404},
		{"500.html", "s500", "0% 100%", "服务异常", "服务器遇到错误，请稍后再试。", 500},
		{"503.html", "s503", "100% 100%", "服务维护中", "服务暂时不可用，请稍后再试。", 503},
		{"429.html", "429", "", "请求过于频繁", "你的请求次数过多，请稍后再试。", 429},
	}
	for _, p := range pages {
		html := errorPageHTML(p.code, p.title, p.msg, p.sprite, p.bgPos)
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
	ensureNginxReadable(target, false)
	return nil
}

func writeEmbeddedErrorAsset(name, target string) error {
	data, err := errorAssetFS.ReadFile("assets/" + name)
	if err != nil {
		return nil
	}
	return os.WriteFile(target, data, 0o644)
}

func findErrorAsset(name string) string {
	candidates := []string{
		filepath.Join("web", "assets", "image", name),
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

func errorPageHTML(code int, title, message, sprite, bgPos string) string {
	var imgBlock string
	if sprite == "429" {
		imgBlock = `<img class="illus" src="429.png" alt="">`
	} else {
		imgBlock = fmt.Sprintf(`<div class="sprite" role="img" aria-label="%d"></div>`, code)
	}
	return fmt.Sprintf(`<!DOCTYPE html>
<html lang="zh-CN">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width,initial-scale=1">
<title>%d - %s</title>
<style>
*{box-sizing:border-box;margin:0;padding:0}
body{min-height:100vh;display:flex;align-items:center;justify-content:center;font-family:-apple-system,BlinkMacSystemFont,"Segoe UI",Roboto,sans-serif;background:#0f1419;color:#e2e8f0;padding:24px}
.card{text-align:center;max-width:420px;width:100%%}
.sprite{width:280px;height:186px;margin:0 auto 24px;background:url(error.png) no-repeat;background-size:200%% 200%%;background-position:%s}
.illus{width:280px;height:auto;margin:0 auto 24px;display:block}
.code{font-size:14px;font-weight:600;color:#22c55e;letter-spacing:.08em;margin-bottom:8px}
h1{font-size:22px;font-weight:600;margin-bottom:8px}
p{font-size:14px;color:#94a3b8;line-height:1.6}
@media (prefers-color-scheme:light){body{background:#f8fafc;color:#1e293b}p{color:#64748b}}
</style>
</head>
<body>
<div class="card">
%s
<div class="code">%d</div>
<h1>%s</h1>
<p>%s</p>
</div>
</body>
</html>`, code, title, bgPos, imgBlock, code, title, message)
}
