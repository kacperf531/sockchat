package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/kacperf531/sockchat"
	"github.com/kacperf531/sockchat/api"
	"github.com/kacperf531/sockchat/services"
	"github.com/kacperf531/sockchat/test_utils"
	"github.com/stretchr/testify/assert"
)

func setUpTestServer(t *testing.T) *httptest.Server {
	t.Helper()

	messageStore := &test_utils.StubMessageStore{}
	channelStore := sockchat.NewChannelStore(messageStore)
	channelStore.CreateChannel("foo")
	userStore := &test_utils.UserStoreDouble{}
	userCache := test_utils.TestingRedisClient
	userProfileService := &sockchat.ProfileService{Store: userStore, Cache: userCache}
	connectedUsers := sockchat.NewConnectedUsersPool(channelStore)
	authService := &services.SockchatAuthService{UserProfiles: userProfileService}
	coreService := &services.SockchatCoreService{
		UserProfiles:   userProfileService,
		Messages:       messageStore,
		ChatChannels:   channelStore,
		ConnectedUsers: connectedUsers}

	httpRouter := http.NewServeMux()
	webAPI := services.NewWebAPI(coreService, authService)
	webAPI.HandleRequests(httpRouter)
	messagingAPI := &services.MessagingAPI{TimeoutAuthorized: defaultTimeoutAuthorized, TimeoutUnauthorized: defaultTimeoutUnauthorized, ConnectedUsers: connectedUsers, UserProfiles: userProfileService}
	messagingAPI.HandleRequests(httpRouter)

	return httptest.NewServer(httpRouter)
}

// Functional test of critical features of a single user case
// User profiles remain stubbed to avoid the need for a MySQL setup
func TestAuthorizedUserFlow(t *testing.T) {
	server := setUpTestServer(t)
	test_utils.TestingRedisClient.FlushAll(context.TODO())
	defer server.Close()

	wsURL := test_utils.GetWsURL(server.URL)
	ws := test_utils.NewTestWS(t, wsURL)
	defer ws.Close()

	t.Run("user can log in", func(t *testing.T) {
		ws.Write(t, api.NewSocketMessage(api.LoginAction, api.LoginRequest{Nick: test_utils.ValidUserNick, Password: test_utils.ValidUserPassword}))
		ws.AssertEventReceivedWithin(t, "logged_in:"+test_utils.ValidUserNick, 2*time.Second)
	})

	t.Run("user can create a channel and is automatically added to it", func(t *testing.T) {
		ws.Write(t, api.NewSocketMessage(api.CreateAction, api.ChannelRequest{Name: "bar"}))
		ws.AssertEventReceivedWithin(t, api.UserJoinedChannelEvent, 2*time.Second)
	})

	t.Run("user can not join a channel they are already in", func(t *testing.T) {
		ws.Write(t, api.NewSocketMessage(api.JoinAction, api.ChannelRequest{Name: "bar"}))
		ws.AssertEventReceivedWithin(t, api.ErrInvalidRequest.Error(), 2*time.Second)
	})

	t.Run("user can leave a channel and gets appropriate notification", func(t *testing.T) {
		ws.Write(t, api.NewSocketMessage(api.LeaveAction, api.ChannelRequest{Name: "bar"}))
		ws.AssertEventReceivedWithin(t, api.YouLeftChannelEvent, 2*time.Second)
	})

	t.Run("user can not message channel they are not in", func(t *testing.T) {
		ws.Write(t, api.NewSocketMessage(api.SendMessageAction, api.SendMessageRequest{Channel: "bar", Text: "Baz"}))
		ws.AssertEventReceivedWithin(t, api.ErrInvalidRequest.Error(), 2*time.Second)
	})

	t.Run("user can re-join a channel they left", func(t *testing.T) {
		ws.Write(t, api.NewSocketMessage(api.JoinAction, api.ChannelRequest{Name: "bar"}))
		ws.AssertEventReceivedWithin(t, api.UserJoinedChannelEvent, 2*time.Second)
	})

	t.Run("user can send a message to a channel they are in", func(t *testing.T) {
		ws.Write(t, api.NewSocketMessage(api.SendMessageAction, api.SendMessageRequest{Channel: "bar", Text: "Baz"}))
		ws.AssertEventReceivedWithin(t, api.NewMessageEvent, 2*time.Second)
	})

}

