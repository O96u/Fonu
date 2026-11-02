package awareness

import "testing"

func TestClassifyProbePath(t *testing.T) {
	st := &ipWindowState{}
	kind, _ := classifyEntry(404, "/.env", "", st)
	if kind != KindScanSuspected {
		t.Fatalf("kind=%s", kind)
	}
}

func TestClassifyBlocked403(t *testing.T) {
	st := &ipWindowState{}
	kind, _ := classifyEntry(403, "/", "", st)
	if kind != KindBlocked {
		t.Fatalf("kind=%s", kind)
	}
	if st.blocked != 1 {
		t.Fatalf("blocked=%d", st.blocked)
	}
}
