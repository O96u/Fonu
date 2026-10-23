package certificate

import "testing"

func TestSSHExportValidate(t *testing.T) {
	cfg := SSHExportConfig{
		Enabled:  true,
		Host:     "10.0.0.1",
		Username: "root",
		Password: "secret",
		RemoteDir: "/etc/certs",
		AuthType: SSHAuthPassword,
	}.Normalize()
	if err := cfg.Validate(); err != nil {
		t.Fatal(err)
	}

	cfg.Password = ""
	cfg.HasPassword = true
	if err := cfg.Validate(); err != nil {
		t.Fatal(err)
	}

	cfg.RemoteDir = "relative"
	if err := cfg.Validate(); err == nil {
		t.Fatal("expected remote dir error")
	}
}

func TestMergeSSHSecrets(t *testing.T) {
	old := SSHExportConfig{Password: "keep", PrivateKey: "key"}
	newSSH := SSHExportConfig{}
	merged := mergeSSHSecrets(newSSH, old)
	if merged.Password != "keep" || merged.PrivateKey != "key" {
		t.Fatalf("merge failed %+v", merged)
	}
}
