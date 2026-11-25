//go:build !windows

package nginx

import "context"

// Never use pkill or terminate unrelated Nginx instances on the NAS.
func prepareStartPlatform(m *Manager, _ context.Context) {
	m.terminateMaster()
	removePIDFile(m.cfg.NginxPIDFile)
}
