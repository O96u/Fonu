package analytics

import "testing"

func TestClassifyUA(t *testing.T) {
	if classifyUA("Mozilla/5.0 (Linux; Android 13) Chrome/120.0") != "Android" {
		t.Fatal("android")
	}
	if classifyUA("Mozilla/5.0 (iPhone; CPU iPhone OS 17_0 like Mac OS X)") != "iOS" {
		t.Fatal("ios")
	}
	if classifyUA("Mozilla/5.0 Chrome/120.0.0.0 Safari/537.36") != "Chrome" {
		t.Fatal("chrome")
	}
}
