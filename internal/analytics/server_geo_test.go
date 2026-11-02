package analytics

import (
	"context"
	"errors"
	"testing"

	"github.com/fonu/fonu/internal/iplocate"
)

type stubIPs struct {
	v4, v6 string
	ok     bool
}

func (s stubIPs) PublicIPs(context.Context) (string, string, bool) {
	return s.v4, s.v6, s.ok
}

func (s stubIPs) LastKnownPublicIPs(context.Context) (string, string, bool) {
	return "", "", false
}

type stubRegions struct {
	region iplocate.Region
}

func (s stubRegions) ParseRegion(string) iplocate.Region { return s.region }

func TestServerGeoUsesStoredPublicIP(t *testing.T) {
	cache := NewServerGeo(stubIPs{v4: "1.2.3.4", ok: true}, stubRegions{region: iplocate.Region{
		Country: "中国", Province: "四川", City: "成都", Label: "四川省 · 成都市",
	}})
	cache.detect = func(context.Context) (string, string, error) {
		t.Fatal("detect should not run when DDNS already has a public IP")
		return "", "", errors.New("unused")
	}
	node := cache.Resolve(context.Background())
	if node.Province != "四川" || node.City != "成都" {
		t.Fatalf("got %+v", node)
	}
	node.City = "changed"
	again := cache.Resolve(context.Background())
	if again.City != "成都" {
		t.Fatalf("cache miss: %+v", again)
	}
}
