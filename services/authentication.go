package services

import (
	"context"
	"encoding/base64"
	"net/http"
	"strings"

	"github.com/kacperf531/sockchat/api"
)

type SockchatAuthService struct {
	UserProfiles api.SockchatProfileStore
}

type authWrapper struct {
	Username string
	Password string
}

func (s *SockchatAuthService) AuthenticateFromBasicToken(ctx context.Context, token string) (bool, error) {
	auth, err := decodeToken(token)
	if err != nil {
		return false, err
	}
	return s.UserProfiles.IsAuthValid(ctx, auth.Username, auth.Password), nil
}

func decodeToken(token string) (*authWrapper, error) {
	encoded, foundBasic := strings.CutPrefix(token, "Basic ")
	if !foundBasic {
		return nil, api.ErrBasicTokenRequired
	}
	data, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return nil, api.ErrCouldNotDecodeToken
	}
	credentials := strings.SplitN(string(data), ":", 2)
	if len(credentials) != 2 {
		return nil, api.ErrBasicTokenRequired
	}
	return &authWrapper{credentials[0], credentials[1]}, nil
}

func newAuthMiddleware(s *SockchatAuthService) func(next http.HandlerFunc) http.HandlerFunc {
	return func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			token, err := tokenFromHeader(r.Header)
			if err != nil {
				writeJsonHttpResponse(w, http.StatusUnauthorized, api.ErrorResponse{ErrorDescription: err.Error()})
			}
			ctx, cancel := context.WithTimeout(r.Context(), ResponseDeadline)
			defer cancel()
			authenticationOK, err := s.AuthenticateFromBasicToken(ctx, token)
			if err != nil {
				writeJsonHttpResponse(w, http.StatusUnauthorized, api.ErrorResponse{ErrorDescription: err.Error()})
				return
			}
			if !authenticationOK {
				writeJsonHttpResponse(w, http.StatusUnauthorized, api.ErrorResponse{ErrorDescription: "unauthorized"})
				return
			}
			next(w, r)
		}
	}
}

func tokenFromHeader(header http.Header) (string, error) {
	token := header.Get("Authorization")
	if token == "" {
		return "", api.ErrAuthorizationRequired
	}
	return token, nil
}
