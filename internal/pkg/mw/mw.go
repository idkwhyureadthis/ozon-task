package mw

import (
	"context"
	"net/http"
)

func AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		userHeader := r.Header.Get("user")
		ctx := r.Context()
		if userHeader != "" {
			ctx = context.WithValue(ctx, "user", userHeader)
		}
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
