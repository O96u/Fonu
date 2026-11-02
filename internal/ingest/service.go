package ingest

import (
	"bufio"
	"context"
	"io"
	"log/slog"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/fonu/fonu/internal/awareness"
	"github.com/fonu/fonu/internal/iplocate"
	"github.com/fonu/fonu/internal/logstore"
	"github.com/fonu/fonu/internal/notify"
	"github.com/fonu/fonu/internal/proxy"
	"github.com/fonu/fonu/internal/settings"
)

const (
	batchSize       = 100
	stateSaveEvery  = 2 * time.Second
	backfillSleep   = 5 * time.Millisecond
	geoTickInterval = 15 * time.Second
	retentionTick   = 6 * time.Hour
)

type LineHandler func(ParsedLine)

type Service struct {
	path      string
	store     *Store
	settings  *settings.Store
	geo       *iplocate.Service
	notify    *notify.Service
	listRules func(ctx context.Context) ([]proxy.Rule, error)
	logger    *slog.Logger

	mu           sync.RWMutex
	bindings     proxy.BindingIndex
	handlers     []LineHandler
	realtime     *RealtimeRing
	backfillDone chan struct{}
}

func NewService(
	store *Store,
	logPath string,
	settingsStore *settings.Store,
	geo *iplocate.Service,
	notifySvc *notify.Service,
	listRules func(ctx context.Context) ([]proxy.Rule, error),
	logger *slog.Logger,
) *Service {
	if logger == nil {
		logger = slog.Default()
	}
	return &Service{
		path:         logPath,
		store:        store,
		settings:     settingsStore,
		geo:          geo,
		notify:       notifySvc,
		listRules:    listRules,
		logger:       logger.With("module", "INGEST"),
		realtime:     NewRealtimeRing(60 * time.Second),
		backfillDone: make(chan struct{}),
	}
}

func (s *Service) Realtime() *RealtimeRing {
	return s.realtime
}

func (s *Service) OnLine(h LineHandler) {
	s.mu.Lock()
	s.handlers = append(s.handlers, h)
	s.mu.Unlock()
}

func (s *Service) BackfillActive() bool {
	select {
	case <-s.backfillDone:
		return false
	default:
		return true
	}
}

func (s *Service) Start(ctx context.Context) {
	s.refreshBindings(ctx)
	go s.bindingLoop(ctx)
	go func() {
		s.runBackfill(ctx)
		close(s.backfillDone)
	}()
	go s.tailLoop(ctx)
	go s.geoLoop(ctx)
	go s.retentionLoop(ctx)
}

func (s *Service) bindingLoop(ctx context.Context) {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.refreshBindings(ctx)
		}
	}
}

func (s *Service) refreshBindings(ctx context.Context) {
	if s.listRules == nil {
		return
	}
	rules, err := s.listRules(ctx)
	if err != nil {
		return
	}
	s.mu.Lock()
	s.bindings = proxy.BuildBindingIndex(rules)
	s.mu.Unlock()
}

func (s *Service) tailLoop(ctx context.Context) {
	select {
	case <-s.backfillDone:
	case <-ctx.Done():
		return
	}
	st, _ := s.store.LoadState(ctx, s.path)
	offset := st.Offset
	lastSave := time.Now()

	for {
		if ctx.Err() != nil {
			return
		}
		offset, batch := s.readFrom(offset)
		if len(batch) > 0 {
			if err := s.store.InsertBatch(ctx, batch); err != nil {
				s.logger.Warn("insert batch failed", "error", err.Error())
			} else {
				s.dispatch(batch)
			}
		}
		if time.Since(lastSave) >= stateSaveEvery {
			_ = s.persistOffset(ctx, offset)
			lastSave = time.Now()
		}
		time.Sleep(500 * time.Millisecond)
	}
}

func (s *Service) persistOffset(ctx context.Context, offset int64) error {
	info, err := os.Stat(s.path)
	if err != nil {
		return err
	}
	return s.store.SaveState(ctx, s.path, IngestState{
		Offset:   offset,
		FileSize: info.Size(),
		MtimeNs:  info.ModTime().UnixNano(),
	})
}

