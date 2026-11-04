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
	"time"

	"github.com/fonu/fonu/internal/db"
)

const pendingDirName = "restore-pending"

type Service struct {
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
	tmpDir := filepath.Join(s.dataDir, "backups")
	if err := os.MkdirAll(tmpDir, 0o755); err != nil {
		return "", err
	}
	name := fmt.Sprintf("fonu-backup-%s.tar.gz", time.Now().Format("20060102-His"))
	outPath := filepath.Join(tmpDir, name)
	file, err := os.Create(outPath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	gw := gzip.NewWriter(file)
	tw := tar.NewWriter(gw)

	entries := []string{
		"fonu.db",
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
	return outPath, nil
}

func (s *Service) Restore(ctx context.Context, archivePath string) error {
	file, err := os.Open(archivePath)
	if err != nil {
		return err
	}
	defer file.Close()

	pending := filepath.Join(s.dataDir, pendingDirName)
	if err := os.RemoveAll(pending); err != nil {
		return err
	}
	if err := os.MkdirAll(pending, 0o755); err != nil {
		return err
	}

	gr, err := gzip.NewReader(file)
	if err != nil {
		_ = os.RemoveAll(pending)
		return fmt.Errorf("无效的备份文件")
	}
	defer gr.Close()

	tr := tar.NewReader(gr)
	wroteDB := false
	for {
		if err := ctx.Err(); err != nil {
			_ = os.RemoveAll(pending)
			return err
		}
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			_ = os.RemoveAll(pending)
			return err
		}
		if header.Typeflag != tar.TypeReg && header.Typeflag != tar.TypeDir {
			continue
		}
		if !allowedEntry(header.Name) {
			continue
		}
		target, err := safeJoin(pending, header.Name)
		if err != nil {
			_ = os.RemoveAll(pending)
			return err
		}
		switch header.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(target, 0o755); err != nil {
				_ = os.RemoveAll(pending)
				return err
			}
		case tar.TypeReg:
			if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
				_ = os.RemoveAll(pending)
				return err
			}
			out, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
			if err != nil {
				_ = os.RemoveAll(pending)
				return err
			}
			if _, err := io.Copy(out, tr); err != nil {
				out.Close()
				_ = os.RemoveAll(pending)
				return err
			}
			if err := out.Close(); err != nil {
				_ = os.RemoveAll(pending)
				return err
			}
			if filepath.ToSlash(header.Name) == "fonu.db" {
				wroteDB = true
			}
		}
	}
	if !wroteDB {
		_ = os.RemoveAll(pending)
		return fmt.Errorf("备份里没有数据库")
	}
	// 进程还占着正在使用的库。先放到旁边校验，下次启动、打开数据库之前再换上去。
	if err := db.PrepareRestore(filepath.Join(pending, "fonu.db")); err != nil {
		_ = os.RemoveAll(pending)
		return fmt.Errorf("备份里的数据库不可用")
	}
	return nil
}

func allowedEntry(name string) bool {
	name = strings.TrimPrefix(filepath.ToSlash(strings.TrimSpace(name)), "./")
	switch {
	case name == "fonu.db":
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
	cleaned := filepath.Clean(filepath.FromSlash(name))
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

	live := filepath.Join(dataDir, "fonu.db")
	bak := live + ".before-restore"
	_ = os.Remove(bak)
	if _, err := os.Stat(live); err == nil {
		if err := os.Rename(live, bak); err != nil {
			return err
		}
	}
	for _, suffix := range []string{"-wal", "-shm"} {
		side := live + suffix
		if _, err := os.Stat(side); err == nil {
			_ = os.Remove(bak + suffix)
			if err := os.Rename(side, bak+suffix); err != nil {
				_ = os.Rename(bak, live)
				return err
			}
		}
	}
	if err := os.Rename(staged, live); err != nil {
		_ = os.Rename(bak, live)
		for _, suffix := range []string{"-wal", "-shm"} {
			_ = os.Rename(bak+suffix, live+suffix)
		}
		return err
	}

	for _, dir := range []string{"certs", "nginx"} {
		src := filepath.Join(pending, dir)
		if _, err := os.Stat(src); err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return err
		}
		dst := filepath.Join(dataDir, dir)
		dstBak := dst + ".before-restore"
		_ = os.RemoveAll(dstBak)
		if _, err := os.Stat(dst); err == nil {
			if err := os.Rename(dst, dstBak); err != nil {
				return err
			}
		}
		if err := os.Rename(src, dst); err != nil {
			_ = os.Rename(dstBak, dst)
			return err
		}
		_ = os.RemoveAll(dstBak)
	}

	_ = os.Remove(bak)
	_ = os.Remove(bak + "-wal")
	_ = os.Remove(bak + "-shm")
	return os.RemoveAll(pending)
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
	tmp, err := os.CreateTemp("", "fonu-export-*.db")
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()
	tmp.Close()
	if err := os.Remove(tmpPath); err != nil {
		return err
	}
	defer os.Remove(tmpPath)
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
		if d.IsDir() {
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
