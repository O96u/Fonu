package ddns

import "testing"

func TestParseCredentialsLegacyToken(t *testing.T) {
	cred, err := ParseCredentials("cf-token")
	if err != nil {
		t.Fatal(err)
	}
	if cred.Provider != "cloudflare" || cred.Token != "cf-token" {
		t.Fatalf("unexpected cred: %+v", cred)
	}
}

func TestCredentialsLoginToken(t *testing.T) {
	cred := Credentials{TokenID: "123", Token: "abc"}
	if cred.LoginToken() != "123,abc" {
		t.Fatal("login token mismatch")
	}
}

func TestFqdnForRecord(t *testing.T) {
	if fqdnForRecord("example.com", "*") != "*.example.com" {
		t.Fatal("wildcard fqdn mismatch")
	}
	if fqdnForRecord("example.com", "@") != "example.com" {
		t.Fatal("apex fqdn mismatch")
	}
}
