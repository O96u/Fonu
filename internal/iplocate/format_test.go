package iplocate

import "testing"

func TestFormatRegionChina(t *testing.T) {
	got := FormatRegion("中国|0|广东省|深圳市|电信")
	want := "广东省 · 深圳市 · 电信"
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

func TestFormatRegionBeijingISPInCitySlot(t *testing.T) {
	got := FormatRegion("中国|0|北京市|电信|CN")
	want := "北京市 · 电信"
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

func TestFormatRegionForeign(t *testing.T) {
	got := FormatRegion("美国|0|0|0|0")
	if got != "美国" {
		t.Fatalf("got %q", got)
	}
}

func TestFormatRegionUnitedStatesAmazon(t *testing.T) {
	got := FormatRegion("United States|0|California|San Jose|Amazon")
	want := "美国"
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
	got = FormatRegion("United States|0|0|San Jose|Amazon")
	if got != "美国" {
		t.Fatalf("short got %q want 美国", got)
	}
}

func TestLabelForPrivate(t *testing.T) {
	if label, ok := LabelForPrivate("192.168.1.1"); !ok || label != "内网" {
		t.Fatalf("got %q ok=%v", label, ok)
	}
}
