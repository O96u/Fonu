package certexport

const (
	certFileName = "fullchain.pem"
	keyFileName  = "privatekey.pem"
)

// CertMeta is passed to the on-change shell hook.
type CertMeta struct {
	Name        string
	StorageName string
	Deleted     bool
	CertDir     string
	CertFile    string
	KeyFile     string
}
