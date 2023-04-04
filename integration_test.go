package sockchat

import (
	"net/http/httptest"
	"testing"
)

func TestIntegration(t *testing.T) {
	store, _ := NewSockChatStore()
	store.CreateChannel("foo")
	store.CreateChannel("bar")
	users := userStoreDouble{}
	server := httptest.NewServer(NewSockChatServer(store, &users))
	wsURL := GetWsURL(server.URL)

	defer server.Close()

	t.Run("user can send a message to a channel they have joined", func(t *testing.T) {
		ws := NewTestWS(t, wsURL)
		ws.Write(t, NewSocketMessage(LoginAction, LoginRequest{Nick: ValidUserNick, Password: ValidUserPassword}))
		<-ws.MessageStash // discard the login response from server
		defer ws.Close()

		ws.Write(t, NewSocketMessage(JoinAction, ChannelRequest{"foo"}))
		<-ws.MessageStash //discard the `channel_joined` from server

		ws.Write(t, NewSocketMessage(SendMessageAction, MessageEvent{"hi!", "foo"}))

		received := <-ws.MessageStash
		want := "new_message"
		if received.Action != want {
			t.Errorf("unexpected action returned from server, got %s, should be %s", received.Action, want)
		}

	})

	t.Run("two users create the channel at the same time", func(t *testing.T) {
		conns := []*TestWS{
			NewTestWS(t, wsURL),
			NewTestWS(t, wsURL)}
		for _, conn := range conns {
			go func(ws *TestWS) {
				ws.Write(t, NewSocketMessage(CreateAction, ChannelRequest{"foo"}))
				ws.Close()
			}(conn)
		}

	})
}
