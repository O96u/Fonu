package iplocate

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/fonu/fonu/internal/config"
	"github.com/lionsoul2014/ip2region/binding/golang/xdb"
)

const (
	// Upstream renamed legacy ip2region.xdb → ip2region_v4.xdb (see ip2region/data).
	defaultV4XDBURL = "https://raw.githubusercontent.com/lionsoul2014/ip2region/master/data/ip2region_v4.xdb"
	ghfastProxyPrefix = "https://ghfast.top/"
	minV4XDBBytes     = 1024 * 1024
)

var defaultV4XDBURLs = []string{
	defaultV4XDBURL,
	ghfastProxyPrefix + defaultV4XDBURL,
}

// Service resolves client IPs with a local ip2region xdb (offline).
type Service struct {
	cfg    config.Config
	client *http.Client

	mu        sync.RWMutex
	searcher  *xdb.Searcher
	ready     bool
	lastErr   string
	ensureMu  sync.Mutex
	ensureTry sync.Once
}

func New(cfg config.Config) *Service {
	return &Service{
		cfg: cfg,
		client: &http.Client{
			Timeout: 5 * time.Minute,
		},
	}
}

func (s *Service) dbPath() string {
	return filepath.Join(s.cfg.DataDir, "geo", "ip2region_v4.xdb")
}

// DBPath is the on-disk ip2region v4 database path.
func (s *Service) DBPath() string {
	return s.dbPath()
}

// Ensure downloads the database if missing and opens the searcher.
func (s *Service) Ensure(ctx context.Context) error {
	s.ensureMu.Lock()
	defer s.ensureMu.Unlock()

	path := s.dbPath()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	if fi, err := os.Stat(path); err != nil || fi.Size() < minV4XDBBytes {
		if err == nil {
			_ = os.Remove(path)
		}
		var lastErr error
		for _, url := range defaultV4XDBURLs {
			if ctx.Err() != nil {
				return ctx.Err()
			}
			if err := s.download(ctx, url, path); err != nil {
				lastErr = err
				continue
			}
			lastErr = nil
			break
		}
		if lastErr != nil {
			s.mu.Lock()
			s.lastErr = lastErr.Error()
			s.mu.Unlock()
			return fmt.Errorf("download ip2region v4: %w", lastErr)
		}
	}
	if err := s.open(path); err != nil {
		s.mu.Lock()
		s.lastErr = err.Error()
		s.mu.Unlock()
		return err
	}
	s.mu.Lock()
	s.lastErr = ""
	s.mu.Unlock()
	return nil
}

func (s *Service) download(ctx context.Context, url, dest string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	resp, err := s.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("%s: HTTP %d", url, resp.StatusCode)
	}
	tmp := dest + ".part"
	f, err := os.Create(tmp)
	if err != nil {
		return err
	}
	n, err := io.Copy(f, resp.Body)
	if err != nil {
		f.Close()
		_ = os.Remove(tmp)
		return err
	}
	if err := f.Close(); err != nil {
		_ = os.Remove(tmp)
		return err
	}
	if n < minV4XDBBytes {
		_ = os.Remove(tmp)
		return fmt.Errorf("%s: file too small (%d bytes)", url, n)
	}
	return os.Rename(tmp, dest)
}

func (s *Service) open(path string) error {
	header, err := xdb.LoadHeaderFromFile(path)
	if err != nil {
		return err
	}
	ver, err := xdb.VersionFromHeader(header)
	if err != nil {
		return err
	}
	searcher, err := xdb.NewWithFileOnly(ver, path)
	if err != nil {
		return err
	}
	s.mu.Lock()
	if s.searcher != nil {
		s.searcher.Close()
	}
	s.searcher = searcher
	s.ready = true
	s.mu.Unlock()
	return nil
}

// Ready reports whether lookups may succeed.
func (s *Service) Ready() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.ready
}

func (s *Service) kickEnsure() {
	s.ensureTry.Do(func() {
		go func() {
			ctx, cancel := context.WithTimeout(context.Background(), 6*time.Minute)
			defer cancel()
			_ = s.Ensure(ctx)
		}()
	})
}

// Lookup returns a display label for ip (empty if unknown or DB not ready).
func (s *Service) Lookup(ip string) string {
	ip = strings.TrimSpace(ip)
	if ip == "" || ip == "—" {
		return ""
	}
	if label, ok := LabelForPrivate(ip); ok {
		return label
	}
	s.mu.RLock()
	searcher := s.searcher
	s.mu.RUnlock()
	if searcher == nil {
		s.kickEnsure()
		return ""
	}
	raw, err := searcher.Search(ip)
	if err != nil {
		return ""
	}
	return FormatRegion(raw)
}

// LookupCached deduplicates lookups for batch enrichment.
func (s *Service) LookupCached(ip string, cache map[string]string) string {
	ip = strings.TrimSpace(ip)
	if v, ok := cache[ip]; ok {
		return v
	}
	v := s.Lookup(ip)
	cache[ip] = v
	return v
}
