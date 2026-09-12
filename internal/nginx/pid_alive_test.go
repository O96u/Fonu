package nginx

import (
	"os"
	"testing"
)

func TestIsPIDAliveCurrentProcess(t *testing.T) {
	pid := os.Getpid()
	if !isPIDAlive(pid) {
		t.Fatalf("expected current pid %d to be alive", pid)
	}
}
