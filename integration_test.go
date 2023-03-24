package sockchat

import (
	"net/http/httptest"
	"strings"
	"testing"
)

func TestIntegration(t *testing.T) {
	store, _ := NewSockChatStore()
	store.CreateChannel("foo")
	store.CreateChannel("bar")
	server := httptest.NewServer(NewSockChatServer(store))
	ws := mustDialWS(t, "ws"+strings.TrimPrefix(server.URL, "http")+"/ws")

	defer ws.Close()
	defer server.Close()

	t.Run("user can send a message to a channel they have joined", func(t *testing.T) {
		mustWriteWSMessage(t, ws, NewSocketMessage("join", JoinChannel{"foo"}))
		mustReadWSMessage(t, ws) // read the `channel_joined` from server

		mustWriteWSMessage(t, ws, NewSocketMessage("send_message", MessageEvent{"hi!", "foo"}))

		got := mustReadWSMessage(t, ws).Action
		want := "new_message"
		if got != want {
			t.Errorf("unexpected action returned from server, got %s, should be %s", got, want)
		}

	})
}
