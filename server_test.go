package sockchat

import (
	"fmt"
	"net/http/httptest"
	"strings"
	"testing"
)

const ChannelWithUser = "channel_with_user"
const ChannelWithoutUser = "channel_without_user"

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

func TestSockChat(t *testing.T) {
	store := &StubChannelStore{Channels: map[string]*Channel{}}
	server := httptest.NewServer(NewSockChatServer(store))
	ws := mustDialWS(t, "ws"+strings.TrimPrefix(server.URL, "http")+"/ws")

	defer ws.Close()
	defer server.Close()

	t.Run("creates channel on request", func(t *testing.T) {
		request := NewSocketMessage("create", ChannelRequest{Name: "FooBar420"})
		mustWriteWSMessage(t, ws, request)

		got := mustReadWSMessage(t, ws).Action
		want := "channel_created"
		AssertResponseAction(t, got, want)
	})

	t.Run("returns error on creating channel with existing name", func(t *testing.T) {
		request := NewSocketMessage("create", ChannelRequest{Name: "already_exists"})
		mustWriteWSMessage(t, ws, request)

		got := mustReadWSMessage(t, ws).Action
		want := "invalid_request_received"
		AssertResponseAction(t, got, want)
	})

	t.Run("can join a channel", func(t *testing.T) {
		request := NewSocketMessage("join", ChannelRequest{Name: ChannelWithoutUser})
		mustWriteWSMessage(t, ws, request)

		got := mustReadWSMessage(t, ws).Action
		want := "channel_joined"
		AssertResponseAction(t, got, want)
	})

	t.Run("can leave a channel", func(t *testing.T) {
		request := NewSocketMessage("leave", ChannelRequest{Name: ChannelWithUser})
		mustWriteWSMessage(t, ws, request)

		got := mustReadWSMessage(t, ws).Action
		want := "channel_left"
		AssertResponseAction(t, got, want)
	})

	t.Run("error if leaving a channel user are not in", func(t *testing.T) {
		request := NewSocketMessage("leave", ChannelRequest{Name: ChannelWithoutUser})
		mustWriteWSMessage(t, ws, request)

		got := mustReadWSMessage(t, ws).Action
		want := "invalid_request_received"
		AssertResponseAction(t, got, want)
	})

	t.Run("can not join a channel they are already in", func(t *testing.T) {
		request := NewSocketMessage("join", ChannelRequest{Name: ChannelWithUser})
		mustWriteWSMessage(t, ws, request)

		got := mustReadWSMessage(t, ws).Action
		want := "invalid_request_received"
		AssertResponseAction(t, got, want)
	})

	t.Run("can not send a message to a channel being outside of", func(t *testing.T) {
		request := NewSocketMessage("send_message", MessageEvent{"foo", ChannelWithoutUser})
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
