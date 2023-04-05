package sockchat

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

const (
	ChannelWithUser    = "channel_with_user"
	ChannelWithoutUser = "channel_without_user"
)

// StubChannelStore implements ChannelStore for testing purposes
type StubChannelStore struct {
	*SockChatStore
	Channels map[string]*Channel
}

func (store *StubChannelStore) CreateChannel(name string) error {
	if name == "already_exists" {
		return fmt.Errorf("channel `%s` already exists", name)
	}
	return nil
}

func (store *StubChannelStore) AddUserToChannel(channelName string, conn *SockChatWS) error {
	if channelName == ChannelWithUser {
		return fmt.Errorf("user already in channel")
	}
	return nil
}

func (store *StubChannelStore) RemoveUserFromChannel(channelName string, conn *SockChatWS) error {
	if channelName != ChannelWithUser {
		return fmt.Errorf("user is not member of the channel")
	}
	return nil
}

func (store *StubChannelStore) ChannelHasUser(channelName string, conn *SockChatWS) bool {
	return channelName == ChannelWithUser
}

func TestSockChatWS(t *testing.T) {
	store := &StubChannelStore{Channels: map[string]*Channel{}}
	users := userStoreDouble{}
	testTimeoutUnauthorized := 200 * time.Millisecond
	testTimeoutAuthorized := 60 * testTimeoutUnauthorized

	server := NewSockChatServer(store, &users)
	server.SetTimeoutValues(testTimeoutAuthorized, testTimeoutUnauthorized)
	testServer := httptest.NewServer(server)
	wsURL := GetWsURL(testServer.URL)
	ws := NewTestWS(t, wsURL)

	defer ws.Close()
	defer testServer.Close()

	t.Run("existing user can log in", func(t *testing.T) {
		request := NewSocketMessage(LoginAction, LoginRequest{Nick: ValidUserNick, Password: ValidUserPassword})
		ws.Write(t, request)

		received := <-ws.MessageStash
		want := fmt.Sprintf("logged_in:%s", ValidUserNick)
		AssertResponseAction(t, received.Action, want)
	})

	t.Run("creates channel on request", func(t *testing.T) {
		request := NewSocketMessage(CreateAction, ChannelRequest{Name: "FooBar420"})
		ws.Write(t, request)

		received := <-ws.MessageStash
		want := "channel_created"
		AssertResponseAction(t, received.Action, want)
	})

	t.Run("returns error on creating channel with existing name", func(t *testing.T) {
		request := NewSocketMessage(CreateAction, ChannelRequest{Name: "already_exists"})
		ws.Write(t, request)

		received := <-ws.MessageStash
		want := "invalid_request_received"
		AssertResponseAction(t, received.Action, want)
	})

	t.Run("can join a channel", func(t *testing.T) {
		request := NewSocketMessage(JoinAction, ChannelRequest{Name: ChannelWithoutUser})
		ws.Write(t, request)

		received := <-ws.MessageStash
		want := "channel_joined"
		AssertResponseAction(t, received.Action, want)
	})

	t.Run("can leave a channel", func(t *testing.T) {
		request := NewSocketMessage(LeaveAction, ChannelRequest{Name: ChannelWithUser})
		ws.Write(t, request)

		received := <-ws.MessageStash
		want := "channel_left"
		AssertResponseAction(t, received.Action, want)
	})

	t.Run("error if leaving a channel user are not in", func(t *testing.T) {
		request := NewSocketMessage(LeaveAction, ChannelRequest{Name: ChannelWithoutUser})
		ws.Write(t, request)

		received := <-ws.MessageStash
		want := "invalid_request_received"
		AssertResponseAction(t, received.Action, want)
	})

	t.Run("can not join a channel they are already in", func(t *testing.T) {
		request := NewSocketMessage(JoinAction, ChannelRequest{Name: ChannelWithUser})
		ws.Write(t, request)

		received := <-ws.MessageStash
		want := "invalid_request_received"
		AssertResponseAction(t, received.Action, want)
	})

	t.Run("can not send a message to a channel being outside of", func(t *testing.T) {
		request := NewSocketMessage(SendMessageAction, MessageEvent{"foo", ChannelWithoutUser})
		ws.Write(t, request)

		received := <-ws.MessageStash
		want := "invalid_request_received"
		AssertResponseAction(t, received.Action, want)

	})

	t.Run("unauthorized connection times out", func(t *testing.T) {
		new_ws := NewTestWS(t, wsURL)
		within(t, testTimeoutUnauthorized+20*time.Millisecond, func() { received := <-new_ws.MessageStash; assert.Equal(t, received.Action, "connection_timed_out") })
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
case <-time.After(testTimeoutUnauthorized+20*time.Millisecond):
	return
}
	})

}

func AssertResponseAction(t *testing.T, got, want string) {
	t.Helper()
	if got != want {
		t.Errorf("unexpected action returned from server, got %s, should be %s", got, want)
	}
}

func within(t testing.TB, d time.Duration, assert func()) {
	t.Helper()

	done := make(chan struct{}, 1)

	go func() {
		assert()
		done <- struct{}{}
	}()

	select {
	case <-time.After(d):
		t.Error("timed out")
	case <-done:
	}
}

func TestSockChatHTTP(t *testing.T) {
	store := &StubChannelStore{Channels: map[string]*Channel{}}
	users := userStoreDouble{}
	server := NewSockChatServer(store, &users)

	t.Run("can register a new user over HTTP endpoint", func(t *testing.T) {
		request := newRegisterRequest(UserRequest{Nick: "Foo", Password: "Bar420"})
		response := httptest.NewRecorder()

		server.ServeHTTP(response, request)

		assert.Equal(t, http.StatusCreated, response.Code)
	})

	t.Run("can not register a new user with missing required data", func(t *testing.T) {
		missingDataTests := []UserRequest{{Nick: "Foo"},
			{Password: "Bar42"}}
		for _, tt := range missingDataTests {
			request := newRegisterRequest(tt)
			response := httptest.NewRecorder()

			server.ServeHTTP(response, request)

			assert.Equal(t, http.StatusUnprocessableEntity, response.Code)
		}

	})

	t.Run("returns 409 on already existing nick", func(t *testing.T) {
		request := newRegisterRequest(UserRequest{Nick: "already_exists", Password: "Bar420"})
		response := httptest.NewRecorder()

		server.ServeHTTP(response, request)

		assert.Equal(t, http.StatusConflict, response.Code)
	})

	t.Run("can edit existing user over HTTP endpoint", func(t *testing.T) {
		request := newEditProfileRequest(UserRequest{Nick: ValidUserNick, Description: "D3scription", Password: ValidUserPassword})
		response := httptest.NewRecorder()

		server.ServeHTTP(response, request)

		assert.Equal(t, http.StatusOK, response.Code)
	})

	t.Run("returns 401 on invalid password", func(t *testing.T) {
		request := newEditProfileRequest(UserRequest{Nick: "Foo", Description: "D3scription", Password: "fishyPassword"})
		response := httptest.NewRecorder()

		server.ServeHTTP(response, request)

		assert.Equal(t, http.StatusUnauthorized, response.Code)
	})
}

func newRegisterRequest(b UserRequest) *http.Request {
	requestBytes, _ := json.Marshal(b)
	req, _ := http.NewRequest(http.MethodGet, "/register", bytes.NewBuffer(requestBytes))
	return req
}

func newEditProfileRequest(b UserRequest) *http.Request {
	requestBytes, _ := json.Marshal(b)
	req, _ := http.NewRequest(http.MethodGet, "/edit_profile", bytes.NewBuffer(requestBytes))
	return req
}
