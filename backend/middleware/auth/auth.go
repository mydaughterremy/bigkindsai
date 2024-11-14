package auth

import (
	"net/http"

	"bigkinds.or.kr/backend/service"
)

type Authenticator struct {
	AuthService *service.AuthService
}

func (a *Authenticator) AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !a.AuthService.Authenticate(*r) {
			w.WriteHeader(http.StatusUnauthorized)
			_, _ = w.Write([]byte("Unauthorized"))
			return
		}
		next.ServeHTTP(w, r)
	})
}
