package sockchat

import (
	"net/http/httptest"
	"testing"

	"github.com/gorilla/websocket"
)

func TestIntegration(t *testing.T) {
	store, _ := NewSockChatStore()
	store.CreateChannel("foo")
	store.CreateChannel("bar")
	users := userStoreSpy{}
	server := httptest.NewServer(NewSockChatServer(store, &users))

	defer server.Close()

	t.Run("user can send a message to a channel they have joined", func(t *testing.T) {
		conn := mustDialWS(t, GetWsURL(server.URL))
		defer conn.Close()

		mustWriteWSMessage(t, conn, NewSocketMessage(JoinAction, ChannelRequest{"foo"}))
		mustReadWSMessage(t, conn) // read the `channel_joined` from server

		mustWriteWSMessage(t, conn, NewSocketMessage(SendMessageAction, MessageEvent{"hi!", "foo"}))

		got := mustReadWSMessage(t, conn).Action
		want := "new_message"
		if got != want {
			t.Errorf("unexpected action returned from server, got %s, should be %s", got, want)
		}

	})

	t.Run("two users create the channel at the same time", func(t *testing.T) {
		conns := []*websocket.Conn{
			mustDialWS(t, GetWsURL(server.URL)),
			mustDialWS(t, GetWsURL(server.URL))}
		for _, conn := range conns {
			go func(conn *websocket.Conn) {
				mustWriteWSMessage(t, conn, NewSocketMessage(CreateAction, ChannelRequest{"foo"}))
				conn.Close()
			}(conn)
		}

	})
}
