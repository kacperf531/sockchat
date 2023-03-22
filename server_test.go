package sockchat

import (
	"encoding/json"
	"fmt"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

// StubChannelStore implements ChannelStore for testing purposes
type StubChannelStore struct {
	Channels map[string]*Channel
}

func (store *StubChannelStore) GetChannel(name string) *Channel {
	return store.Channels[name]
}

func (store *StubChannelStore) CreateChannel(name string) error {
	if store.GetChannel(name) != nil {
		return fmt.Errorf("channel `%s` already exists", name)
	}
	store.Channels[name] = &Channel{}
	return nil
}

func (store *StubChannelStore) JoinChannel(channelName string, conn *SockChatWS) error {
	channel := store.GetChannel(channelName)
	if channel == nil {
		return fmt.Errorf("channel `%s` does not exist", channelName)
	}
	channel.Users = append(channel.Users, conn)
	return nil
}

func TestSockChat(t *testing.T) {
	store := &StubChannelStore{map[string]*Channel{"Foo420": {}}}
	server := httptest.NewServer(NewSockChatServer(store))

	defer server.Close()

	t.Run("create a chat channel", func(t *testing.T) {
		ws := mustDialWS(t, "ws"+strings.TrimPrefix(server.URL, "http")+"/ws")
		defer ws.Close()

		newChannel := Channel{Name: "Foo123"}
		payloadBytes, _ := json.Marshal(newChannel)
		request := SocketMessage{Action: "create", Payload: payloadBytes}

		mustWriteWSMessage(t, ws, request)
		receivedMessage := mustReadWSMessage(t, ws)
		got := receivedMessage.Action
		want := "channel_created"
		if got != "channel_created" {
			t.Errorf("unexpected action returned from server, got %s, want %s", got, want)
		}
		AssertChannelExists(t, store, newChannel.Name)
	})

	t.Run("can not create channel with existing name", func(t *testing.T) {
		ws := mustDialWS(t, "ws"+strings.TrimPrefix(server.URL, "http")+"/ws")
		defer ws.Close()

		newChannel := Channel{Name: "Foo420"}
		payloadBytes, _ := json.Marshal(newChannel)
		request := SocketMessage{Action: "create", Payload: payloadBytes}

		mustWriteWSMessage(t, ws, request)
		got := mustReadWSMessage(t, ws).Action
		want := "invalid_request_received"
		if got != want {
			t.Errorf("unexpected action returned from server")
		}
	})

	t.Run("can join a channel", func(t *testing.T) {
		ws := mustDialWS(t, "ws"+strings.TrimPrefix(server.URL, "http")+"/ws")
		defer ws.Close()

		channelName := "Foo420"
		payloadBytes, _ := json.Marshal(Channel{Name: channelName})
		request := SocketMessage{Action: "join", Payload: payloadBytes}

		mustWriteWSMessage(t, ws, request)
		got := mustReadWSMessage(t, ws).Action
		want := "channel_joined"
		if got != want {
			t.Errorf("unexpected action returned from server")
		}
		channelUserCount := len(store.GetChannel(channelName).Users)
		if channelUserCount != 1 {
			t.Errorf("expected exactly 1 user connected to %s channel, got %d", channelName, channelUserCount)
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

func AssertChannelExists(t *testing.T, store ChannelStore, channel string) {
	t.Helper()
	passed := retryUntil(100*time.Millisecond, func() bool {
		return store.GetChannel(channel) != nil
	})

	if !passed {
		t.Error("expected channel, got nil")
	}
}

func retryUntil(d time.Duration, f func() bool) bool {
	deadline := time.Now().Add(d)
	for time.Now().Before(deadline) {
		if f() {
			return true
		}
	}
	return false
}
