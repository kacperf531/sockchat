package sockchat

import (
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestAuthorizedUserFlow(t *testing.T) {
	messageStore := &messageStoreSpy{}
	channelStore, _ := NewChannelStore(messageStore)
	channelStore.CreateChannel("foo")
	users := &userStoreDouble{}

	server := httptest.NewServer(NewSockChatServer(channelStore, users, messageStore))
	defer server.Close()

	wsURL := GetWsURL(server.URL)

	t.Run("user can send a message to a channel they have joined", func(t *testing.T) {
		ws := NewTestWS(t, wsURL)
		defer ws.Close()
		ws.Write(t, NewSocketMessage(LoginAction, LoginRequest{Nick: ValidUserNick, Password: ValidUserPassword}))
		ws.Write(t, NewSocketMessage(JoinAction, ChannelRequest{"foo"}))
		ws.Write(t, NewSocketMessage(SendMessageAction, SendMessageRequest{Channel: "foo", Text: "hi!"}))

		ws.AssertEventReceivedWithin(t, NewMessageEvent, 2*time.Second)
	})
}

func TestMultipleConnectionsSync(t *testing.T) {
	messageStore := &messageStoreSpy{}
	channelStore, _ := NewChannelStore(messageStore)
	channelStore.CreateChannel("foo")
	users := &userStoreDouble{}

	server := httptest.NewServer(NewSockChatServer(channelStore, users, messageStore))
	wsURL := GetWsURL(server.URL)

	conns := make([]*TestWS, 2)
	for i := 0; i < 2; i++ {
		conns[i] = NewTestWS(t, wsURL)
		defer conns[i].Close()
	}
	defer server.Close()

	t.Run("User can log in using two connections at the same time", func(t *testing.T) {
		logged_in_conns := make(chan *TestWS)
		for _, conn := range conns {
			go func(ws *TestWS) {
				ws.Write(t, NewSocketMessage(LoginAction, LoginRequest{Nick: ValidUserNick, Password: ValidUserPassword}))
				ws.AssertEventReceivedWithin(t, "logged_in:"+ValidUserNick, 2*time.Second)
				logged_in_conns <- ws
			}(conn)
			<-logged_in_conns
		}

	})

	t.Run("Both connections receive information about joining the channel when one of the connections sent request to join", func(t *testing.T) {
		conns[0].Write(t, NewSocketMessage(JoinAction, ChannelRequest{"foo"}))
		for _, conn := range conns {
			conn.AssertEventReceivedWithin(t, UserJoinedChannelEvent, 2*time.Second)
		}
	})

	t.Run("Second user's conn receives event about new message in the channel they're in", func(t *testing.T) {
		msgText := "Baz"
		conns[1].Write(t, NewSocketMessage(SendMessageAction, SendMessageRequest{Channel: "foo", Text: msgText}))
		select {
		case message_received := <-conns[0].MessageStash:
			assert.Equal(t, msgText, UnmarshalMessageEvent(message_received.Payload).Text)
			return
		case <-time.After(100 * time.Millisecond):
			t.Error("Message not received by second connection")
		}
	})

	t.Run("Both connections get notified when user leaves the channel", func(t *testing.T) {
		conns[0].Write(t, NewSocketMessage(LeaveAction, ChannelRequest{"foo"}))
		for _, conn := range conns {
			conn.AssertEventReceivedWithin(t, YouLeftChannelEvent, 2*time.Second)
		}
	})

	t.Run("After leaving the channel, no connection can send messages to it", func(t *testing.T) {
		conns[1].Write(t, NewSocketMessage(SendMessageAction, SendMessageRequest{Channel: "foo", Text: "Baz"}))
		conns[1].AssertEventReceivedWithin(t, InvalidRequestEvent, 2*time.Second)
	})
}

func TestMultipleUsersSync(t *testing.T) {

	messageStore := &messageStoreSpy{}
	channelStore, _ := NewChannelStore(messageStore)
	channelStore.CreateChannel("foo")
	users := &userStoreDouble{}
	server := httptest.NewServer(NewSockChatServer(channelStore, users, messageStore))
	wsURL := GetWsURL(server.URL)

	conns := make([]*TestWS, 2)
	for i := 0; i < 2; i++ {
		conns[i] = NewTestWS(t, wsURL)
		defer conns[i].Close()
	}
	defer server.Close()

	t.Run("Two users can log in using two connections at the same time", func(t *testing.T) {
		done := make(chan bool, 2)
		go func() {
			conns[0].Write(t, NewSocketMessage(LoginAction, LoginRequest{Nick: ValidUserNick, Password: ValidUserPassword}))
			conns[0].AssertEventReceivedWithin(t, "logged_in:"+ValidUserNick, 2*time.Second)
			done <- true
		}()
		go func() {
			conns[1].Write(t, NewSocketMessage(LoginAction, LoginRequest{Nick: ValidUser2Nick, Password: ValidUserPassword}))
			conns[1].AssertEventReceivedWithin(t, "logged_in:"+ValidUser2Nick, 2*time.Second)
			done <- true
		}()
		for i := 0; i < 2; i++ {
			<-done
		}
	})

	t.Run("Two users can join the same channel", func(t *testing.T) {
		go conns[0].Write(t, NewSocketMessage(JoinAction, ChannelRequest{"foo"}))
		go conns[1].Write(t, NewSocketMessage(JoinAction, ChannelRequest{"foo"}))
		for _, conn := range conns {
			conn.AssertEventReceivedWithin(t, UserJoinedChannelEvent, 2*time.Second)
		}
	})

	t.Run("One of the users disconnects - check if the other user gets notified", func(t *testing.T) {
		conns[0].Close()
		conns[1].AssertEventReceivedWithin(t, UserLeftChannelEvent, 2*time.Second)
	})

	t.Run("Other user can send messages to the channel", func(t *testing.T) {
		conns[1].Write(t, NewSocketMessage(SendMessageAction, SendMessageRequest{Channel: "foo", Text: "Baz"}))
		conns[1].AssertEventReceivedWithin(t, NewMessageEvent, 2*time.Second)
	})

}
