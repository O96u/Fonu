//go:build windows

package nginx

import "context"

func prepareStartPlatform(m *Manager, _ context.Context) {
	m.terminateMaster()
	removePIDFile(m.cfg.NginxPIDFile)
}
