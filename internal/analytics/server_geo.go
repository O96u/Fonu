package analytics

import (
	"context"
	"sync"
	"time"

	"github.com/fonu/fonu/internal/iplocate"
	"github.com/fonu/fonu/internal/publicip"
)

// ServerNode is where this Fonu instance sits, from its public IP.
type ServerNode struct {
	Country  string `json:"country,omitempty"`
	Province string `json:"province,omitempty"`
	City     string `json:"city,omitempty"`
	Label    string `json:"label,omitempty"`
}

type publicIPLookup interface {
	PublicIPs(ctx context.Context) (ipv4, ipv6 string, ok bool)
	LastKnownPublicIPs(ctx context.Context) (ipv4, ipv6 string, ok bool)
}

type regionLookup interface {
	ParseRegion(ip string) iplocate.Region
}

// ServerGeoCache resolves and caches this machine's geo for the access map.
type ServerGeoCache struct {
	ips     publicIPLookup
	regions regionLookup
	detect  func(context.Context) (ipv4, ipv6 string, err error)

	mu       sync.Mutex
	until    time.Time
	node     ServerNode
	valid    bool
	inflight chan struct{}
}

func NewServerGeo(ips publicIPLookup, regions regionLookup) *ServerGeoCache {
	return &ServerGeoCache{
		ips:     ips,
		regions: regions,
		detect:  publicip.Detect,
	}
}

// Warm resolves the public IP in the background so the first homepage request does not wait on it.
func (c *ServerGeoCache) Warm() {
	if c == nil {
		return
	}
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		c.Resolve(ctx)
	}()
}

func (c *ServerGeoCache) Resolve(ctx context.Context) ServerNode {
	if c == nil {
		return ServerNode{}
	}
	c.mu.Lock()
	if c.valid && time.Now().Before(c.until) {
		node := c.node
		c.mu.Unlock()
		return node
	}
	if c.inflight != nil {
		wait := c.inflight
		c.mu.Unlock()
		select {
		case <-wait:
		case <-ctx.Done():
			return ServerNode{}
		}
		c.mu.Lock()
		node := c.node
		c.mu.Unlock()
		return node
	}
	done := make(chan struct{})
	c.inflight = done
	c.mu.Unlock()

	node := c.lookup(ctx)

	c.mu.Lock()
	c.node = node
	c.valid = true
	if node.Country != "" || node.Province != "" {
		c.until = time.Now().Add(5 * time.Minute)
	} else {
		c.until = time.Now().Add(30 * time.Second)
	}
	c.inflight = nil
	close(done)
	c.mu.Unlock()
	return node
}

func (c *ServerGeoCache) lookup(ctx context.Context) ServerNode {
	ip := ""
	if c.ips != nil {
		if v4, v6, ok := c.ips.PublicIPs(ctx); ok {
			ip = pickPublicIP(v4, v6)
		}
		if ip == "" {
			if v4, v6, ok := c.ips.LastKnownPublicIPs(ctx); ok {
				ip = pickPublicIP(v4, v6)
			}
		}
	}
	if ip == "" && c.detect != nil {
		detectCtx, cancel := context.WithTimeout(ctx, 1500*time.Millisecond)
		defer cancel()
		v4, v6, err := c.detect(detectCtx)
		if err == nil {
			ip = pickPublicIP(v4, v6)
		}
	}
	if ip == "" || c.regions == nil {
		return ServerNode{}
	}
	region := c.regions.ParseRegion(ip)
	if region.Country == "" && region.Province == "" && region.Label == "" {
		return ServerNode{}
	}
	return ServerNode{
		Country:  region.Country,
		Province: region.Province,
		City:     region.City,
		Label:    region.Label,
	}
}

func pickPublicIP(v4, v6 string) string {
	if publicip.IsPublicIPv4(v4) {
		return v4
	}
	if publicip.IsPublicIPv6(v6) {
		return v6
	}
	return ""
}
