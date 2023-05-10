package services_test

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/kacperf531/sockchat"
	"github.com/kacperf531/sockchat/api"
	"github.com/kacperf531/sockchat/services"
	"github.com/kacperf531/sockchat/test_utils"
	"github.com/stretchr/testify/require"
)

func TestSockChatWebAPI(t *testing.T) {
	sampleMessage := api.MessageEvent{Text: "foo", Channel: "bar", Author: "baz"}
	messageStore := &test_utils.StubMessageStore{Messages: api.ChannelHistory{&sampleMessage}}
	userProfiles := &sockchat.ProfileService{Store: &test_utils.UserStoreDouble{}, Cache: test_utils.TestingRedisClient}
	validToken := "Basic " + base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("%s:%s", test_utils.ValidUserNick, test_utils.ValidUserPassword)))

	router := http.NewServeMux()

	core := &services.SockchatCoreService{UserProfiles: userProfiles, ChatChannels: &test_utils.StubChannelStore{}, Messages: messageStore}
	webAPI := services.NewWebAPI(core, &services.SockchatAuthService{UserProfiles: userProfiles})
	webAPI.HandleRequests(router)

	t.Run("can register over HTTP", func(t *testing.T) {
		req := newRegisterRequest(api.CreateProfileRequest{Nick: test_utils.ValidUserNick, Password: test_utils.ValidUserPassword})
		res := httptest.NewRecorder()

		router.ServeHTTP(res, req)
		require.Equal(t, http.StatusCreated, res.Code)
	})

	t.Run("returns error for unauthorized request to channel history", func(t *testing.T) {
		req := newChannelHistoryRequest(test_utils.ChannelWithUser)
		res := httptest.NewRecorder()

		router.ServeHTTP(res, req)
		require.Equal(t, http.StatusUnauthorized, res.Code)
		require.Equal(t, api.ErrorResponse{ErrorDescription: api.ErrAuthorizationRequired.Error()}, decodeErrorResponse(res.Body))
	})

	t.Run("returns history for authorized request", func(t *testing.T) {
		req := newChannelHistoryRequest(test_utils.ChannelWithUser)
		res := httptest.NewRecorder()
		req.Header.Set("authorization", validToken)

		router.ServeHTTP(res, req)
		require.Equal(t, http.StatusOK, res.Code)
		require.Equal(t, api.ChannelHistory{&sampleMessage}, decodeChannelHistoryResponse(res.Body))
	})

	t.Run("returns error for unauthorized request to edit profile", func(t *testing.T) {
		req := newEditProfileRequest(api.EditProfileRequest{Description: "bar"})
		res := httptest.NewRecorder()

		router.ServeHTTP(res, req)
		require.Equal(t, http.StatusUnauthorized, res.Code)
	})

	t.Run("returns error for unauthorized request to get profile", func(t *testing.T) {
		req := newGetProfileRequest(test_utils.ValidUserNick)
		res := httptest.NewRecorder()

		router.ServeHTTP(res, req)
		require.Equal(t, http.StatusUnauthorized, res.Code)
	})

}

func newRegisterRequest(b api.CreateProfileRequest) *http.Request {
	requestBytes, _ := json.Marshal(b)
	req, _ := http.NewRequest(http.MethodPost, "/register", bytes.NewBuffer(requestBytes))
	return req
}

func newGetProfileRequest(nick string) *http.Request {
	req, _ := http.NewRequest(http.MethodGet, "/profile?nick="+nick, nil)
	return req
}

func newEditProfileRequest(b api.EditProfileRequest) *http.Request {
	requestBytes, _ := json.Marshal(b)
	req, _ := http.NewRequest(http.MethodPost, "/edit_profile", bytes.NewBuffer(requestBytes))
	return req
}

func newChannelHistoryRequest(channel string) *http.Request {
	req, _ := http.NewRequest(http.MethodGet, "/history?channel="+channel, nil)
	return req
}

func decodeErrorResponse(b *bytes.Buffer) api.ErrorResponse {
	var errResponse api.ErrorResponse
	json.NewDecoder(b).Decode(&errResponse)
	return errResponse
}

func decodeChannelHistoryResponse(b *bytes.Buffer) api.ChannelHistory {
	var history api.ChannelHistory
	json.NewDecoder(b).Decode(&history)
	return history
}