func (s *Service) readFrom(offset int64) (int64, []FactRow) {
	file, err := os.Open(s.path)
	if err != nil {
		return offset, nil
	}
	defer file.Close()

	info, err := file.Stat()
	if err != nil {
		return offset, nil
	}
	if info.Size() < offset {
		offset = 0
	}
	if _, err := file.Seek(offset, 0); err != nil {
		return offset, nil
	}

	s.mu.RLock()
	bindings := s.bindings
	s.mu.RUnlock()

	var batch []FactRow
	reader := bufio.NewReader(file)
	for {
		line, err := reader.ReadString('\n')
		if len(line) > 0 {
			row, ok := s.parseLine(strings.TrimRight(strings.TrimRight(line, "\n"), "\r"), bindings)
			if ok {
				batch = append(batch, row)
				s.realtime.Add(row)
			}
			if len(batch) >= batchSize {
				pos, seekErr := file.Seek(0, io.SeekCurrent)
				if seekErr == nil {
					return pos, batch
				}
				return info.Size(), batch
			}
		}
		if err != nil {
			if err == io.EOF {
				pos, seekErr := file.Seek(0, io.SeekCurrent)
				if seekErr == nil {
					return pos, batch
				}
				return info.Size(), batch
			}
			return offset, batch
		}
	}
}

func (s *Service) dispatch(rows []FactRow) {
	for _, row := range rows {
		entry := logstore.AccessEntry{
			Time:         row.TS.UTC().Format(time.RFC3339),
			Domain:       row.Host,
			Method:       row.Method,
			Path:         row.Path,
			Status:       row.Status,
			ResponseTime: row.RequestMS / 1000,
			ClientIP:     row.ClientIP,
			BytesSent:    row.BytesSent,
			UserAgent:    row.UserAgent,
		}
		p := ParsedLine{Entry: entry, At: row.TS}
		s.mu.RLock()
		handlers := append([]LineHandler(nil), s.handlers...)
		s.mu.RUnlock()
		for _, h := range handlers {
			if h != nil {
				h(p)
			}
		}
	}
}

func (s *Service) parseLine(line string, bindings proxy.BindingIndex) (FactRow, bool) {
	entry, ok := logstore.ParseAccess(line)
	if !ok || !logstore.IsDashboardAccess(entry) {
		return FactRow{}, false
	}
	at, okTime := logstore.ParseAccessTime(entry.Time)
	if !okTime {
		at = time.Now()
	}
	service := proxy.ResolveServiceLabel(entry.Domain, entry.ServerPort, entry.Upstream, bindings)
	kind := string(awareness.ClassifyStatic(entry.Status, entry.Path, entry.UserAgent))
	geo := s.resolveGeo(entry.ClientIP)
	reqMS := entry.ResponseTime * 1000
	return FactRow{
		TS:          at,
		ClientIP:    entry.ClientIP,
		Host:        entry.Domain,
		Service:     service,
		Method:      entry.Method,
		Path:        entry.Path,
		Status:      entry.Status,
		BytesSent:   entry.BytesSent,
		RequestMS:   reqMS,
		UserAgent:   entry.UserAgent,
		Kind:        kind,
		GeoCountry:  geo.GeoCountry,
		GeoProvince: geo.GeoProvince,
		GeoCity:     geo.GeoCity,
		GeoLabel:    geo.GeoLabel,
		GeoPending:  geo.GeoPending,
	}, true
}

func (s *Service) resolveGeo(ip string) FactRow {
	ip = strings.TrimSpace(ip)
	if ip == "" || ip == "—" {
		return FactRow{GeoPending: false}
	}
	ctx := context.Background()
	if cached, ok, _ := s.store.LoadGeoCache(ctx, ip); ok && cached.GeoLabel != "" {
		return FactRow{
			GeoCountry:  cached.GeoCountry,
			GeoProvince: cached.GeoProvince,
			GeoCity:     cached.GeoCity,
			GeoLabel:    cached.GeoLabel,
			GeoPending:  false,
		}
	}
	if s.geo != nil {
		reg := s.geo.ParseRegion(ip)
		if reg.Label != "" {
			row := FactRow{
				GeoCountry:  reg.Country,
				GeoProvince: reg.Province,
				GeoCity:     reg.City,
				GeoLabel:    reg.Label,
				GeoPending:  false,
			}
			_ = s.store.SaveGeoCache(ctx, ip, row)
			return row
		}
	}
	if !iplocate.IsPublicIP(ip) {
		if s.geo != nil {
			reg := s.geo.ParseRegion(ip)
			return FactRow{
				GeoCountry: reg.Country,
				GeoLabel:   reg.Label,
				GeoPending: false,
			}
		}
		return FactRow{GeoPending: false}
	}
	return FactRow{GeoPending: true}
}

