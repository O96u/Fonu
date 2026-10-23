package fsbrowse

import (
	"os"
	"path/filepath"
	"testing"
)

func TestListDirectories(t *testing.T) {
	root := t.TempDir()
	sub := filepath.Join(root, "alpha")
	if err := os.Mkdir(sub, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "file.txt"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}

	res, err := ListDirectories(root)
	if err != nil {
		t.Fatal(err)
	}
	if res.Path != root {
		t.Fatalf("path %q", res.Path)
	}
	if len(res.Entries) != 1 || res.Entries[0].Name != "alpha" {
		t.Fatalf("entries %+v", res.Entries)
	}

	child, err := ListDirectories(sub)
	if err != nil {
		t.Fatal(err)
	}
	if child.Parent != root {
		t.Fatalf("parent %q", child.Parent)
	}
}

func TestListDirectoriesRejectsRelative(t *testing.T) {
	_, err := ListDirectories("relative/path")
	if err == nil {
		t.Fatal("expected error")
	}
}
