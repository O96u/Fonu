package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/fonu/fonu/internal/proxy"
	"github.com/fonu/fonu/internal/notify"
)

// mutateProxy runs while applyMu is held. Load non-transaction stores first:
// SQLite may use a single connection. Never query them while tx owns it.
func mutateProxy[T any](ctx context.Context, s *ProxyService, mutate func(*proxy.Store) (T, error)) (T, error) {
	var zero T
	certs, err := s.loadCertSources(ctx)
	if err != nil { return zero, err }
	opts := s.loadGenerateOptions(ctx)
	oldRules, err := s.store.ListEnabled(ctx)
	if err != nil { return zero, err }
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil { return zero, err }
	defer tx.Rollback()
	store := proxy.NewStoreWithTx(tx)
	value, err := mutate(store)
	if err != nil { return zero, err }
	rules, err := store.ListEnabled(ctx)
	if err != nil { return zero, err }
	if _, err := s.nginx.ApplyWithOptions(ctx, rules, certs, opts); err != nil {
		_ = tx.Rollback()
		if s.notify != nil {
			s.notify.Alert(ctx, notify.EventNginxReloadFailure, "Nginx 重载失败", err.Error())
		}
		return zero, fmt.Errorf("配置发布失败，数据库变更已取消：%w", err)
	}
	if err := tx.Commit(); err != nil {
		_ = tx.Rollback()
		// The HTTP request may already be canceled; use an independent bounded context.
		recovery, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		defer cancel()
		_, rollbackErr := s.nginx.ApplyWithOptions(recovery, oldRules, certs, opts)
		if rollbackErr != nil {
			return zero, errors.Join(err, fmt.Errorf("恢复旧配置失败：%w", rollbackErr))
		}
		return zero, fmt.Errorf("数据库提交失败，已请求恢复旧配置：%w", err)
	}
	s.nginx.SetGenerateOptions(opts)
	return value, nil
}
