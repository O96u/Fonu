package nginx

import (
	"fmt"
	"strings"
)

// ValidateConfigSyntax performs a lightweight syntax check when nginx -t is unavailable.
func ValidateConfigSyntax(content string) error {
	depth := 0
	for i, raw := range strings.Split(content, "\n") {
		trimmed := strings.TrimSpace(raw)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}
		line := stripInlineComment(trimmed)
		if line == "" {
			continue
		}
		depth += strings.Count(line, "{") - strings.Count(line, "}")
		if depth < 0 {
			return fmt.Errorf("第 %d 行：多余的 `}`", i+1)
		}
		if err := validateConfigLine(line, i+1); err != nil {
			return err
		}
	}
	if depth != 0 {
		return fmt.Errorf("配置块大括号未配对")
	}
	return nil
}

func stripInlineComment(line string) string {
	inSingle := false
	inDouble := false
	for i := 0; i < len(line); i++ {
		ch := line[i]
		switch ch {
		case '\'':
			if !inDouble {
				inSingle = !inSingle
			}
		case '"':
			if !inSingle {
				inDouble = !inDouble
			}
		case '#':
			if !inSingle && !inDouble {
				return strings.TrimSpace(line[:i])
			}
		}
	}
	return strings.TrimSpace(line)
}

func validateConfigLine(line string, lineNo int) error {
	if line == "}" || line == "{" {
		return nil
	}
	if strings.HasSuffix(line, "{") {
		return nil
	}
	if strings.HasSuffix(line, ";") {
		return nil
	}
	return fmt.Errorf("第 %d 行：无效的 Nginx 语法（应以 ; 或 { 结尾）", lineNo)
}
