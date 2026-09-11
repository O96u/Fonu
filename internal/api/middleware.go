package api

import (
	"context"
	"net/http"

	"github.com/fonu/fonu/internal/auth"
)

type contextKey string

const sessionContextKey contextKey = "session_id"

func SessionMiddleware(authSvc *auth.Service) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			cookie, err := r.Cookie("fonu_session")
			if err == nil && cookie.Value != "" {
				ctx := context.WithValue(r.Context(), sessionContextKey, cookie.Value)
				r = r.WithContext(ctx)
			}
			next.ServeHTTP(w, r)
		})
	}
}

func RequireAuth(authSvc *auth.Service) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			sessionID, _ := r.Context().Value(sessionContextKey).(string)
			if err := authSvc.ValidateSession(r.Context(), sessionID); err != nil {
				writeError(w, http.StatusUnauthorized, "未登录或会话已过期")
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func sessionIDFromContext(ctx context.Context) string {
	v, _ := ctx.Value(sessionContextKey).(string)
	return v
}
