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
)

type Service struct {
	dataDir string
}

func New(dataDir string) *Service {
	return &Service{dataDir: dataDir}
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
		if err := addToArchive(ctx, tw, s.dataDir, entry); err != nil {
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

	gr, err := gzip.NewReader(file)
	if err != nil {
		return fmt.Errorf("无效的备份文件")
	}
	defer gr.Close()

	tr := tar.NewReader(gr)
	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		if header.Typeflag != tar.TypeReg && header.Typeflag != tar.TypeDir {
			continue
		}
		target := filepath.Join(s.dataDir, header.Name)
		if !strings.HasPrefix(filepath.Clean(target), filepath.Clean(s.dataDir)) {
			return fmt.Errorf("备份条目非法")
		}
		switch header.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(target, 0o755); err != nil {
				return err
			}
		case tar.TypeReg:
			if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
				return err
			}
			out, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, os.FileMode(header.Mode))
			if err != nil {
				return err
			}
			if _, err := io.Copy(out, tr); err != nil {
				out.Close()
				return err
			}
			out.Close()
		}
	}
	return nil
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
	path := filepath.Join(baseDir, rel)
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()
	header, err := tar.FileInfoHeader(info, "")
	if err != nil {
		return err
	}
	header.Name = filepath.ToSlash(rel)
	if err := tw.WriteHeader(header); err != nil {
		return err
	}
	_, err = io.Copy(tw, file)
	return err
}
