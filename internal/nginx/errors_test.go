package nginx

import (
	"os"
	"path/filepath"
	"strings"
	"strconv"
	"testing"

	"github.com/fonu/fonu/internal/config"
)

func TestEnsureErrorPagesWritesAssets(t *testing.T) {
	dir := t.TempDir()
	cfg := config.Config{DataDir: dir}

	if err := EnsureErrorPages(cfg); err != nil {
		t.Fatal(err)
	}

	errorsDir := cfg.ErrorsDir()
	for _, name := range []string{"403.html", "404.html", "429.html", "500.html", "502.html", "503.html", "error.png", "429.png"} {
		path := filepath.Join(errorsDir, name)
		info, err := os.Stat(path)
		if err != nil {
			t.Fatalf("missing %s: %v", name, err)
		}
		if info.Size() == 0 {
			t.Fatalf("%s is empty", name)
		}
	}

	for _, code := range []int{403,404,429,500,502,503} {
		num:=strconv.Itoa(code)
		html,err:=os.ReadFile(filepath.Join(errorsDir,num+".html"))
		if err!=nil{t.Fatal(err)}
		body:=string(html)
		for _,want:=range []string{"<title>"+num,">"+num+"</span>","prefers-color-scheme:dark","Fonu","noindex,nofollow","viewport"} {
			if !strings.Contains(body,want){t.Fatalf("%s.html missing %q",num,want)}
		}
		for _,unwanted:=range []string{"/fonu-errors/","<script","https://","<img"} {
			if strings.Contains(body,unwanted){t.Fatalf("%s.html has external dependency %q",num,unwanted)}
		}
	}
}