func TestMultipleConnectionsSync(t *testing.T) {
	server := setUpTestServer(t)
	test_utils.TestingRedisClient.FlushAll(context.TODO())
	defer server.Close()

	wsURL := test_utils.GetWsURL(server.URL)
	conns := make([]*test_utils.TestWS, 2)
	for i := 0; i < 2; i++ {
		conns[i] = test_utils.NewTestWS(t, wsURL)
		defer conns[i].Close()
	}

	t.Run("User can log in using two connections at the same time", func(t *testing.T) {
		logged_in_conns := make(chan *test_utils.TestWS)
		for _, conn := range conns {
			go func(ws *test_utils.TestWS) {
				ws.Write(t, api.NewSocketMessage(api.LoginAction, api.LoginRequest{Nick: test_utils.ValidUserNick, Password: test_utils.ValidUserPassword}))
				ws.AssertEventReceivedWithin(t, "logged_in:"+test_utils.ValidUserNick, 2*time.Second)
				logged_in_conns <- ws
			}(conn)
			<-logged_in_conns
		}

	})

	t.Run("Both connections receive information about joining the channel when one of the connections sent request to join", func(t *testing.T) {
		conns[0].Write(t, api.NewSocketMessage(api.JoinAction, api.ChannelRequest{Name: "foo"}))
		for _, conn := range conns {
			conn.AssertEventReceivedWithin(t, api.UserJoinedChannelEvent, 2*time.Second)
		}
	})

	t.Run("Second user's conn receives event about new message in the channel they're in", func(t *testing.T) {
		msgText := "Baz"
		conns[1].Write(t, api.NewSocketMessage(api.SendMessageAction, api.SendMessageRequest{Channel: "foo", Text: msgText}))
		select {
		case message_received := <-conns[0].MessageStash:
			message, err := api.UnmarshalMessageEvent(message_received.Payload)
			assert.NoError(t, err)
			assert.Equal(t, msgText, message.Text)
			return
		case <-time.After(100 * time.Millisecond):
			t.Error("Message not received by second connection")
		}
	})

	t.Run("Both connections get notified when user leaves the channel", func(t *testing.T) {
		conns[0].Write(t, api.NewSocketMessage(api.LeaveAction, api.ChannelRequest{Name: "foo"}))
		for _, conn := range conns {
			conn.AssertEventReceivedWithin(t, api.YouLeftChannelEvent, 2*time.Second)
		}
	})

	t.Run("After leaving the channel, no connection can send messages to it", func(t *testing.T) {
		conns[1].Write(t, api.NewSocketMessage(api.SendMessageAction, api.SendMessageRequest{Channel: "foo", Text: "Baz"}))
		conns[1].AssertEventReceivedWithin(t, api.ErrInvalidRequest.Error(), 2*time.Second)
	})
}

func TestMultipleUsersSync(t *testing.T) {

	server := setUpTestServer(t)
	test_utils.TestingRedisClient.FlushAll(context.TODO())
	defer server.Close()

	wsURL := test_utils.GetWsURL(server.URL)
	conns := make([]*test_utils.TestWS, 2)
	for i := 0; i < 2; i++ {
		conns[i] = test_utils.NewTestWS(t, wsURL)
		defer conns[i].Close()
	}

	t.Run("Two users can log in using two connections at the same time", func(t *testing.T) {
		done := make(chan bool, 2)
		go func() {
			conns[0].Write(t, api.NewSocketMessage(api.LoginAction, api.LoginRequest{Nick: test_utils.ValidUserNick, Password: test_utils.ValidUserPassword}))
			conns[0].AssertEventReceivedWithin(t, "logged_in:"+test_utils.ValidUserNick, 2*time.Second)
			done <- true
		}()
		go func() {
			conns[1].Write(t, api.NewSocketMessage(api.LoginAction, api.LoginRequest{Nick: test_utils.ValidUser2Nick, Password: test_utils.ValidUserPassword}))
			conns[1].AssertEventReceivedWithin(t, "logged_in:"+test_utils.ValidUser2Nick, 2*time.Second)
			done <- true
		}()
		for i := 0; i < 2; i++ {
			<-done
		}
	})

	t.Run("Two users can join the same channel", func(t *testing.T) {
		go conns[0].Write(t, api.NewSocketMessage(api.JoinAction, api.ChannelRequest{Name: "foo"}))
		go conns[1].Write(t, api.NewSocketMessage(api.JoinAction, api.ChannelRequest{Name: "foo"}))
		for _, conn := range conns {
			conn.AssertEventReceivedWithin(t, api.UserJoinedChannelEvent, 2*time.Second)
		}
	})

	t.Run("Two users can send message to the channel at the same time", func(t *testing.T) {
		done := make(chan bool, 2)
		go func() {
			conns[0].Write(t, api.NewSocketMessage(api.SendMessageAction, api.SendMessageRequest{Channel: "foo", Text: "Bar"}))
			conns[0].AssertEventReceivedWithin(t, api.NewMessageEvent, 2*time.Second)
			done <- true
		}()
		go func() {
			conns[1].Write(t, api.NewSocketMessage(api.SendMessageAction, api.SendMessageRequest{Channel: "foo", Text: "Baz"}))
			conns[1].AssertEventReceivedWithin(t, api.NewMessageEvent, 2*time.Second)
			done <- true
		}()
		for i := 0; i < 2; i++ {
			<-done
		}
	})

	t.Run("One of the users disconnects - check if the other user gets notified", func(t *testing.T) {
		conns[0].Close()
		conns[1].AssertEventReceivedWithin(t, api.UserLeftChannelEvent, 2*time.Second)
	})

	t.Run("Other user can send messages to the channel", func(t *testing.T) {
		conns[1].Write(t, api.NewSocketMessage(api.SendMessageAction, api.SendMessageRequest{Channel: "foo", Text: "Baz"}))
		conns[1].AssertEventReceivedWithin(t, api.NewMessageEvent, 2*time.Second)
	})

}
