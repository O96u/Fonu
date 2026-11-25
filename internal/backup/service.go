package backup

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/fonu/fonu/internal/db"
	"github.com/fonu/fonu/internal/secret"
)

const pendingDirName = "restore-pending"

type Service struct {
	mu sync.Mutex
	dataDir    string
	onRestored func()
}

func New(dataDir string) *Service {
	return &Service{dataDir: dataDir}
}

func (s *Service) SetOnRestored(fn func()) {
	s.onRestored = fn
}

func (s *Service) NotifyRestored() {
	if s.onRestored != nil {
		go s.onRestored()
	}
}

func (s *Service) Export(ctx context.Context) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	tmpDir := filepath.Join(s.dataDir, "backups")
	if err := os.MkdirAll(tmpDir, 0o755); err != nil {
		return "", err
	}
	name := fmt.Sprintf("fonu-backup-%s-*.tar.gz", time.Now().Format("20060102-His"))
	file, err := os.CreateTemp(tmpDir, name)
	if err != nil {
		return "", err
	}
	outPath := file.Name()
	success := false
	defer func() { file.Close(); if !success { os.Remove(outPath) } }()

	gw := gzip.NewWriter(file)
	tw := tar.NewWriter(gw)

	entries := []string{
		"fonu.db",
		secret.KeyFileName,
		"certs",
		"nginx",
	}
	for _, entry := range entries {
		var err error
		if entry == "fonu.db" {
			err = addDatabaseSnapshot(ctx, tw, s.dataDir)
		} else {
			err = addToArchive(ctx, tw, s.dataDir, entry)
		}
		if err != nil {
			_ = tw.Close()
			_ = gw.Close()
			_ = os.Remove(outPath)
			return "", err
		}
	}

	if err := tw.Close(); err != nil {
		return "", err
	}
	if err := gw.Close(); err != nil {
		return "", err
	}
	if err := file.Close(); err != nil { return "", err }
	success = true
	return outPath, nil
}

func (s *Service) Restore(ctx context.Context, archivePath string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	file, err := os.Open(archivePath)
	if err != nil {
		return err
	}
	defer file.Close()

	pending := filepath.Join(s.dataDir, pendingDirName)
	if _, err := os.Stat(pending); err == nil { return fmt.Errorf("已有待恢复备份，请等待重启") } else if !os.IsNotExist(err) { return err }
	stage, err := os.MkdirTemp(s.dataDir, ".restore-stage-*")
	if err != nil { return err }
	defer os.RemoveAll(stage)

	gr, err := gzip.NewReader(file)
	if err != nil {
		_ = os.RemoveAll(stage)
		return fmt.Errorf("无效的备份文件")
	}
	defer gr.Close()

	tr := tar.NewReader(gr)
	wroteDB := false
	var total int64
	count := 0
	seen := make(map[string]bool)
	for {
		if err := ctx.Err(); err != nil {
			_ = os.RemoveAll(stage)
			return err
		}
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			_ = os.RemoveAll(stage)
			return err
		}
		count++
		if count > 100000 || header.Size < 0 || header.Size > (2<<30)-total {
			_ = os.RemoveAll(stage)
			return fmt.Errorf("备份解压大小或文件数量超出限制")
		}
		total += header.Size
		if header.Typeflag != tar.TypeReg && header.Typeflag != tar.TypeDir {
			continue
		}
		if !allowedEntry(header.Name) {
			continue
		}
		target, err := safeJoin(stage, header.Name)
		if err != nil {
			_ = os.RemoveAll(stage)
			return err
		}
		if seen[target] { return fmt.Errorf("备份包含重复条目") }
		seen[target] = true
		switch header.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(target, 0o755); err != nil {
				_ = os.RemoveAll(stage)
				return err
			}
		case tar.TypeReg:
			if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
				_ = os.RemoveAll(stage)
				return err
			}
			out, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o600)
			if err != nil {
				_ = os.RemoveAll(stage)
				return err
			}
			if _, err := io.Copy(out, tr); err != nil {
				out.Close()
				_ = os.RemoveAll(stage)
				return err
			}
			if err := out.Sync(); err != nil { out.Close(); return err }
			if err := out.Close(); err != nil {
				_ = os.RemoveAll(stage)
				return err
			}
			if filepath.ToSlash(header.Name) == "fonu.db" {
				wroteDB = true
			}
		}
	}
	// Consume the gzip footer to verify its checksum, even after tar EOF.
	if n, err := io.Copy(io.Discard, io.LimitReader(gr, (1<<20)+1)); err != nil { return err } else if n > 1<<20 { return fmt.Errorf("备份包含过多尾部数据") }
	if !wroteDB {
		_ = os.RemoveAll(stage)
		return fmt.Errorf("备份里没有数据库")
	}
	if key, err := os.ReadFile(filepath.Join(stage, secret.KeyFileName)); err == nil {
		if err := secret.ValidateStoredKey(string(key)); err != nil {
			_ = os.RemoveAll(stage)
			return err
		}
	} else if !os.IsNotExist(err) {
		_ = os.RemoveAll(stage)
		return err
	}
	// 进程还占着正在使用的库。先放到旁边校验，下次启动、打开数据库之前再换上去。
	if err := db.PrepareRestore(filepath.Join(stage, "fonu.db")); err != nil {
		_ = os.RemoveAll(stage)
		return fmt.Errorf("备份里的数据库不可用")
	}
	// Publish only a complete, verified archive; interrupted extraction is never applied.
	return os.Rename(stage, pending)
}

