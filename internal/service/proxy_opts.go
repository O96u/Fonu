package service

import (
	"context"

	"github.com/fonu/fonu/internal/chinacidr"
	"github.com/fonu/fonu/internal/nginx"
	"github.com/fonu/fonu/internal/settings"
)

func (s *ProxyService) loadGenerateOptions(ctx context.Context) nginx.GenerateOptions {
	opts := nginx.GenerateOptions{}
	if s.settings == nil {
		return opts
	}
	raw, err := s.settings.Get(ctx, settings.KeyTrustedProxy)
	if err == nil {
		opts.TrustedProxy = nginx.ParseTrustedProxyJSON(raw)
	}
	blRaw, err := s.settings.Get(ctx, settings.KeyGlobalIPBlacklist)
	if err == nil {
		opts.GlobalIPBlacklist = chinacidr.ParseGlobalIPBlacklist(blRaw)
	}
	return opts
}
