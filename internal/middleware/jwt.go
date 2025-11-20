package middleware

import (
	"context"
	"net/http"
	"os"
	"strings"

	"github.com/vaughan-dsouza/BeGo/internal/utils"
)

func AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		auth := r.Header.Get("Authorization")
		if auth == "" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		parts := strings.SplitN(auth, " ", 2)
		if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		token := strings.TrimSpace(parts[1])
		if token == "" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		claims, err := utils.VerifyToken(token, os.Getenv("ACCESS_SECRET"))
		if err != nil {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		// push user ID into context
		ctx := context.WithValue(r.Context(), utils.CtxUserIDKey, claims.SubjectInt())

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
