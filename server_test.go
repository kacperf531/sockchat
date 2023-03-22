package sockchat

import (
	"encoding/json"
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
	store.Channels[name] = &Channel{}
	return nil
}

func TestSockChat(t *testing.T) {

	t.Run("create a chat channel", func(t *testing.T) {

		newChannel := Channel{Name: "Foo420"}
		payloadBytes, _ := json.Marshal(newChannel)
		request := WSMsg{Action: "create", Payload: payloadBytes}

		store := &StubChannelStore{make(map[string]*Channel)}
		server := httptest.NewServer(NewSockChatServer(store))
		ws := mustDialWS(t, "ws"+strings.TrimPrefix(server.URL, "http")+"/ws")

		defer server.Close()
		defer ws.Close()

		mustWriteWSMessage(t, ws, request)
		AssertChannelExists(t, store, newChannel.Name)
	})
}

func mustDialWS(t *testing.T, url string) *websocket.Conn {
	ws, _, err := websocket.DefaultDialer.Dial(url, nil)

	if err != nil {
		t.Fatalf("could not open a ws connection on %s %v", url, err)
	}

	return ws
}

func mustWriteWSMessage(t testing.TB, conn *websocket.Conn, message WSMsg) {
	t.Helper()
	payloadBytes, err := json.Marshal(message)
	if err != nil {
		t.Fatalf("could not marshal message before sending to the server %v", err)
	}
	if err := conn.WriteMessage(websocket.TextMessage, payloadBytes); err != nil {
		t.Fatalf("could not send message over ws connection %v", err)
	}
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
