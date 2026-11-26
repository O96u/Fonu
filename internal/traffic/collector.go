package traffic

import (
	"bufio"
	"context"
	"database/sql"
	"io"
	"os"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/fonu/fonu/internal/ingest"
	"github.com/fonu/fonu/internal/logstore"
	"github.com/fonu/fonu/internal/notify"
	"github.com/fonu/fonu/internal/proxy"
)

const (
	rateWindow       = 5 * time.Second
	activeWindow     = 65 * time.Second
	persistInterval  = 30 * time.Second
	sampleBucket = 250 * time.Millisecond
)

type RuleTraffic struct {
	RuleID       int64   `json:"rule_id"`
	UploadTotal  int64   `json:"upload_total"`
	DownloadTotal int64  `json:"download_total"`
	UploadRate   float64 `json:"upload_rate"`
	DownloadRate float64 `json:"download_rate"`
	Connections  int     `json:"connections"`
}

type ClientConn struct {
	IP       string `json:"ip"`
	LastSeen string `json:"last_seen"`
}

type Collector struct {
	path   string
	store  *Store
	notify *notify.Service

	mu    sync.RWMutex
	hosts map[string]*hostState
}

type hostState struct {
	persistedUpload int64
	persistedDownload int64
	uploadTotal   int64
	downloadTotal int64
	clients       map[string]time.Time
	samples       []sample
}

type sample struct {
	at   time.Time
	up   int64
	down int64
}

func NewCollector(db *sql.DB, accessLogPath string, notifySvc *notify.Service) *Collector {
	return &Collector{
		path:   accessLogPath,
		store:  NewStore(db),
		notify: notifySvc,
		hosts:  map[string]*hostState{},
	}
}

func (c *Collector) AttachIngest(ingestSvc interface{ OnLine(h ingest.LineHandler) }) {
	if ingestSvc == nil {
		return
	}
	ingestSvc.OnLine(func(p ingest.ParsedLine) {
		c.ingestEntry(p.Entry, true)
	})
}

func (c *Collector) Start(ctx context.Context) {
	if totals, err := c.store.LoadTotals(ctx); err == nil {
		c.mu.Lock()
		for host, t := range totals {
			st := c.ensureHost(host)
			st.uploadTotal = t.Upload
			st.downloadTotal = t.Download
			st.persistedUpload=t.Upload
			st.persistedDownload=t.Download
		}
		c.mu.Unlock()
	}

	go c.persistLoop(ctx)
}

func (c *Collector) tailLoop(ctx context.Context) {
	offset := c.logOffset()
	for {
		if ctx.Err() != nil {
			return
		}
		offset = c.readFrom(offset)
		time.Sleep(500 * time.Millisecond)
	}
}

func (c *Collector) logOffset() int64 {
	info, err := os.Stat(c.path)
	if err != nil {
		return 0
	}
	return info.Size()
}

func (c *Collector) readFrom(offset int64) int64 {
	file, err := os.Open(c.path)
	if err != nil {
		return offset
	}
	defer file.Close()

	info, err := file.Stat()
	if err != nil {
		return offset
	}
	if info.Size() < offset {
		offset = 0
	}
	if _, err := file.Seek(offset, 0); err != nil {
		return offset
	}

	reader := bufio.NewReader(file)
	for {
		line, err := reader.ReadString('\n')
		if len(line) > 0 {
			c.ingestLine(strings.TrimRight(line, "\r\n"), true)
		}
		if err != nil {
			if err == io.EOF {
				pos, seekErr := file.Seek(0, io.SeekCurrent)
				if seekErr == nil {
					return pos
				}
				return info.Size()
			}
			return offset
		}
	}
}

func (c *Collector) persistLoop(ctx context.Context) {
	ticker := time.NewTicker(persistInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			_ = c.persist(context.Background())
			return
		case <-ticker.C:
			_ = c.persist(ctx)
		}
	}
}

func (c *Collector) persist(ctx context.Context) error {
	c.mu.RLock()
	snapshot := make(map[string]hostTotals, len(c.hosts))
	for host, st := range c.hosts {
		if st.uploadTotal!=st.persistedUpload||st.downloadTotal!=st.persistedDownload{
			snapshot[host] = hostTotals{Upload: st.uploadTotal, Download: st.downloadTotal}
		}
	}
	c.mu.RUnlock()
	if len(snapshot)==0{return nil}
	if err:=c.store.SaveTotals(ctx,snapshot);err!=nil{return err}
	c.mu.Lock()
	for host,t:=range snapshot{
		if st:=c.hosts[host];st!=nil{st.persistedUpload=t.Upload;st.persistedDownload=t.Download}
	}
	c.mu.Unlock()
	return nil
}

func (c *Collector) ingestLine(line string, trackLive bool) {
	entry, ok := logstore.ParseAccess(line)
	if !ok || entry.Domain == "" {
		return
	}
	c.ingestEntry(entry, trackLive)
}

