package service

import (
 "context"
 "path/filepath"
 "testing"
 "github.com/fonu/fonu/internal/db"
)
func TestProxyBusinessMetadata(t *testing.T){
 ctx:=context.Background()
 conn,err:=db.Open(filepath.Join(t.TempDir(),"test.db"));if err!=nil{t.Fatal(err)};defer conn.Close()
 if err=db.Migrate(ctx,conn,filepath.Join("..","..","migrations"));err!=nil{t.Fatal(err)}
 // No nginx manager: metadata operations must never publish.
 s:=&ProxyService{db:conn}
 if _,err=conn.ExecContext(ctx,"INSERT INTO proxy_rules(domain,upstream) VALUES('a.example.com','http://127.0.0.1:8080')");err!=nil{t.Fatal(err)}
 state,err:=s.ChangeProxyBusiness(ctx,ProxyBusinessChange{Action:"create",Name:"影视"});if err!=nil||len(state.Groups)!=1{t.Fatalf("%+v %v",state,err)}
 id:=state.Groups[0].ID
 if _,err=s.ChangeProxyBusiness(ctx,ProxyBusinessChange{Action:"assign",ID:id,RuleIDs:[]int64{1,999}});err==nil{t.Fatal("invalid batch accepted")}
 state,err=s.ProxyBusiness(ctx);if err!=nil||len(state.Members)!=0{t.Fatal("failed batch persisted")}
 state,err=s.ChangeProxyBusiness(ctx,ProxyBusinessChange{Action:"assign",ID:id,RuleIDs:[]int64{1}});if err!=nil||len(state.Members)!=1{t.Fatal(err)}
 state,err=s.ChangeProxyBusiness(ctx,ProxyBusinessChange{Action:"delete",ID:id});if err!=nil||len(state.Members)!=0{t.Fatal(err)}
 var count int;if err=conn.QueryRowContext(ctx,"SELECT count(*) FROM proxy_rules").Scan(&count);err!=nil||count!=1{t.Fatal("deleting group deleted rule")}
}
