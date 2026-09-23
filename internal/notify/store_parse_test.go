package notify

import (
	"testing"

	"github.com/fonu/fonu/internal/settings"
)

func TestBoolFromMap(t *testing.T) {
	values := map[string]string{
		settings.KeyNotifyOnDDNSIPChange: "true",
		settings.KeyNotifyOnLoginFailure: "0",
		settings.KeyNotifyOnIPFrequentAccess: "1",
	}
	if !boolFromMap(values, settings.KeyNotifyOnDDNSIPChange) {
		t.Fatal("expected true for true")
	}
	if boolFromMap(values, settings.KeyNotifyOnLoginFailure) {
		t.Fatal("expected false for 0")
	}
	if !boolFromMap(values, settings.KeyNotifyOnIPFrequentAccess) {
		t.Fatal("expected true for 1")
	}
}

func TestMapHasNotifySavePayloadEventKeys(t *testing.T) {
	if mapHasNotifySavePayload(map[string]string{settings.KeyNotifyOnLoginFailure: "1"}) {
		return
	}
	t.Fatal("expected notify event key to trigger save payload")
}

func TestMapHasNotifySavePayloadEmpty(t *testing.T) {
	if mapHasNotifySavePayload(map[string]string{"theme": "dark"}) {
		t.Fatal("expected unrelated keys to be ignored")
	}
}
