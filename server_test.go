package sockchat

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/kacperf531/sockchat/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSockChatWS(t *testing.T) {
	channelStore := &StubChannelStore{Channels: map[string]*Channel{ChannelWithUser: {members: make(map[SockchatUserHandler]bool)}}}
	messageStore := &messageStoreSpy{}
	users := &userStoreDouble{}
	testTimeoutUnauthorized := 200 * time.Millisecond
	testTimeoutAuthorized := 20 * testTimeoutUnauthorized

	server := NewSockChatServer(channelStore, users, messageStore)

	server.SetTimeoutValues(testTimeoutAuthorized, testTimeoutUnauthorized)
	testServer := httptest.NewServer(server)
	wsURL := GetWsURL(testServer.URL)
	ws := NewTestWS(t, wsURL)

	defer ws.Close()
	defer testServer.Close()

	request := NewSocketMessage(LoginAction, LoginRequest{Nick: ValidUserNick, Password: ValidUserPassword})
	handler, _ := server.authorizedUsers.GetHandler(ValidUserNick)
	channelStore.Channels[ChannelWithUser].AddMember(handler)
	ws.Write(t, request)

	received := <-ws.MessageStash
	want := fmt.Sprintf("logged_in:%s", ValidUserNick)
	require.Equal(t, want, received.Action)

	t.Run("creates channel on request", func(t *testing.T) {
		channelName := "FooBar420"
		request := NewSocketMessage(CreateAction, ChannelRequest{Name: channelName})
		ws.Write(t, request)

		received := <-ws.MessageStash
		// User joins channel on creation
		want := UserJoinedChannelEvent
		require.Equal(t, want, received.Action)
	})

	t.Run("returns error on creating channel with existing name", func(t *testing.T) {
		request := NewSocketMessage(CreateAction, ChannelRequest{Name: "already_exists"})
		ws.Write(t, request)

		received := <-ws.MessageStash
		want := InvalidRequestEvent
		assert.Equal(t, want, received.Action)
	})

	t.Run("can join a channel", func(t *testing.T) {
		request := NewSocketMessage(JoinAction, ChannelRequest{Name: ChannelWithoutUser})
		ws.Write(t, request)

		received := <-ws.MessageStash
		want := UserJoinedChannelEvent
		require.Equal(t, want, received.Action)
		details := UnmarshalChannelUserChangeEvent(received.Payload)
		assert.Equal(t, ChannelWithoutUser, details.Channel)
		assert.Equal(t, ValidUserNick, details.Nick)
	})

	t.Run("can leave a channel", func(t *testing.T) {
		request := NewSocketMessage(LeaveAction, ChannelRequest{Name: ChannelWithUser})
		ws.Write(t, request)

		received := <-ws.MessageStash
		want := UserLeftChannelEvent
		require.Equal(t, received.Action, want)
		details := UnmarshalChannelUserChangeEvent(received.Payload)
		assert.Equal(t, details.Channel, ChannelWithUser)
		assert.Equal(t, details.Nick, ValidUserNick)
	})

	t.Run("error if leaving a channel user are not in", func(t *testing.T) {
		request := NewSocketMessage(LeaveAction, ChannelRequest{Name: ChannelWithoutUser})
		ws.Write(t, request)

		ws.AssertEventReceivedWithin(t, InvalidRequestEvent, 200*time.Millisecond)
	})

	t.Run("can not join a channel they are already in", func(t *testing.T) {
		request := NewSocketMessage(JoinAction, ChannelRequest{Name: ChannelWithUser})
		ws.Write(t, request)

		ws.AssertEventReceivedWithin(t, InvalidRequestEvent, 200*time.Millisecond)
	})

	t.Run("can not send a message to a channel being outside of", func(t *testing.T) {
		request := NewSocketMessage(SendMessageAction, SendMessageRequest{"foo", ChannelWithoutUser})
		ws.Write(t, request)

		ws.AssertEventReceivedWithin(t, InvalidRequestEvent, 200*time.Millisecond)

	})

	t.Run("unauthorized connection times out", func(t *testing.T) {
		new_ws := NewTestWS(t, wsURL)
		new_ws.AssertEventReceivedWithin(t, "connection_timed_out", testTimeoutUnauthorized+20*time.Millisecond)
	})

	t.Run("connection timeout period is extended after logging in", func(t *testing.T) {
		new_ws := NewTestWS(t, wsURL)
		request := NewSocketMessage(LoginAction, LoginRequest{Nick: ValidUserNick, Password: ValidUserPassword})
		new_ws.Write(t, request)
		<-new_ws.MessageStash // read `login` response
		select {
		case received := <-new_ws.MessageStash:
			if received.Action == "connection_timed_out" {
				t.Error("connection timed out", received)
			} else {
				t.Error("unexpected message received", received)
			}
		case <-time.After(testTimeoutUnauthorized + 20*time.Millisecond):
			return
		}
	})

}

