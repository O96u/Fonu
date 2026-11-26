package ddns

import(
 "context"
 "errors"
 "testing"
)

func TestBusyDDNSRejectsConcurrentMutations(t *testing.T){
 s:=&Service{}
 s.updateMu.Lock()
 defer s.updateMu.Unlock()
 if _,err:=s.Update(context.Background(),1,SaveInput{});!errors.Is(err,ErrUpdateBusy){t.Fatalf("update: %v",err)}
 if err:=s.Delete(context.Background(),1);!errors.Is(err,ErrUpdateBusy){t.Fatalf("delete: %v",err)}
 if err:=s.runUpdate(context.Background(),Config{});!errors.Is(err,ErrUpdateBusy){t.Fatalf("run: %v",err)}
}