func allowedEntry(name string) bool {
	name = strings.TrimPrefix(filepath.ToSlash(strings.TrimSpace(name)), "./")
	switch {
	case name == "fonu.db" || name == secret.KeyFileName:
		return true
	case name == "certs" || strings.HasPrefix(name, "certs/"):
		return true
	case name == "nginx" || strings.HasPrefix(name, "nginx/"):
		return true
	default:
		return false
	}
}

func safeJoin(root, name string) (string, error) {
	normalized := strings.TrimSuffix(strings.TrimPrefix(filepath.ToSlash(name), "./"), "/")
	cleaned := filepath.Clean(filepath.FromSlash(normalized))
	if filepath.ToSlash(cleaned) != normalized { return "", fmt.Errorf("备份条目路径非规范") }
	if cleaned == "." || cleaned == ".." || strings.HasPrefix(cleaned, ".."+string(os.PathSeparator)) || filepath.IsAbs(cleaned) {
		return "", fmt.Errorf("备份条目非法")
	}
	target := filepath.Join(root, cleaned)
	rel, err := filepath.Rel(filepath.Clean(root), filepath.Clean(target))
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(os.PathSeparator)) {
		return "", fmt.Errorf("备份条目非法")
	}
	return target, nil
}

func ApplyPending(dataDir string) error {
	if err := recoverRestore(dataDir); err != nil { return err }
	pending := filepath.Join(dataDir, pendingDirName)
	staged := filepath.Join(pending, "fonu.db")
	if _, err := os.Stat(staged); err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	if err := db.PrepareRestore(staged); err != nil {
		_ = os.RemoveAll(pending)
		return fmt.Errorf("待恢复的数据库不可用")
	}

	return swapRestoreFiles(dataDir, pending, os.Rename)
}

func addDatabaseSnapshot(ctx context.Context, tw *tar.Writer, dataDir string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	src := filepath.Join(dataDir, "fonu.db")
	if _, err := os.Stat(src); err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	tmp, err := os.MkdirTemp("", "fonu-export-*")
	if err != nil { return err }
	defer os.RemoveAll(tmp)
	tmpPath := filepath.Join(tmp,"fonu.db")
	if err := db.Snapshot(src, tmpPath); err != nil {
		return err
	}
	info, err := os.Stat(tmpPath)
	if err != nil {
		return err
	}
	return addNamedFile(tw, "fonu.db", tmpPath, info)
}

func addToArchive(ctx context.Context, tw *tar.Writer, baseDir, rel string) error {
	path := filepath.Join(baseDir, rel)
	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	if info.IsDir() {
		return addDir(ctx, tw, baseDir, rel)
	}
	return addFile(tw, baseDir, rel, info)
}

func addDir(ctx context.Context, tw *tar.Writer, baseDir, rel string) error {
	root := filepath.Join(baseDir, rel)
	return filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if err := ctx.Err(); err != nil { return err }
		if d.IsDir() || !d.Type().IsRegular() {
			return nil
		}
		if strings.Contains(path, string(os.PathSeparator)+"backups"+string(os.PathSeparator)) {
			return nil
		}
		info, err := d.Info()
		if err != nil {
			return err
		}
		relPath, err := filepath.Rel(baseDir, path)
		if err != nil {
			return err
		}
		return addFile(tw, baseDir, relPath, info)
	})
}

func addFile(tw *tar.Writer, baseDir, rel string, info os.FileInfo) error {
	return addNamedFile(tw, filepath.ToSlash(rel), filepath.Join(baseDir, rel), info)
}

func addNamedFile(tw *tar.Writer, name, path string, info os.FileInfo) error {
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()
	header, err := tar.FileInfoHeader(info, "")
	if err != nil {
		return err
	}
	header.Name = name
	if err := tw.WriteHeader(header); err != nil {
		return err
	}
	_, err = io.Copy(tw, file)
	return err
}
