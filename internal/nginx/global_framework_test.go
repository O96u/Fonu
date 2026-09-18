package nginx

import (
	"strings"
	"testing"

	"github.com/fonu/fonu/internal/config"
)

func TestGenerateGlobalFrameworkIncludesComments(t *testing.T) {
	cfg := config.Config{DataDir: t.TempDir()}
	out := GenerateGlobalFramework(cfg)
	for _, want := range []string{
		"# 以下部分由 Fonu 自动生成",
		"# 主进程",
		"# MIME 类型",
		"# 访问日志格式与路径",
		"# 传输优化",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("missing comment %q in:\n%s", want, out)
		}
	}
	if strings.Contains(out, "client_max_body_size") {
		t.Fatalf("framework should not include client_max_body_size:\n%s", out)
	}
}

func TestDefaultGlobalHTTPSnippet(t *testing.T) {
	out := DefaultGlobalHTTPSnippet()
	if !strings.Contains(out, "client_max_body_size 50m;") {
		t.Fatalf("missing upload limit in snippet:\n%s", out)
	}
	if !strings.Contains(out, "# 上传大小限制") {
		t.Fatalf("missing comment in snippet:\n%s", out)
	}
}
