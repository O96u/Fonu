package logstore

import (
	"bytes"
	"io"
	"os"
	"time"
)

const tailReadChunk = 64 * 1024

// TailLines returns up to max most recent non-empty lines without scanning the whole file.
func TailLines(path string, max int) ([]string, error) {
	return tailLines(path, max)
}

// tailLines returns up to max most recent non-empty lines without scanning the whole file.
func tailLines(path string, max int) ([]string, error) {
	if max <= 0 {
		return nil, nil
	}
	file, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	defer file.Close()

	info, err := file.Stat()
	if err != nil {
		return nil, err
	}
	size := info.Size()
	if size == 0 {
		return nil, nil
	}

	var pending []byte
	var lines []string
	offset := size

	for offset > 0 && len(lines) < max {
		readSize := int64(tailReadChunk)
		if readSize > offset {
			readSize = offset
		}
		offset -= readSize
		chunk := make([]byte, readSize)
		if _, err := file.ReadAt(chunk, offset); err != nil && err != io.EOF {
			return nil, err
		}
		buf := append(chunk, pending...)
		pending = nil
		for len(buf) > 0 && len(lines) < max {
			i := bytes.LastIndexByte(buf, '\n')
			if i < 0 {
				pending = buf
				break
			}
			line := stringsTrimCR(string(buf[i+1:]))
			buf = buf[:i]
			if line != "" {
				lines = append(lines, line)
			}
		}
	}
	if len(pending) > 0 && len(lines) < max {
		line := stringsTrimCR(string(pending))
		if line != "" {
			lines = append(lines, line)
		}
	}

	for i, j := 0, len(lines)-1; i < j; i, j = i+1, j-1 {
		lines[i], lines[j] = lines[j], lines[i]
	}
	return lines, nil
}

// ReadAccessLinesSince returns fonu dashboard access lines with time >= since, chronological order.
// Scans from the end of the file and stops once older lines are seen.
func ReadAccessLinesSince(path string, since time.Time, maxLines int) ([]string, error) {
	return readAccessLinesSince(path, since, maxLines)
}

func readAccessLinesSince(path string, since time.Time, maxLines int) ([]string, error) {
	file, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	defer file.Close()

	info, err := file.Stat()
	if err != nil {
		return nil, err
	}
	size := info.Size()
	if size == 0 {
		return nil, nil
	}

	var pending []byte
	var newestFirst []string
	offset := size
	since = since.In(time.Local)

	for offset > 0 {
		readSize := int64(tailReadChunk)
		if readSize > offset {
			readSize = offset
		}
		offset -= readSize
		chunk := make([]byte, readSize)
		if _, err := file.ReadAt(chunk, offset); err != nil && err != io.EOF {
			return nil, err
		}
		buf := append(chunk, pending...)
		pending = nil
		stop := false
		for len(buf) > 0 {
			i := bytes.LastIndexByte(buf, '\n')
			if i < 0 {
				pending = buf
				break
			}
			line := stringsTrimCR(string(buf[i+1:]))
			buf = buf[:i]
			if line == "" {
				continue
			}
			entry, ok := parseAccess(line)
			if !ok || !IsDashboardAccess(entry) {
				continue
			}
			at, ok := parseAccessTime(entry.Time)
			if !ok {
				newestFirst = append(newestFirst, line)
				continue
			}
			at = at.In(since.Location())
			if at.Before(since) {
				stop = true
				break
			}
			newestFirst = append(newestFirst, line)
			if maxLines > 0 && len(newestFirst) >= maxLines {
				stop = true
				break
			}
		}
		if stop {
			break
		}
	}

	out := make([]string, len(newestFirst))
	for i := range newestFirst {
		out[i] = newestFirst[len(newestFirst)-1-i]
	}
	return out, nil
}

func stringsTrimCR(s string) string {
	if len(s) > 0 && s[len(s)-1] == '\r' {
		return s[:len(s)-1]
	}
	return s
}
