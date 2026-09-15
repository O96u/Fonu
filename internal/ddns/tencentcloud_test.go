package ddns

import "testing"

func TestTencentSubdomain(t *testing.T) {
	if tencentSubdomain("@") != "@" {
		t.Fatal("expected @")
	}
	if tencentSubdomain("www") != "www" {
		t.Fatal("expected www")
	}
}

func TestCredentialsTencentCloud(t *testing.T) {
	cred := CredentialsFromSave(SaveInput{
		Provider:   "tencentcloud",
		APIToken:   "AKIDxxx",
		APISecret:  "secret",
	})
	if cred.Provider != "tencentcloud" || !cred.HasValues() {
		t.Fatal("expected tencentcloud credentials")
	}
	if err := cred.Validate("tencentcloud"); err != nil {
		t.Fatal(err)
	}
}
