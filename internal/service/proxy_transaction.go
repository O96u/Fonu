package service

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/fonu/fonu/internal/proxy"
	"github.com/fonu/fonu/internal/nginx"
	"github.com/fonu/fonu/internal/notify"
)

// mutateProxy runs while applyMu is held. Load non-transaction stores first:
// SQLite may use a single connection. Never query them while tx owns it.
func mutateProxy[T any](ctx context.Context, s *ProxyService, mutate func(*proxy.Store) (T, error)) (T, error) {
	return mutateProxyWithFiles(ctx, s, func(store *proxy.Store, _ *sql.Tx) (T, error) {
		return mutate(store)
	}, nil, nil, nil)
}

// Candidate options may override custom files without touching live files.
// persist runs only after a successful reload request; restore runs before compensation.
func mutateProxyWithFiles[T any](ctx context.Context, s *ProxyService,
	mutate func(*proxy.Store, *sql.Tx) (T, error),
	override func(*nginx.GenerateOptions), persist func() error, restore func() error,
) (T, error) {
	var zero T
	certs, err := s.loadCertSources(ctx)
	if err != nil { return zero, err }
	oldOpts := s.loadGenerateOptions(ctx)
	opts := oldOpts
	if override != nil { override(&opts) }
	oldRules, err := s.store.ListEnabled(ctx)
	if err != nil { return zero, err }
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil { return zero, err }
	defer tx.Rollback()
	store := proxy.NewStoreWithTx(tx)
	value, err := mutate(store, tx)
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
	var commitErr error
	if persist != nil { commitErr = persist() }
	if commitErr == nil { commitErr = tx.Commit() }
	if commitErr != nil {
		_ = tx.Rollback()
		if restore != nil {
			if err := restore(); err != nil { return zero, errors.Join(commitErr, fmt.Errorf("恢复自定义文件失败：%w", err)) }
		}
		// The HTTP request may already be canceled; use an independent bounded context.
		recovery, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		defer cancel()
		_, rollbackErr := s.nginx.ApplyWithOptions(recovery, oldRules, certs, oldOpts)
		if rollbackErr != nil {
			return zero, errors.Join(commitErr, fmt.Errorf("恢复旧配置失败：%w", rollbackErr))
		}
		return zero, fmt.Errorf("配置持久化失败，已请求恢复旧配置：%w", commitErr)
	}
	s.nginx.SetGenerateOptions(s.loadGenerateOptions(ctx))
	return value, nil
}
