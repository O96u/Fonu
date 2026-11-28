package ddns

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
)

type aliACMETestTransport func(*http.Request) (*http.Response, error)

func (f aliACMETestTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func TestAliDNSACMEAddTXTRequestTTL(t *testing.T) {
	added := 0
	a := &AliDNS{client: &http.Client{Transport: aliACMETestTransport(func(req *http.Request) (*http.Response, error) {
		q := req.URL.Query()
		body := "{}"
		switch q.Get("Action") {
		case "DescribeDomains":
			body = `{"Domains":{"Domain":[{"DomainName":"example.com"}]}}`
		case "DescribeDomainRecords":
			body = `{"TotalCount":0,"DomainRecords":{"Record":[]}}`
		case "AddDomainRecord":
			added++
			if q.Get("TTL") != "600" || q.Get("DomainName") != "example.com" || q.Get("RR") != "_acme-challenge.nas" || q.Get("Type") != "TXT" || q.Get("Value") != "challenge-token" {
				t.Errorf("unexpected AddDomainRecord parameters: %v", q)
			}
		default:
			t.Errorf("unexpected action: %s", q.Get("Action"))
		}
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(body)), Header: make(http.Header)}, nil
	})}}
	if err := a.UpsertTXTRecord(context.Background(), Credentials{Token: "test", Secret: "test"}, "_acme-challenge.nas.example.com", "challenge-token"); err != nil {
		t.Fatal(err)
	}
	if added != 1 {
		t.Fatalf("expected one AddDomainRecord, got %d", added)
	}
}
