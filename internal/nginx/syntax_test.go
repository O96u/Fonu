package nginx

import "testing"

func TestValidateConfigSyntax(t *testing.T) {
	valid := `server {
    listen 80;
    location / {
        proxy_pass http://127.0.0.1:8080;
    }
}`
	if err := ValidateConfigSyntax(valid); err != nil {
		t.Fatalf("expected valid config: %v", err)
	}

	cases := []struct {
		name    string
		content string
	}{
		{"garbage line", "server {\n    1 1\n}\n"},
		{"missing semicolon", "server {\n    listen 80\n}\n"},
		{"unbalanced brace", "server {\n    listen 80;\n"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if err := ValidateConfigSyntax(tc.content); err == nil {
				t.Fatalf("expected error for %q", tc.name)
			}
		})
	}

	t.Run("allows comment lines", func(t *testing.T) {
		content := `# 规则：测试
server {
    # 监听 80
    listen 80;
}
# HTTP 跳转
`
		if err := ValidateConfigSyntax(content); err != nil {
			t.Fatalf("expected comments to pass: %v", err)
		}
	})
}
