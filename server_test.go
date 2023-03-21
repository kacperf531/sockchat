package sockchat

import (
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

// StubChannelStore implements ChannelStore for testing purposes
type StubChannelStore struct {
	Channels map[string]int
}

func (store *StubChannelStore) GetChannel(name string) int {
	return store.Channels[name]
}

func (store *StubChannelStore) CreateChannel(name string) int {
	store.Channels[name] = 1
	return 1
}

func TestSockChat(t *testing.T) {

	t.Run("create a chat channel", func(t *testing.T) {

		channelName := "Foo420"

		channels := &StubChannelStore{make(map[string]int)}
		server := httptest.NewServer(NewSockChatServer(channels))
		ws := mustDialWS(t, "ws"+strings.TrimPrefix(server.URL, "http")+"/ws")

		defer server.Close()
		defer ws.Close()

		writeWSMessage(t, ws, channelName)
		AssertChannelExists(t, channels, channelName)
	})
}

func mustDialWS(t *testing.T, url string) *websocket.Conn {
	ws, _, err := websocket.DefaultDialer.Dial(url, nil)

	if err != nil {
		t.Fatalf("could not open a ws connection on %s %v", url, err)
	}

	return ws
}

func writeWSMessage(t testing.TB, conn *websocket.Conn, message string) {
	t.Helper()
	if err := conn.WriteMessage(websocket.TextMessage, []byte(message)); err != nil {
		t.Fatalf("could not send message over ws connection %v", err)
	}
}

func AssertChannelExists(t *testing.T, channels ChannelStore, soughtChannel string) {
	t.Helper()

	passed := retryUntil(100*time.Millisecond, func() bool {
		return channels.GetChannel(soughtChannel) != 0
	})

	if !passed {
		t.Errorf("expected channel ID other than 0")
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
