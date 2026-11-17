package app

import (
	"errors"
	"net"
	"net/http"
	"os"
	"time"

	"github.com/fonu/fonu/internal/fnos"
)

func (a *App) serveGateway(handler http.Handler) {
	socketPath := fnos.SocketPath()
	if socketPath == "" {
		return
	}
	if gatewaySocketBusy(socketPath) {
		a.logger.Error("fnos gateway socket is already in use", "module", "FNOS", "path", socketPath)
		return
	}
	if err := os.Remove(socketPath); err != nil && !errors.Is(err, os.ErrNotExist) {
		a.logger.Error("remove stale fnos gateway socket", "module", "FNOS", "path", socketPath, "error", err.Error())
		return
	}
	ln, err := net.Listen("unix", socketPath)
	if err != nil {
		a.logger.Error("fnos gateway listen failed", "module", "FNOS", "path", socketPath, "error", err.Error())
		return
	}
	if err := os.Chmod(socketPath, 0o666); err != nil {
		_ = ln.Close()
		_ = os.Remove(socketPath)
		a.logger.Error("fnos gateway socket permissions failed", "module", "FNOS", "path", socketPath, "error", err.Error())
		return
	}

	prefix := fnos.Prefix()
	server := &http.Server{
		Handler:           fnos.Handler(prefix, handler),
		ReadHeaderTimeout: a.server.ReadHeaderTimeout,
	}
	a.gateway = server
	a.gatewaySocket = socketPath
	go func() {
		err := server.Serve(ln)
		if err != nil && !errors.Is(err, http.ErrServerClosed) && !errors.Is(err, net.ErrClosed) {
			a.logger.Error("fnos gateway stopped", "module", "FNOS", "error", err.Error())
		}
	}()
	a.logger.Info("fnos gateway listening", "module", "FNOS", "socket", socketPath, "prefix", prefix)
}

func gatewaySocketBusy(path string) bool {
	conn, err := net.DialTimeout("unix", path, 200*time.Millisecond)
	if err != nil {
		return false
	}
	_ = conn.Close()
	return true
}
