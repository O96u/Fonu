//go:build windows

package nginx

import "context"

func prepareStartPlatform(m *Manager, ctx context.Context) {
	stopOrphanMasters(m.cfg.NginxDir())
	_ = m.runNginx(ctx, []string{"-c", m.cfg.NginxConfigPath(), "-s", "quit"}, "")
}
