package notify
import("fmt";"testing";"time";"github.com/fonu/fonu/internal/logstore")
func TestAttackThresholdsAndCooldown(t *testing.T){
 now:=time.Now();var state attackState
 entry:=logstore.AccessEntry{Domain:"nas.example.com",ServerPort:443,ClientIP:"8.8.8.8",Path:"/.env",Status:403}
 for i:=0;i<4;i++{if got:=state.observe(entry,now,true,true);len(got)!=0{t.Fatal("early scan")}}
 if got:=state.observe(entry,now,true,true);len(got)!=1||got[0].event!=EventScan{t.Fatalf("scan: %+v",got)}
 if got:=state.observe(entry,now,true,true);len(got)!=0{t.Fatal("cooldown failed")}
 state=attackState{};entry.Path="/";entry.Status=200;entry.ClientIP="192.168.1.2"
 for i:=0;i<700;i++{if got:=state.observe(entry,now,true,true);len(got)!=0{t.Fatal("private network alerted")}}
 for i:=0;i<599;i++{entry.ClientIP=fmt.Sprintf("8.8.8.%d",i%10+1);if got:=state.observe(entry,now,true,true);len(got)!=0{t.Fatal("early cc")}}
 if got:=state.observe(entry,now,true,true);len(got)!=1||got[0].event!=EventCC{t.Fatalf("cc: %+v",got)}
 state=attackState{};entry.Status=429
 for i:=0;i<99;i++{if got:=state.observe(entry,now,false,true);len(got)!=0{t.Fatal("early limit")}}
 if got:=state.observe(entry,now,false,true);len(got)!=1||got[0].event!=EventCC{t.Fatalf("limited: %+v",got)}
 if got:=state.observe(entry,now.Add(time.Minute),false,true);len(got)!=0{t.Fatal("window not reset")}
}
