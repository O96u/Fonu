package logstore

import (
	"io"
	"os"
	"path/filepath"

	"gopkg.in/natefinch/lumberjack.v2"
)

func NewAppLogWriter(logsDir string) io.Writer {
	if err := os.MkdirAll(logsDir, 0o755); err != nil {
		return os.Stdout
	}
	return &lumberjack.Logger{
		Filename:   filepath.Join(logsDir, "app.log"),
		MaxSize:    20,
		MaxBackups: 10,
		MaxAge:     30,
		Compress:   true,
	}
}
