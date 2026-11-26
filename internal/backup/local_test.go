package backup
import("context";"os";"path/filepath";"testing";"time";"github.com/fonu/fonu/internal/db")
func TestLocalBackupLifecycle(t *testing.T){
 ctx:=context.Background();dir:=t.TempDir()
 conn,err:=db.Open(filepath.Join(dir,"fonu.db"));if err!=nil{t.Fatal(err)}
 if _,err:=conn.Exec("CREATE TABLE testdata(id INTEGER); INSERT INTO testdata VALUES(7)");err!=nil{t.Fatal(err)};defer conn.Close()
 s:=New(dir);row,err:=s.CreateLocal(ctx);if err!=nil{t.Fatal(err)}
 if !localName.MatchString(row.Name)||row.Kind!="manual"{t.Fatalf("invalid backup metadata: %+v",row)}
 f,err:=s.OpenLocal(row.Name);if err!=nil{t.Fatal(err)};f.Close()
 if _,err:=s.OpenLocal("../fonu.db");err==nil{t.Fatal("traversal accepted")}
 if err:=s.AutoBackup(ctx,1);err!=nil{t.Fatal(err)}
 rows,err:=s.ListLocal();if err!=nil||len(rows)!=2{t.Fatalf("rows=%v err=%v",rows,err)}
 if err:=s.AutoBackup(ctx,1);err!=nil{t.Fatal(err)}
 for _,r:=range rows{if r.Kind=="auto"{p:=filepath.Join(dir,"backups","local",r.Name);old:=time.Now().Add(-25*time.Hour);if err:=os.Chtimes(p,old,old);err!=nil{t.Fatal(err)}}}
 if err:=s.AutoBackup(ctx,1);err!=nil{t.Fatal(err)}
 rows,err=s.ListLocal();if err!=nil||len(rows)!=2{t.Fatalf("retention rows=%v err=%v",rows,err)}
 if err:=s.RestoreLocal(ctx,row.Name);err!=nil{t.Fatal(err)}
 if err:=s.DeleteLocal(row.Name);err!=nil{t.Fatal(err)}
 if _,err:=s.OpenLocal(row.Name);err==nil{t.Fatal("deleted backup remained")}
}
