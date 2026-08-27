package transport

import (
	"context"
	"net/http"
	"strings"
)

// contextKey is unexported so no other package can collide with our context keys.
type contextKey string

const userIDKey contextKey = "userID"

// UserIDFromContext returns the authenticated user id that requireAuth stored.
func UserIDFromContext(ctx context.Context) (string, bool) {
	id, ok := ctx.Value(userIDKey).(string)
	return id, ok
}

// requireAuth wraps a handler: it verifies the Bearer token (via the auth
// service) and, on success, puts the caller's user id in the request context.
// On any failure it responds 401 and the wrapped handler never runs.
func (s *Server) requireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token, ok := strings.CutPrefix(r.Header.Get("Authorization"), "Bearer ")
		if !ok || token == "" {
			http.Error(w, "missing or malformed Authorization header", http.StatusUnauthorized)
			return
		}

		userID, err := s.auth.Verify(r.Context(), token)
		if err != nil {
			http.Error(w, "invalid or expired token", http.StatusUnauthorized)
			return
		}

		ctx := context.WithValue(r.Context(), userIDKey, userID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
