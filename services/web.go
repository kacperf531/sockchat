package services

import (
	"context"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"time"

	"github.com/kacperf531/sockchat/api"
)

const ResponseDeadline = 5 * time.Second

var HTTPStatuses = map[error]int{
	api.ErrNickAlreadyUsed:  http.StatusConflict,
	api.ErrNickRequired:     http.StatusUnprocessableEntity,
	api.ErrPasswordRequired: http.StatusUnprocessableEntity,
	api.ErrInvalidRequest:   http.StatusBadRequest,
	api.ErrChannelNotFound:  http.StatusNotFound,
	api.ErrInternal:         http.StatusInternalServerError,
}

type WebAPI struct {
	CoreService *SockchatCoreService
	AuthService *SockchatAuthService
}

func NewWebAPI(core *SockchatCoreService, authService *SockchatAuthService) *WebAPI {
	return &WebAPI{CoreService: core, AuthService: authService}
}

func (s *WebAPI) HandleRequests(router *http.ServeMux) {
	router.Handle("/register", http.HandlerFunc(s.registerProfile))

	authenticate := newAuthMiddleware(s.AuthService)
	router.Handle("/edit_profile", authenticate(s.editProfile))
	router.Handle("/history", authenticate(s.getChannelHistory))
	router.Handle("/profile", authenticate(s.getProfile))
}

func (s *WebAPI) registerProfile(w http.ResponseWriter, r *http.Request) {
	userData := readCreateProfileRequest(w, r)
	ctx, cancel := context.WithTimeout(r.Context(), ResponseDeadline)
	defer cancel()
	res, err := s.CoreService.RegisterProfile(userData, ctx)
	if err != nil {
		writeJsonHttpResponse(w, HTTPStatuses[err], &api.ErrorResponse{ErrorDescription: err.Error()})
		return
	}
	writeJsonHttpResponse(w, http.StatusCreated, res)
}

func (s *WebAPI) getProfile(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), ResponseDeadline)
	defer cancel()
	res, err := s.CoreService.GetProfile(&api.GetProfileRequest{Nick: r.URL.Query().Get("nick")}, ctx)
	if err != nil {
		writeJsonHttpResponse(w, HTTPStatuses[err], &api.ErrorResponse{ErrorDescription: err.Error()})
		return
	}
	writeJsonHttpResponse(w, http.StatusOK, res)
}

func (s *WebAPI) editProfile(w http.ResponseWriter, r *http.Request) {
	userData := readEditProfileRequest(w, r)
	ctx, cancel := context.WithTimeout(r.Context(), ResponseDeadline)
	defer cancel()
	username, _, _ := r.BasicAuth()
	res, err := s.CoreService.EditProfile(&EditProfileWrapper{Nick: username, Request: userData}, ctx)
	if err != nil {
		writeJsonHttpResponse(w, HTTPStatuses[err], &api.ErrorResponse{ErrorDescription: err.Error()})
		return
	}
	writeJsonHttpResponse(w, http.StatusOK, res)
}

func (s *WebAPI) getChannelHistory(w http.ResponseWriter, r *http.Request) {
	channelName := r.URL.Query().Get("channel")
	soughtPhrase := r.URL.Query().Get("search")
	ctx, cancel := context.WithTimeout(r.Context(), ResponseDeadline)
	defer cancel()
	res, err := s.CoreService.GetChannelHistory(&api.GetChannelHistoryRequest{Channel: channelName, Search: soughtPhrase}, ctx)
	if err != nil {
		writeJsonHttpResponse(w, HTTPStatuses[err], &api.ErrorResponse{ErrorDescription: err.Error()})
		return
	}
	writeJsonHttpResponse(w, http.StatusOK, res)
}

func writeJsonHttpResponse(w http.ResponseWriter, statusCode int, data interface{}) error {
	output, err := json.Marshal(data)
	if err != nil {
		return err
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	_, err = w.Write(output)
	if err != nil {
		log.Print(err)
		return nil
	}
	return nil
}

func readCreateProfileRequest(w http.ResponseWriter, r *http.Request) *api.CreateProfileRequest {
	req, err := ParseRequest(r, "create_profile")
	if err != nil {
		writeJsonHttpResponse(w, http.StatusBadRequest, api.NewSocketError(api.ErrInvalidRequest.Error()))
		return nil
	}
	return req.(*api.CreateProfileRequest)
}

func readEditProfileRequest(w http.ResponseWriter, r *http.Request) *api.EditProfileRequest {
	req, err := ParseRequest(r, "edit_profile")
	if err != nil {
		writeJsonHttpResponse(w, http.StatusBadRequest, api.NewSocketError(api.ErrInvalidRequest.Error()))
		return nil
	}
	return req.(*api.EditProfileRequest)
}

func ParseRequest(r *http.Request, action string) (any, error) {
	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		return nil, err
	}
	switch action {
	case "create_profile":
		return api.UnmarshalCreateProfileRequest(bodyBytes)
	case "edit_profile":
		return api.UnmarshalEditProfileRequest(bodyBytes)
	}
	return nil, api.ErrInvalidRequest
}