func TestSockChatHTTP(t *testing.T) {
	sampleMessage := common.MessageEvent{Text: "foo", Channel: "bar", Author: "baz"}
	messageStore := &messageStoreStub{messages: []*common.MessageEvent{&sampleMessage}}
	channelStore := &StubChannelStore{make(map[string]*Channel)}
	users := &userStoreDouble{}

	server := NewSockChatServer(channelStore, users, messageStore)

	t.Run("can register a new user over HTTP endpoint", func(t *testing.T) {
		request := newRegisterRequest(CreateProfileRequest{Nick: "Foo", Password: "Bar420"})
		response := httptest.NewRecorder()

		server.ServeHTTP(response, request)

		assert.Equal(t, http.StatusCreated, response.Code)
	})

	t.Run("can not register a new user with missing required data", func(t *testing.T) {
		missingDataTests := []CreateProfileRequest{{Nick: "Foo"},
			{Password: "Bar42"}}
		for _, tt := range missingDataTests {
			request := newRegisterRequest(tt)
			response := httptest.NewRecorder()

			server.ServeHTTP(response, request)

			assert.Equal(t, http.StatusUnprocessableEntity, response.Code)
		}

	})

	t.Run("returns 409 on already existing nick", func(t *testing.T) {
		request := newRegisterRequest(CreateProfileRequest{Nick: "already_exists", Password: "Bar420"})
		response := httptest.NewRecorder()

		server.ServeHTTP(response, request)

		assert.Equal(t, http.StatusConflict, response.Code)
	})

	t.Run("can edit existing user over HTTP endpoint", func(t *testing.T) {
		request := newEditProfileRequest(EditProfileRequest{Description: "D3scription"})
		request.SetBasicAuth(ValidUserNick, ValidUserPassword)
		response := httptest.NewRecorder()

		server.ServeHTTP(response, request)

		assert.Equal(t, http.StatusOK, response.Code)
	})

	t.Run("returns 401 on invalid password", func(t *testing.T) {
		request := newEditProfileRequest(EditProfileRequest{Description: "D3scription"})
		request.SetBasicAuth(ValidUserNick, "fishyPassword")
		response := httptest.NewRecorder()

		server.ServeHTTP(response, request)

		assert.Equal(t, http.StatusUnauthorized, response.Code)
	})

	t.Run("returns error for unauthorized request to channel history", func(t *testing.T) {
		request := newChannelHistoryRequest(ChannelWithUser)
		response := httptest.NewRecorder()

		server.ServeHTTP(response, request)

		require.Equal(t, http.StatusUnauthorized, response.Code)
	})

	t.Run("returns error for request for history of channel which does not exist", func(t *testing.T) {
		request := newChannelHistoryRequest("not_exists")
		request.SetBasicAuth(ValidUserNick, ValidUserPassword)
		response := httptest.NewRecorder()

		server.ServeHTTP(response, request)

		require.Equal(t, http.StatusNotFound, response.Code)
	})

	t.Run("returns messages history", func(t *testing.T) {
		request := newChannelHistoryRequest(ChannelWithUser)
		request.SetBasicAuth(ValidUserNick, ValidUserPassword)
		response := httptest.NewRecorder()

		server.ServeHTTP(response, request)

		require.Equal(t, http.StatusOK, response.Code)
		sampleMessageBytes, _ := json.Marshal(sampleMessage)
		assert.Contains(t, response.Body.String(), string(sampleMessageBytes))
	})
}

func newRegisterRequest(b CreateProfileRequest) *http.Request {
	requestBytes, _ := json.Marshal(b)
	req, _ := http.NewRequest(http.MethodPost, "/register", bytes.NewBuffer(requestBytes))
	return req
}

func newEditProfileRequest(b EditProfileRequest) *http.Request {
	requestBytes, _ := json.Marshal(b)
	req, _ := http.NewRequest(http.MethodPost, "/edit_profile", bytes.NewBuffer(requestBytes))
	return req
}

func newChannelHistoryRequest(channel string) *http.Request {
	req, _ := http.NewRequest(http.MethodGet, "/history?channel="+channel, nil)
	return req
}
