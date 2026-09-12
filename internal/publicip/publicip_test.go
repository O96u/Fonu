package publicip

import "testing"

func TestIsPublicIPv4(t *testing.T) {
	if IsPublicIPv4("192.168.1.1") {
		t.Fatal("private ip should be rejected")
	}
	if !IsPublicIPv4("8.8.8.8") {
		t.Fatal("public ip should be accepted")
	}
}
