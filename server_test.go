package sockchat

import (
	"encoding/json"
	"fmt"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gorilla/websocket"
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
		return fmt.Errorf("User already in channel")
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

	t.Run("can send a message to a channel being in", func(t *testing.T) {
		request := NewSocketMessage("send_message", MessageEvent{"foo", ChannelWithUser})
		mustWriteWSMessage(t, ws, request)

		got := mustReadWSMessage(t, ws).Action
		want := "new_message"
		if got != want {
			t.Errorf("unexpected action returned from server, got %s, should be %s", got, want)
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

func mustDialWS(t *testing.T, url string) *websocket.Conn {
	ws, _, err := websocket.DefaultDialer.Dial(url, nil)

	if err != nil {
		t.Fatalf("could not open a ws connection on %s %v", url, err)
	}

	return ws
}

func mustWriteWSMessage(t testing.TB, conn *websocket.Conn, message SocketMessage) {
	t.Helper()
	payloadBytes, err := json.Marshal(message)
	if err != nil {
		t.Fatalf("could not marshal message before sending to the server %v", err)
	}
	if err := conn.WriteMessage(websocket.TextMessage, payloadBytes); err != nil {
		t.Fatalf("could not send message over ws connection %v", err)
	}
}

func mustReadWSMessage(t testing.TB, conn *websocket.Conn) SocketMessage {
	t.Helper()
	receivedMessage := &SocketMessage{}
	if err := conn.ReadJSON(receivedMessage); err != nil {
		t.Fatalf("could not parse message coming from ws %v", err)
	}
	return *receivedMessage
}
