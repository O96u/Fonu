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

func TestPurgeRetentionChunksPreserveBoundary(t *testing.T){
 conn,err:=db.Open(filepath.Join(t.TempDir(),"retention.db"));if err!=nil{t.Fatal(err)};defer conn.Close()
 ctx:=context.Background()
 if err:=db.Migrate(ctx,conn,filepath.Join("..","..","migrations"));err!=nil{t.Fatal(err)}
 _,err=conn.Exec("WITH RECURSIVE n(i) AS (SELECT 1 UNION ALL SELECT i+1 FROM n WHERE i<4001) INSERT INTO access_facts(ts,client_ip,bytes_sent) SELECT '2026-01-01T00:00:00Z','8.8.8.8',1 FROM n")
 if err!=nil{t.Fatal(err)}
 _,err=conn.Exec("INSERT INTO access_facts(ts,client_ip,bytes_sent) VALUES('2026-01-02T00:00:00Z','8.8.8.8',1)")
 if err!=nil{t.Fatal(err)}
 cutoff:=time.Date(2026,1,2,0,0,0,0,time.UTC)
 n,err:=NewStore(conn).PurgeBefore(ctx,cutoff);if err!=nil||n!=4001{t.Fatalf("purge count=%d err=%v",n,err)}
 var remaining int
 if err:=conn.QueryRow("SELECT count(*) FROM access_facts").Scan(&remaining);err!=nil||remaining!=1{t.Fatalf("remaining=%d err=%v",remaining,err)}
}
