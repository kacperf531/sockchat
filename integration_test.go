package sockchat

import (
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gorilla/websocket"
)

func TestIntegration(t *testing.T) {
	store, _ := NewSockChatStore()
	store.CreateChannel("foo")
	store.CreateChannel("bar")
	server := httptest.NewServer(NewSockChatServer(store))

	defer server.Close()

	t.Run("user can send a message to a channel they have joined", func(t *testing.T) {
		conn := mustDialWS(t, "ws"+strings.TrimPrefix(server.URL, "http")+"/ws")
		defer conn.Close()

		mustWriteWSMessage(t, conn, NewSocketMessage("join", JoinChannel{"foo"}))
		mustReadWSMessage(t, conn) // read the `channel_joined` from server

		mustWriteWSMessage(t, conn, NewSocketMessage("send_message", MessageEvent{"hi!", "foo"}))

		got := mustReadWSMessage(t, conn).Action
		want := "new_message"
		if got != want {
			t.Errorf("unexpected action returned from server, got %s, should be %s", got, want)
		}

	})

	t.Run("two users create the channel at the same time", func(t *testing.T) {
		conns := []*websocket.Conn{
			mustDialWS(t, "ws"+strings.TrimPrefix(server.URL, "http")+"/ws"),
			mustDialWS(t, "ws"+strings.TrimPrefix(server.URL, "http")+"/ws")}
		for _, conn := range conns {
			go func(conn *websocket.Conn) {
				mustWriteWSMessage(t, conn, NewSocketMessage("create_channel", CreateChannel{"foo"}))
				conn.Close()
			}(conn)
		}

	})
}
