//go:build !windows

package nginx

import "context"

func prepareStartPlatform(m *Manager, ctx context.Context) {}
