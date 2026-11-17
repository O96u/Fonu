// Package fnos adapts Fonu to the fnOS unified gateway.
//
// The desktop entry forwards /app/fonu to a Unix socket and injects
// X-Trim-Userid, X-Trim-Username and X-Trim-Isadmin. Those headers are only
// attached to the request context by Handler, which is used exclusively on
// that socket. The TCP admin port never calls Handler, so a client cannot
// become an administrator by sending the same headers to :6893.
package fnos

import (
	"context"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

const defaultAppName = "fonu"

// Identity is the fnOS user forwarded by the unified gateway.
type Identity struct {
	UID      string
	Username string
	IsAdmin  bool
}

type contextKey int

const (
	identityKey contextKey = iota
	prefixKey
)

// SocketPath is ${TRIM_APPDEST}/app.sock, or FONU_GATEWAY_SOCKET when set.
// An empty path means the process is not running as an fnOS app.
func SocketPath() string {
	if p := strings.TrimSpace(os.Getenv("FONU_GATEWAY_SOCKET")); p != "" {
		return p
	}
	dest := strings.TrimSpace(os.Getenv("TRIM_APPDEST"))
	if dest == "" {
		return ""
	}
	return filepath.Join(dest, "app.sock")
}

// Prefix is the public path registered in app/ui/config.
func Prefix() string {
	if p := strings.TrimSpace(os.Getenv("FONU_GATEWAY_PREFIX")); p != "" {
		if !strings.HasPrefix(p, "/") {
			p = "/" + p
		}
		return strings.TrimRight(p, "/")
	}
	name := strings.TrimSpace(os.Getenv("TRIM_APPNAME"))
	if name == "" {
		name = defaultAppName
	}
	name = strings.ReplaceAll(name, ".", "-")
	return "/app/" + name
}

// ParseIdentity reads the gateway user headers.
// X-Trim-Uid is accepted because some fnOS builds use that name.
func ParseIdentity(h http.Header) (Identity, bool) {
	uid := cleanHeader(firstHeader(h, "X-Trim-Userid", "X-Trim-Uid"))
	username := cleanHeader(h.Get("X-Trim-Username"))
	if uid == "" && username == "" {
		return Identity{}, false
	}
	admin := strings.TrimSpace(h.Get("X-Trim-Isadmin"))
	return Identity{
		UID:      uid,
		Username: username,
		IsAdmin:  strings.EqualFold(admin, "true") || admin == "1",
	}, true
}

// Handler strips the gateway prefix and records the forwarded user.
// Requests outside the prefix are rejected so the socket cannot be used as a
// second unauthenticated entry to the admin API.
func Handler(prefix string, next http.Handler) http.Handler {
	prefix = strings.TrimRight(strings.TrimSpace(prefix), "/")
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == prefix && (r.Method == http.MethodGet || r.Method == http.MethodHead) {
			http.Redirect(w, r, prefix+"/", http.StatusFound)
			return
		}
		path, ok := stripPrefix(r.URL.Path, prefix)
		if !ok {
			http.NotFound(w, r)
			return
		}
		ctx := withPrefix(r.Context(), prefix)
		if ident, ok := ParseIdentity(r.Header); ok {
			ctx = withIdentity(ctx, ident)
		}
		cloned := r.Clone(ctx)
		cloned.URL.Path = path
		cloned.URL.RawPath = ""
		next.ServeHTTP(w, cloned)
	})
}

func stripPrefix(path, prefix string) (string, bool) {
	if prefix == "" {
		if path == "" {
			return "/", true
		}
		return path, true
	}
	if path == prefix {
		return "/", true
	}
	if strings.HasPrefix(path, prefix+"/") {
		rest := strings.TrimPrefix(path, prefix)
		if rest == "" {
			return "/", true
		}
		return rest, true
	}
	return "", false
}

func withIdentity(ctx context.Context, ident Identity) context.Context {
	return context.WithValue(ctx, identityKey, ident)
}

func withPrefix(ctx context.Context, prefix string) context.Context {
	return context.WithValue(ctx, prefixKey, prefix)
}

// IdentityFromContext returns the user injected by Handler.
func IdentityFromContext(ctx context.Context) (Identity, bool) {
	ident, ok := ctx.Value(identityKey).(Identity)
	if !ok || (ident.UID == "" && ident.Username == "") {
		return Identity{}, false
	}
	return ident, true
}

// PrefixFromContext is the gateway prefix for the current request.
// Session cookies set on that request are scoped to this path.
func PrefixFromContext(ctx context.Context) string {
	prefix, _ := ctx.Value(prefixKey).(string)
	return prefix
}

func firstHeader(h http.Header, names ...string) string {
	for _, name := range names {
		if v := strings.TrimSpace(h.Get(name)); v != "" {
			return v
		}
	}
	return ""
}

func cleanHeader(v string) string {
	v = strings.TrimSpace(v)
	v = strings.Map(func(r rune) rune {
		if r < 32 || r == 127 {
			return -1
		}
		return r
	}, v)
	runes := []rune(v)
	if len(runes) > 128 {
		v = string(runes[:128])
	}
	return v
}
