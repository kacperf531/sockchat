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
	Channels map[string]*Channel
}

func (store *StubChannelStore) GetChannel(name string) (*Channel, error) {
	switch name {
	case ChannelWithoutUser:
		return &Channel{}, nil
	case ChannelWithUser:
		return store.Channels[ChannelWithUser], nil
	}
	return store.Channels[name], nil
}

func (store *StubChannelStore) CreateChannel(name string) error {
	if name == "already_exists" {
		return fmt.Errorf("channel `%s` already exists", name)
	}
	return nil
}

func (store *StubChannelStore) JoinChannel(channelName string, conn *SockChatWS) error {
	switch channelName {
	case ChannelWithoutUser:
		return nil
	case ChannelWithUser:
		return fmt.Errorf("user already in channel")
	}
	return nil
}

func (store *StubChannelStore) ChannelHasUser(channelName string, conn *SockChatWS) bool {
	return channelName == ChannelWithUser
}

func TestSockChat(t *testing.T) {
	store := &StubChannelStore{map[string]*Channel{}}
	server := httptest.NewServer(NewSockChatServer(store))
	ws := mustDialWS(t, "ws"+strings.TrimPrefix(server.URL, "http")+"/ws")

	defer ws.Close()
	defer server.Close()

	t.Run("creates channel on request", func(t *testing.T) {
		request := NewSocketMessage("create", Channel{Name: "FooBar420"})
		mustWriteWSMessage(t, ws, request)

		got := mustReadWSMessage(t, ws).Action
		want := "channel_created"
		if got != want {
			t.Errorf("unexpected action returned from server, got %s, want %s", got, want)
		}
	})

	t.Run("returns error on creating channel with existing name", func(t *testing.T) {
		request := NewSocketMessage("create", Channel{Name: "already_exists"})
		mustWriteWSMessage(t, ws, request)

		got := mustReadWSMessage(t, ws).Action
		want := "invalid_request_received"
		if got != want {
			t.Errorf("unexpected action returned from server")
		}
	})

	t.Run("can join a channel", func(t *testing.T) {
		request := NewSocketMessage("join", Channel{Name: ChannelWithoutUser})
		mustWriteWSMessage(t, ws, request)

		got := mustReadWSMessage(t, ws).Action
		want := "channel_joined"
		if got != want {
			t.Errorf("unexpected action returned from server")
		}
	})

	t.Run("can not join a channel they are already in", func(t *testing.T) {
		request := NewSocketMessage("join", Channel{Name: ChannelWithUser})
		mustWriteWSMessage(t, ws, request)

		got := mustReadWSMessage(t, ws).Action
		want := "invalid_request_received"
		if got != want {
			t.Errorf("unexpected action returned from server")
		}
	})

	t.Run("can not send a message to a channel being outside of", func(t *testing.T) {
		request := NewSocketMessage("send_message", MessageEvent{"foo", ChannelWithoutUser})
		mustWriteWSMessage(t, ws, request)

		got := mustReadWSMessage(t, ws).Action
		want := "invalid_request_received"
		if got != want {
			t.Errorf("unexpected action returned from server, got %s, should be %s", got, want)
		}

	})

}
