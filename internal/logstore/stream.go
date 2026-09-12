package logstore

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"strings"
	"time"
)

type LineFilter func(line string) bool

func StreamFile(ctx context.Context, path string, w io.Writer, flush func() error, tail int, filter LineFilter) error {
	file, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			fmt.Fprintf(w, "event: info\ndata: 日志文件尚未生成\n\n")
			_ = flush()
			return nil
		}
		return err
	}
	defer file.Close()

	if tail > 0 {
		lines, err := tailLines(path, tail)
		if err != nil {
			return err
		}
		for _, line := range lines {
			if strings.TrimSpace(line) == "" {
				continue
			}
			if filter != nil && !filter(line) {
				continue
			}
			fmt.Fprintf(w, "event: log\ndata: %s\n\n", trimSSE(line+"\n"))
			if err := flush(); err != nil {
				return err
			}
		}
	}

	if _, err := file.Seek(0, io.SeekEnd); err != nil {
		return err
	}
	reader := bufio.NewReader(file)
	for {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		line, err := reader.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				time.Sleep(500 * time.Millisecond)
				continue
			}
			return err
		}
		if filter != nil && !filter(strings.TrimRight(line, "\r\n")) {
			continue
		}
		fmt.Fprintf(w, "event: log\ndata: %s\n\n", trimSSE(line))
		if err := flush(); err != nil {
			return err
		}
	}
}

func trimSSE(line string) string {
	line = line[:len(line)-1]
	if len(line) > 0 && line[len(line)-1] == '\r' {
		line = line[:len(line)-1]
	}
	return line
}