func (s *Service) geoLoop(ctx context.Context) {
	ticker := time.NewTicker(geoTickInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.runGeoEnrich(ctx)
		}
	}
}

func (s *Service) runGeoEnrich(ctx context.Context) {
	if s.geo == nil {
		return
	}
	ips, err := s.store.PendingIPs(ctx, 100)
	if err != nil || len(ips) == 0 {
		return
	}
	remoteEnabled, _ := s.settings.GetBool(ctx, settings.KeyAnalyticsGeoRemoteEnabled)
	remoteURL, _ := s.settings.Get(ctx, settings.KeyAnalyticsGeoRemoteURL)

	var needRemote []string
	for _, ip := range ips {
		reg := s.geo.ParseRegion(ip)
		if reg.Label != "" {
			row := FactRow{
				GeoCountry: reg.Country, GeoProvince: reg.Province,
				GeoCity: reg.City, GeoLabel: reg.Label,
			}
			_ = s.store.SaveGeoCache(ctx, ip, row)
			_ = s.store.ApplyGeo(ctx, ip, row)
			continue
		}
		if remoteEnabled {
			needRemote = append(needRemote, ip)
		}
	}
	if remoteEnabled && len(needRemote) > 0 {
		batchCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
		remote := s.geo.RemoteEnrichBatch(batchCtx, remoteURL, needRemote)
		cancel()
		for ip, reg := range remote {
			if reg.Label == "" {
				continue
			}
			row := FactRow{
				GeoCountry: reg.Country, GeoProvince: reg.Province,
				GeoCity: reg.City, GeoLabel: reg.Label,
			}
			_ = s.store.SaveGeoCache(ctx, ip, row)
			_ = s.store.ApplyGeo(ctx, ip, row)
		}
	}
}

func (s *Service) retentionLoop(ctx context.Context) {
	ticker := time.NewTicker(retentionTick)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.purgeRetention(ctx)
		}
	}
}

func (s *Service) purgeRetention(ctx context.Context) {
	days, err := s.settings.GetInt(ctx, settings.KeyAnalyticsRetentionDays)
	if err != nil || days <= 0 {
		days = 30
	}
	cutoff := time.Now().UTC().AddDate(0, 0, -days)
	if n, err := s.store.PurgeBefore(ctx, cutoff); err == nil && n > 0 {
		s.logger.Info("purged access analytics rows", "count", n, "before", cutoff.Format(time.RFC3339))
	}
}

func (s *Service) runBackfill(ctx context.Context) {
	st, err := s.store.LoadState(ctx, s.path)
	if err != nil {
		s.seekEOF(ctx)
		return
	}
	if st.Offset > 0 {
		return
	}
	skip, _ := s.settings.GetBool(ctx, settings.KeyAnalyticsSkipHistoryBackfill)
	if skip {
		s.seekEOF(ctx)
		return
	}

	file, err := os.Open(s.path)
	if err != nil {
		s.seekEOF(ctx)
		return
	}
	defer file.Close()

	s.mu.RLock()
	bindings := s.bindings
	s.mu.RUnlock()

	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	var batch []FactRow
	for scanner.Scan() {
		if ctx.Err() != nil {
			return
		}
		row, ok := s.parseLine(scanner.Text(), bindings)
		if !ok {
			continue
		}
		batch = append(batch, row)
		if len(batch) >= batchSize {
			if err := s.store.InsertBatch(ctx, batch); err != nil {
				s.logger.Warn("backfill batch failed", "error", err.Error())
			}
			batch = batch[:0]
			time.Sleep(backfillSleep)
		}
	}
	if len(batch) > 0 {
		_ = s.store.InsertBatch(ctx, batch)
	}
	s.seekEOF(ctx)
	s.logger.Info("access analytics backfill finished")
}

func (s *Service) seekEOF(ctx context.Context) {
	info, err := os.Stat(s.path)
	if err != nil {
		return
	}
	_ = s.store.SaveState(ctx, s.path, IngestState{
		Offset:   info.Size(),
		FileSize: info.Size(),
		MtimeNs:  info.ModTime().UnixNano(),
	})
}
