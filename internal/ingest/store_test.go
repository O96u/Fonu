package ingest

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/fonu/fonu/internal/db"
)

func TestStoreInsertAndState(t *testing.T) {
	dir := t.TempDir()
	conn, err := db.Open(filepath.Join(dir, "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	if err := db.Migrate(context.Background(), conn, filepath.Join("..", "..", "migrations")); err != nil {
		t.Fatal(err)
	}
	store := NewStore(conn)
	ctx := context.Background()
	now := time.Now().UTC()
	rows := []FactRow{{
		TS: now, ClientIP: "1.2.3.4", Host: "a.example.com", Service: "web",
		Method: "GET", Path: "/", Status: 200, Kind: "access", GeoPending: true,
	}}
	if err := store.InsertBatch(ctx, rows); err != nil {
		t.Fatal(err)
	}
	if err := store.SaveState(ctx, "/tmp/access.log", IngestState{Offset: 123, FileSize: 456}); err != nil {
		t.Fatal(err)
	}
	st, err := store.LoadState(ctx, "/tmp/access.log")
	if err != nil || st.Offset != 123 {
		t.Fatalf("state=%+v err=%v", st, err)
	}
}

func TestBatchAndStateRollback(t *testing.T) {
 ctx:=context.Background()
 conn,err:=db.Open(filepath.Join(t.TempDir(),"test.db"));if err!=nil{t.Fatal(err)};defer conn.Close()
 if err:=db.Migrate(ctx,conn,filepath.Join("..","..","migrations"));err!=nil{t.Fatal(err)}
 s:=NewStore(conn)
 row:=FactRow{TS:time.Now(),ClientIP:"8.8.8.8",Host:"example.com",Method:"GET",Path:"/",Status:200,Kind:"access"}
 if _,err:=conn.Exec("CREATE TRIGGER fail_ingest_state BEFORE INSERT ON ingest_state BEGIN SELECT RAISE(ABORT, 'test failure'); END");err!=nil{t.Fatal(err)}
 if err:=s.InsertBatchAndState(ctx,[]FactRow{row},"access.log",IngestState{Offset:100});err==nil{t.Fatal("expected state failure")}
 var count int
 if err:=conn.QueryRow("SELECT COUNT(*) FROM access_facts").Scan(&count);err!=nil||count!=0{t.Fatalf("partial commit: %d %v",count,err)}
 if _,err:=conn.Exec("DROP TRIGGER fail_ingest_state");err!=nil{t.Fatal(err)}
 if err:=s.InsertBatchAndState(ctx,[]FactRow{row},"access.log",IngestState{Offset:100});err!=nil{t.Fatal(err)}
 if err:=conn.QueryRow("SELECT COUNT(*) FROM access_facts").Scan(&count);err!=nil||count!=1{t.Fatalf("count: %d %v",count,err)}
 state,err:=s.LoadState(ctx,"access.log");if err!=nil||state.Offset!=100{t.Fatalf("state: %+v %v",state,err)}
}
