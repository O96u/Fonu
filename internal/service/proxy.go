package service

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/fonu/fonu/internal/nginx"
	"github.com/fonu/fonu/internal/proxy"
)

type ProxyService struct {
	db      *sql.DB
	store   *proxy.Store
	nginx   *nginx.Manager
}

func NewProxyService(db *sql.DB, store *proxy.Store, nginxMgr *nginx.Manager) *ProxyService {
	return &ProxyService{db: db, store: store, nginx: nginxMgr}
}

func (s *ProxyService) List(ctx context.Context) ([]proxy.Rule, error) {
	return s.store.List(ctx)
}

func (s *ProxyService) Get(ctx context.Context, id int64) (proxy.Rule, error) {
	return s.store.Get(ctx, id)
}

func (s *ProxyService) Create(ctx context.Context, in proxy.CreateInput) (proxy.Rule, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return proxy.Rule{}, err
	}
	defer func() { _ = tx.Rollback() }()

	store := proxy.NewStoreWithTx(tx)
	rule, err := store.Create(ctx, in)
	if err != nil {
		return proxy.Rule{}, err
	}

	rules, err := store.ListEnabled(ctx)
	if err != nil {
		return proxy.Rule{}, err
	}
	if _, err := s.nginx.Apply(ctx, rules); err != nil {
		return proxy.Rule{}, err
	}
	if err := tx.Commit(); err != nil {
		return proxy.Rule{}, err
	}
	return rule, nil
}

func (s *ProxyService) Update(ctx context.Context, id int64, in proxy.UpdateInput) (proxy.Rule, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return proxy.Rule{}, err
	}
	defer func() { _ = tx.Rollback() }()

	store := proxy.NewStoreWithTx(tx)
	rule, err := store.Update(ctx, id, in)
	if err != nil {
		return proxy.Rule{}, err
	}

	rules, err := store.ListEnabled(ctx)
	if err != nil {
		return proxy.Rule{}, err
	}
	if _, err := s.nginx.Apply(ctx, rules); err != nil {
		return proxy.Rule{}, err
	}
	if err := tx.Commit(); err != nil {
		return proxy.Rule{}, err
	}
	return rule, nil
}

func (s *ProxyService) Delete(ctx context.Context, id int64) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	store := proxy.NewStoreWithTx(tx)
	if err := store.Delete(ctx, id); err != nil {
		return err
	}

	rules, err := store.ListEnabled(ctx)
	if err != nil {
		return err
	}
	if _, err := s.nginx.Apply(ctx, rules); err != nil {
		return err
	}
	return tx.Commit()
}

func (s *ProxyService) ReloadAll(ctx context.Context) error {
	rules, err := s.store.ListEnabled(ctx)
	if err != nil {
		return err
	}
	_, err = s.nginx.Apply(ctx, rules)
	return err
}

func (s *ProxyService) ValidateRule(ctx context.Context, candidate proxy.Rule, existing []proxy.Rule) error {
	rules := make([]proxy.Rule, 0, len(existing)+1)
	replaced := false
	for _, r := range existing {
		if r.ID == candidate.ID {
			if candidate.Enabled {
				rules = append(rules, candidate)
			}
			replaced = true
			continue
		}
		if r.Enabled {
			rules = append(rules, r)
		}
	}
	if !replaced && candidate.Enabled {
		rules = append(rules, candidate)
	}
	if err := s.nginx.ValidateOnly(ctx, rules); err != nil {
		return fmt.Errorf("%w", err)
	}
	return nil
}
