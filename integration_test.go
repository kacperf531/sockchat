package sockchat

import (
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

		ws.Write(t, NewSocketMessage(SendMessageAction, SendMessageRequest{Channel: "foo", Text: "hi!"}))

		received := <-ws.MessageStash
		wantAction := "new_message"
		require.Equal(t, wantAction, received.Action, "action should be %s", wantAction)
		message := UnmarshalMessageEvent(received.Payload)
		wantAuthor := ValidUserNick
		assert.Equal(t, wantAuthor, message.Author, "author should be %s", wantAuthor)

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
