package ingest

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/fonu/fonu/internal/db"
)

func testAccessLine(at time.Time, n int) string {
	return fmt.Sprintf("%s example.com GET /p%d 200 0.001 1.2.3.4 10.0.0.1:80 10 20\n", at.UTC().Format(time.RFC3339), n)
}

func TestReadFromDoesNotSkipBufferedLines(t *testing.T) {
	dir := t.TempDir()
	conn, err := db.Open(filepath.Join(dir, "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	if err := db.Migrate(context.Background(), conn, filepath.Join("..", "..", "migrations")); err != nil {
		t.Fatal(err)
	}
	logPath := filepath.Join(dir, "access.log")
	const n = 250
	base := time.Date(2026, 11, 2, 1, 0, 0, 0, time.UTC)
	var buf []byte
	for i := 0; i < n; i++ {
		buf = append(buf, testAccessLine(base.Add(time.Duration(i)*time.Second), i)...)
	}
	if err := os.WriteFile(logPath, buf, 0o644); err != nil {
		t.Fatal(err)
	}
	svc := NewService(NewStore(conn), logPath, nil, nil, nil, nil, slog.Default())
	var offset int64
	got := 0
	for i := 0; i < 20; i++ {
		next, batch := svc.readFrom(offset)
		got += len(batch)
		if next <= offset {
			break
		}
		offset = next
	}
	if got != n {
		t.Fatalf("parsed %d lines, want %d (offset %d)", got, n, offset)
	}
}

func TestTailLoopReadsEachLineOnce(t *testing.T) {
	dir := t.TempDir()
	conn, err := db.Open(filepath.Join(dir, "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	ctx := context.Background()
	if err := db.Migrate(ctx, conn, filepath.Join("..", "..", "migrations")); err != nil {
		t.Fatal(err)
	}
	logPath := filepath.Join(dir, "access.log")
	base := time.Date(2026, 11, 2, 3, 0, 0, 0, time.UTC)
	var buf []byte
	var mid int64
	const total = 40
	const already = 10
	for i := 0; i < total; i++ {
		if i == already {
			mid = int64(len(buf))
		}
		buf = append(buf, testAccessLine(base.Add(time.Duration(i)*time.Second), i)...)
	}
	if err := os.WriteFile(logPath, buf, 0o644); err != nil {
		t.Fatal(err)
	}
	store := NewStore(conn)
	if err := store.SaveState(ctx, logPath, IngestState{Offset: mid, FileSize: int64(len(buf))}); err != nil {
		t.Fatal(err)
	}
	svc := NewService(store, logPath, nil, nil, nil, nil, slog.Default())
	runCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	svc.Start(runCtx)

	want := total - already
	deadline := time.Now().Add(3 * time.Second)
	var n int
	for time.Now().Before(deadline) {
		if err := conn.QueryRow(`SELECT COUNT(*) FROM access_facts`).Scan(&n); err != nil {
			t.Fatal(err)
		}
		if n > want {
			t.Fatalf("tail reread the same lines, rows=%d want %d", n, want)
		}
		if n == want {
			break
		}
		time.Sleep(50 * time.Millisecond)
	}
	if n != want {
		t.Fatalf("rows=%d want %d", n, want)
	}
	time.Sleep(1200 * time.Millisecond)
	if err := conn.QueryRow(`SELECT COUNT(*) FROM access_facts`).Scan(&n); err != nil {
		t.Fatal(err)
	}
	if n != want {
		t.Fatalf("tail kept inserting, rows=%d want %d", n, want)
	}
}

func TestRewindSkippedTail(t *testing.T) {
	dir := t.TempDir()
	conn, err := db.Open(filepath.Join(dir, "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	ctx := context.Background()
	if err := db.Migrate(ctx, conn, filepath.Join("..", "..", "migrations")); err != nil {
		t.Fatal(err)
	}
	logPath := filepath.Join(dir, "access.log")
	base := time.Date(2026, 11, 2, 2, 0, 0, 0, time.UTC)
	var buf []byte
	var firstLen int
	for i := 0; i < 3; i++ {
		line := testAccessLine(base.Add(time.Duration(i)*time.Minute), i)
		if i == 0 {
			firstLen = len(line)
		}
		buf = append(buf, line...)
	}
	if err := os.WriteFile(logPath, buf, 0o644); err != nil {
		t.Fatal(err)
	}
	store := NewStore(conn)
	if err := store.InsertBatch(ctx, []FactRow{{
		TS: base, ClientIP: "1.2.3.4", Host: "example.com", Method: "GET", Path: "/p0",
		Status: 200, Kind: "access",
	}}); err != nil {
		t.Fatal(err)
	}
	if err := store.SaveState(ctx, logPath, IngestState{Offset: int64(len(buf)), FileSize: int64(len(buf))}); err != nil {
		t.Fatal(err)
	}
	svc := NewService(store, logPath, nil, nil, nil, nil, slog.Default())
	svc.rewindSkippedTail(ctx)
	st, err := store.LoadState(ctx, logPath)
	if err != nil {
		t.Fatal(err)
	}
	if st.Offset != int64(firstLen) {
		t.Fatalf("offset=%d want %d", st.Offset, firstLen)
	}
}
