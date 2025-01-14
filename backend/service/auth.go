package service

import (
	"net/http"
	"os"
)

type AuthService struct {
}

func NewAuthService() *AuthService {
	return &AuthService{}
}

func (s *AuthService) Authenticate(r http.Request) bool {
	return r.Header.Get("Authorization") == "Bearer "+os.Getenv("UPSTAGE_KINDSAI_KEY")
}
