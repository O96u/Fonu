package settings

import (
	"context"
	"strings"
	"time"
)

func (s *Store) Location(ctx context.Context) *time.Location {
	if s == nil {
		return time.Local
	}
	name, err := s.Get(ctx, KeyTimezone)
	if err != nil || strings.TrimSpace(name) == "" {
		name = Defaults[KeyTimezone]
	}
	loc, err := time.LoadLocation(strings.TrimSpace(name))
	if err != nil {
		return time.Local
	}
	return loc
}

func (s *Store) Now(ctx context.Context) time.Time {
	return time.Now().In(s.Location(ctx))
}
