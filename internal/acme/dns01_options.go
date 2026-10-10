package acme

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	"github.com/fonu/fonu/internal/settings"
	"github.com/go-acme/lego/v4/challenge/dns01"
)

type dns01Options struct {
	PropagationTimeout time.Duration
	RecursiveNS        []string
	DisableAuthNS      bool
	SkipPropagation    bool
}

func (s *Service) loadDNS01Options(ctx context.Context) dns01Options {
	opts := dns01Options{
		PropagationTimeout: 180 * time.Second,
		RecursiveNS: []string{
			"223.5.5.5:53",
			"119.29.29.29:53",
			"114.114.114.114:53",
			"1.1.1.1:53",
			"8.8.8.8:53",
		},
	}
	if s == nil || s.settings == nil {
		return opts
	}
	if v, err := s.settings.GetInt(ctx, settings.KeyACMEDNSPropagationTimeout); err == nil && v > 0 {
		opts.PropagationTimeout = time.Duration(v) * time.Second
	}
	if v, err := s.settings.GetBool(ctx, settings.KeyACMEDNSDisableAuthNS); err == nil && v {
		opts.DisableAuthNS = true
	}
	if v, err := s.settings.GetBool(ctx, settings.KeyACMEDNSIgnorePropagation); err == nil && v {
		opts.SkipPropagation = true
	}
	if raw, _ := s.settings.Get(ctx, settings.KeyACMEDNSRecursiveNS); strings.TrimSpace(raw) != "" {
		var list []string
		if json.Unmarshal([]byte(raw), &list) == nil && len(list) > 0 {
			opts.RecursiveNS = list
		}
	}
	return opts
}

func applyDNS01Options(opts dns01Options) []dns01.ChallengeOption {
	out := []dns01.ChallengeOption{dns01.AddRecursiveNameservers(opts.RecursiveNS)}
	if opts.DisableAuthNS {
		out = append(out, dns01.DisableAuthoritativeNssPropagationRequirement())
	}
	if opts.SkipPropagation {
		out = append(out, dns01.PropagationWait(0, true))
	}
	return out
}
