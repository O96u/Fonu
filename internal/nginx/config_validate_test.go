package nginx

import "testing"

func TestAssertValidSSLBlocksRejectsListenSSLWithoutCertificate(t *testing.T) {
	content := `http {
    server {
        listen 8017 ssl;
        listen [::]:8017 ssl;
        server_name app.example.com;
        location / {
            proxy_pass http://127.0.0.1:1;
        }
    }
}
`
	if err := assertValidSSLBlocks(content); err == nil {
		t.Fatal("expected invalid ssl block to fail validation")
	}
}

func TestAssertValidSSLBlocksAcceptsCertificateAfterLocation(t *testing.T) {
	content := `http {
    server {
        listen 8017 ssl;
        server_name app.example.com;
        location / {
            proxy_pass http://127.0.0.1:1;
        }
        ssl_certificate /data/certs/wildcard.example.com/fullchain.pem;
        ssl_certificate_key /data/certs/wildcard.example.com/privatekey.pem;
    }
}
`
	if err := assertValidSSLBlocks(content); err != nil {
		t.Fatalf("expected config with certificate directives to pass: %v", err)
	}
}
