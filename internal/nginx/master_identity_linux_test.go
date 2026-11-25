//go:build linux

package nginx

import (
	"os"
	"path/filepath"
	"testing"
	"github.com/fonu/fonu/internal/config"
)

func TestManagerRejectsUnrelatedLivePID(t *testing.T){
	cfg:=config.Config{DataDir:t.TempDir(),NginxPIDFile:filepath.Join(t.TempDir(),"nginx.pid"),NginxBin:"/proc/self/exe"}
	m:=&Manager{cfg:cfg}
	if m.ownsMaster(os.Getpid()){t.Fatal("non-nginx process accepted as master")}
}
