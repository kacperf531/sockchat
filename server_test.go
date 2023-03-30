package sockchat

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

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
	server := httptest.NewServer(NewSockChatServer(store, &users))
	ws := mustDialWS(t, "ws"+strings.TrimPrefix(server.URL, "http")+"/ws")

	defer ws.Close()
	defer server.Close()

	t.Run("creates channel on request", func(t *testing.T) {
		request := NewSocketMessage(CreateAction, ChannelRequest{Name: "FooBar420"})
		mustWriteWSMessage(t, ws, request)

		got := mustReadWSMessage(t, ws).Action
		want := "channel_created"
		AssertResponseAction(t, got, want)
	})

	t.Run("returns error on creating channel with existing name", func(t *testing.T) {
		request := NewSocketMessage(CreateAction, ChannelRequest{Name: "already_exists"})
		mustWriteWSMessage(t, ws, request)

		got := mustReadWSMessage(t, ws).Action
		want := "invalid_request_received"
		AssertResponseAction(t, got, want)
	})

	t.Run("can join a channel", func(t *testing.T) {
		request := NewSocketMessage(JoinAction, ChannelRequest{Name: ChannelWithoutUser})
		mustWriteWSMessage(t, ws, request)

		got := mustReadWSMessage(t, ws).Action
		want := "channel_joined"
		AssertResponseAction(t, got, want)
	})

	t.Run("can leave a channel", func(t *testing.T) {
		request := NewSocketMessage(LeaveAction, ChannelRequest{Name: ChannelWithUser})
		mustWriteWSMessage(t, ws, request)

		got := mustReadWSMessage(t, ws).Action
		want := "channel_left"
		AssertResponseAction(t, got, want)
	})

	t.Run("error if leaving a channel user are not in", func(t *testing.T) {
		request := NewSocketMessage(LeaveAction, ChannelRequest{Name: ChannelWithoutUser})
		mustWriteWSMessage(t, ws, request)

		got := mustReadWSMessage(t, ws).Action
		want := "invalid_request_received"
		AssertResponseAction(t, got, want)
	})

	t.Run("can not join a channel they are already in", func(t *testing.T) {
		request := NewSocketMessage(JoinAction, ChannelRequest{Name: ChannelWithUser})
		mustWriteWSMessage(t, ws, request)

		got := mustReadWSMessage(t, ws).Action
		want := "invalid_request_received"
		AssertResponseAction(t, got, want)
	})

	t.Run("can not send a message to a channel being outside of", func(t *testing.T) {
		request := NewSocketMessage(SendMessageAction, MessageEvent{"foo", ChannelWithoutUser})
		mustWriteWSMessage(t, ws, request)

		got := mustReadWSMessage(t, ws).Action
		want := "invalid_request_received"
		AssertResponseAction(t, got, want)

	})

}

func AssertResponseAction(t *testing.T, got, want string) {
	t.Helper()
	if got != want {
		t.Errorf("unexpected action returned from server, got %s, should be %s", got, want)
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

	t.Run("can edit existing user over HTTP endpoint", func(t *testing.T) {
		request := newEditProfileRequest(UserRequest{Nick: "Foo", Description: "D3scription", Password: "foo420"})
		response := httptest.NewRecorder()

		server.ServeHTTP(response, request)

		assert.Equal(t, http.StatusOK, response.Code)
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
