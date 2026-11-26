package certexport

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPairRollbackOnSecondPublish(t *testing.T){
	src,dst:=t.TempDir(),t.TempDir()
	for _,name:=range []string{certFileName,keyFileName}{
		if err:=os.WriteFile(filepath.Join(src,name),[]byte("new-"+name),0600);err!=nil{t.Fatal(err)}
		if err:=os.WriteFile(filepath.Join(dst,name),[]byte("old-"+name),0600);err!=nil{t.Fatal(err)}
	}
	ops:=pairOps{stat:os.Lstat,remove:os.Remove,write:writeLocalStage,rename:func(from,to string)error{
		if strings.HasPrefix(filepath.Base(from),".fonu-stage-")&&filepath.Base(to)==keyFileName{return errors.New("injected failure")}
		return os.Rename(from,to)
	}}
	if err:=publishPEMPair(src,dst,filepath.Join,ops);err==nil{t.Fatal("expected failure")}
	for _,name:=range []string{certFileName,keyFileName}{
		data,err:=os.ReadFile(filepath.Join(dst,name));if err!=nil||string(data)!="old-"+name{t.Fatalf("rollback failed: %s %v",name,err)}
	}
	entries,err:=os.ReadDir(dst);if err!=nil||len(entries)!=2{t.Fatalf("temporary files left: %v %v",entries,err)}
	if err:=copyPEMPair(src,dst);err!=nil{t.Fatal(err)}
	for _,name:=range []string{certFileName,keyFileName}{
		data,err:=os.ReadFile(filepath.Join(dst,name));if err!=nil||string(data)!="new-"+name{t.Fatalf("publish failed: %s %v",name,err)}
	}
}

func TestPairMissingSourceDoesNotChangeDestination(t *testing.T){
	src,dst:=t.TempDir(),t.TempDir()
	if err:=os.WriteFile(filepath.Join(src,certFileName),[]byte("new"),0600);err!=nil{t.Fatal(err)}
	if err:=os.WriteFile(filepath.Join(dst,certFileName),[]byte("old"),0600);err!=nil{t.Fatal(err)}
	if err:=copyPEMPair(src,dst);err==nil{t.Fatal("missing key accepted")}
	data,err:=os.ReadFile(filepath.Join(dst,certFileName));if err!=nil||string(data)!="old"{t.Fatal("previous cert modified")}
}
