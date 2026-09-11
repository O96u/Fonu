package ddns

import "context"

type Provider interface {
	Name() string
	Verify(ctx context.Context, cred Credentials) error
	UpdateRecord(ctx context.Context, cred Credentials, rootDomain, recordName, recordType, ip string) error
	GetRecordIP(ctx context.Context, cred Credentials, rootDomain, recordName, recordType string) (string, error)
}