func (c *Collector) ingestEntry(entry logstore.AccessEntry, trackLive bool) {
	host := EndpointKey(entry.Domain, entry.ServerPort)
	up := entry.RequestLength
	down := entry.BytesSent
	if up == 0 && down == 0 && !trackLive {
		return
	}

	now := time.Now()
	c.mu.Lock()
	st := c.ensureHost(host)
	st.uploadTotal += up
	st.downloadTotal += down
	if trackLive {
		if up > 0 || down > 0 {
			bucket:=now.Truncate(sampleBucket)
			n:=len(st.samples)
			if n>0&&st.samples[n-1].at.Equal(bucket){
				st.samples[n-1].up+=up
				st.samples[n-1].down+=down
			}else{
				st.samples=append(st.samples,sample{at:bucket,up:up,down:down})
			}
		}
		if entry.ClientIP != "" && entry.ClientIP != "-" {
			st.clients[entry.ClientIP] = now
		}
		c.pruneHost(st, now)
	}
	c.mu.Unlock()
	// Notification may query settings or perform other work; never hold
	// the traffic lock while it runs.
	if trackLive&&c.notify!=nil&&entry.ClientIP!=""&&entry.ClientIP!="-"{c.notify.RecordAccessHit(context.Background(),entry.ClientIP)}
}

func (c *Collector) ensureHost(host string) *hostState {
	st, ok := c.hosts[host]
	if !ok {
		st = &hostState{clients: map[string]time.Time{}}
		c.hosts[host] = st
	}
	return st
}

func (c *Collector) pruneHost(st *hostState, now time.Time) {
	cutoff := now.Add(-rateWindow)
	i:=0
	for i<len(st.samples)&&!st.samples[i].at.Add(sampleBucket).After(cutoff){i++}
	if i>0{st.samples=st.samples[i:]}
	activeCutoff := now.Add(-activeWindow)
	for ip, seen := range st.clients {
		if seen.Before(activeCutoff) {
			delete(st.clients, ip)
		}
	}
}

func (c *Collector) aggregateEndpoints(endpoints []proxy.Endpoint, now time.Time) (uploadTotal, downloadTotal int64, uploadRate, downloadRate float64, connections int) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	var upSamples, downSamples int64
	 clientSet := map[string]time.Time{}

	for _, ep := range endpoints {
		key := EndpointKey(ep.Hostname, ep.Port)
		st, ok := c.hosts[key]
		if !ok {
			continue
		}
		uploadTotal += st.uploadTotal
		downloadTotal += st.downloadTotal
		cutoff := now.Add(-rateWindow)
		for _, s := range st.samples {
			if !s.at.Add(sampleBucket).After(cutoff) {
				continue
			}
			upSamples += s.up
			downSamples += s.down
		}
		activeCutoff := now.Add(-activeWindow)
		for ip, seen := range st.clients {
			if seen.Before(activeCutoff) {
				continue
			}
			if prev, ok := clientSet[ip]; !ok || seen.After(prev) {
				clientSet[ip] = seen
			}
		}
	}

	// A fixed rolling window avoids exaggerated rates after isolated
	// requests and keeps all high-frequency samples within that window.
	span:=rateWindow.Seconds()
	if upSamples > 0 {
		uploadRate = float64(upSamples) / span
	}
	if downSamples > 0 {
		downloadRate = float64(downSamples) / span
	}
	connections = len(clientSet)
	return uploadTotal, downloadTotal, uploadRate, downloadRate, connections
}

func (c *Collector) SnapshotForRules(rules []proxy.Rule) []RuleTraffic {
	now := time.Now()
	out := make([]RuleTraffic, 0, len(rules))
	for _, rule := range rules {
		endpoints := rule.Endpoints()
		up, down, upRate, downRate, conns := c.aggregateEndpoints(endpoints, now)
		out = append(out, RuleTraffic{
			RuleID:        rule.ID,
			UploadTotal:   up,
			DownloadTotal: down,
			UploadRate:    upRate,
			DownloadRate:  downRate,
			Connections:   conns,
		})
	}
	return out
}

func (c *Collector) ClientsForRule(rule proxy.Rule) []ClientConn {
	now := time.Now()
	activeCutoff := now.Add(-activeWindow)
	clientSet := map[string]time.Time{}

	c.mu.RLock()
	for _, ep := range rule.Endpoints() {
		key := EndpointKey(ep.Hostname, ep.Port)
		st, ok := c.hosts[key]
		if !ok {
			continue
		}
		for ip, seen := range st.clients {
			if seen.Before(activeCutoff) {
				continue
			}
			if prev, ok := clientSet[ip]; !ok || seen.After(prev) {
				clientSet[ip] = seen
			}
		}
	}
	c.mu.RUnlock()

	out := make([]ClientConn, 0, len(clientSet))
	for ip, seen := range clientSet {
		out = append(out, ClientConn{IP: ip, LastSeen: seen.UTC().Format(time.RFC3339)})
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].LastSeen > out[j].LastSeen
	})
	return out
}
