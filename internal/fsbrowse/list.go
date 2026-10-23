package fsbrowse

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
)

const maxEntries = 500

// DirEntry is one child directory under ListResult.Path.
type DirEntry struct {
	Name string `json:"name"`
	Path string `json:"path"`
}

// ListResult is the response for listing directories at Path.
type ListResult struct {
	Path    string     `json:"path"`
	Parent  string     `json:"parent"`
	Entries []DirEntry `json:"entries"`
}

// ListDirectories returns immediate child directories of dir. Empty dir uses the platform root.
func ListDirectories(dir string) (ListResult, error) {
	dir = strings.TrimSpace(dir)
	if dir == "" {
		dir = defaultRoot()
	}
	dir = filepath.Clean(dir)
	if !filepath.IsAbs(dir) {
		return ListResult{}, fmt.Errorf("路径必须为绝对路径")
	}
	info, err := os.Stat(dir)
	if err != nil {
		return ListResult{}, fmt.Errorf("无法访问目录：%s", dir)
	}
	if !info.IsDir() {
		return ListResult{}, fmt.Errorf("不是目录：%s", dir)
	}

	entries, err := readSubdirs(dir)
	if err != nil {
		return ListResult{}, err
	}
	parent := parentDir(dir)
	return ListResult{
		Path:    dir,
		Parent:  parent,
		Entries: entries,
	}, nil
}

func readSubdirs(dir string) ([]DirEntry, error) {
	items, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("读取目录失败：%s", err.Error())
	}
	out := make([]DirEntry, 0, len(items))
	for _, item := range items {
		if !item.IsDir() {
			continue
		}
		name := item.Name()
		if name == "" || name == "." || name == ".." {
			continue
		}
		out = append(out, DirEntry{
			Name: name,
			Path: filepath.Join(dir, name),
		})
		if len(out) >= maxEntries {
			break
		}
	}
	sort.Slice(out, func(i, j int) bool {
		return strings.ToLower(out[i].Name) < strings.ToLower(out[j].Name)
	})
	return out, nil
}

func defaultRoot() string {
	if runtime.GOOS == "windows" {
		return "C:\\"
	}
	return "/"
}

func parentDir(dir string) string {
	dir = filepath.Clean(dir)
	root := defaultRoot()
	if dir == root {
		return ""
	}
	if runtime.GOOS != "windows" && dir == "/" {
		return ""
	}
	parent := filepath.Dir(dir)
	if parent == dir {
		return ""
	}
	return parent
}
