package services_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/kacperf531/sockchat"
	"github.com/kacperf531/sockchat/api"
	"github.com/kacperf531/sockchat/services"
	"github.com/kacperf531/sockchat/test_utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSockChatWS(t *testing.T) {
	testTimeoutUnauthorized := 200 * time.Millisecond
	testTimeoutAuthorized := 20 * testTimeoutUnauthorized
	userProfiles := &sockchat.ProfileService{Store: &test_utils.UserStoreDouble{}, Cache: test_utils.TestingRedisClient}

	router := http.NewServeMux()
	channelStore := &test_utils.StubChannelStore{}
	messagingAPI := &services.MessagingAPI{TimeoutAuthorized: testTimeoutAuthorized, TimeoutUnauthorized: testTimeoutUnauthorized, ConnectedUsers: sockchat.NewConnectedUsersPool(channelStore), UserProfiles: userProfiles}

	messagingAPI.HandleRequests(router)
	testServer := httptest.NewServer(router)
	wsURL := test_utils.GetWsURL(testServer.URL)
	ws := test_utils.NewTestWS(t, wsURL)

	defer ws.Close()
	defer testServer.Close()

	// login the user first
	request := api.NewSocketMessage(api.LoginAction, api.LoginRequest{Nick: test_utils.ValidUserNick, Password: test_utils.ValidUserPassword})
	ws.Write(t, request)

	received := <-ws.MessageStash
	want := fmt.Sprintf("logged_in:%s", test_utils.ValidUserNick)
	require.Equal(t, want, received.Action)

	t.Run("creates channel on request", func(t *testing.T) {
		channelName := "FooBar420"
		request := api.NewSocketMessage(api.CreateAction, api.ChannelRequest{Name: channelName})
		ws.Write(t, request)

		received := <-ws.MessageStash
		// User joins channel on creation
		want := api.UserJoinedChannelEvent
		require.Equal(t, want, received.Action)
	})

	t.Run("returns error on creating channel with existing name", func(t *testing.T) {
		request := api.NewSocketMessage(api.CreateAction, api.ChannelRequest{Name: "already_exists"})
		ws.Write(t, request)

		received := <-ws.MessageStash
		want := api.ErrInvalidRequest.Error()
		assert.Equal(t, want, received.Action)
	})

	t.Run("can join a channel", func(t *testing.T) {
		request := api.NewSocketMessage(api.JoinAction, api.ChannelRequest{Name: test_utils.ChannelWithoutUser})
		ws.Write(t, request)

		received := <-ws.MessageStash
		want := api.UserJoinedChannelEvent
		require.Equal(t, want, received.Action)
		details, err := api.UnmarshalChannelUserChangeEvent(received.Payload)
		assert.NoError(t, err)
		assert.Equal(t, test_utils.ChannelWithoutUser, details.Channel)
		assert.Equal(t, test_utils.ValidUserNick, details.Nick)
	})

	t.Run("can leave a channel", func(t *testing.T) {
		request := api.NewSocketMessage(api.LeaveAction, api.ChannelRequest{Name: test_utils.ChannelWithUser})
		ws.Write(t, request)

		received := <-ws.MessageStash
		want := api.UserLeftChannelEvent
		require.Equal(t, received.Action, want)
		details, err := api.UnmarshalChannelUserChangeEvent(received.Payload)
		assert.NoError(t, err)
		assert.Equal(t, details.Channel, test_utils.ChannelWithUser)
		assert.Equal(t, details.Nick, test_utils.ValidUserNick)
	})

	t.Run("error if leaving a channel user are not in", func(t *testing.T) {
		request := api.NewSocketMessage(api.LeaveAction, api.ChannelRequest{Name: test_utils.ChannelWithoutUser})
		ws.Write(t, request)

		ws.AssertEventReceivedWithin(t, api.ErrInvalidRequest.Error(), 200*time.Millisecond)
	})

	t.Run("can not join a channel they are already in", func(t *testing.T) {
		request := api.NewSocketMessage(api.JoinAction, api.ChannelRequest{Name: test_utils.ChannelWithUser})
		ws.Write(t, request)

		ws.AssertEventReceivedWithin(t, api.ErrInvalidRequest.Error(), 200*time.Millisecond)
	})

	t.Run("can not send a message to a channel being outside of", func(t *testing.T) {
		request := api.NewSocketMessage(api.SendMessageAction, api.SendMessageRequest{Channel: "foo", Text: test_utils.ChannelWithoutUser})
		ws.Write(t, request)

		ws.AssertEventReceivedWithin(t, api.ErrInvalidRequest.Error(), 200*time.Millisecond)

	})

	t.Run("unauthorized connection times out", func(t *testing.T) {
		new_ws := test_utils.NewTestWS(t, wsURL)
		new_ws.AssertEventReceivedWithin(t, "connection_timed_out", testTimeoutUnauthorized+20*time.Millisecond)
	})

	t.Run("connection timeout period is extended after logging in", func(t *testing.T) {
		new_ws := test_utils.NewTestWS(t, wsURL)
		request := api.NewSocketMessage(api.LoginAction, api.LoginRequest{Nick: test_utils.ValidUserNick, Password: test_utils.ValidUserPassword})
		new_ws.Write(t, request)
		<-new_ws.MessageStash // read `login` response
		select {
		case received := <-new_ws.MessageStash:
			if received.Action == "connection_timed_out" {
				t.Error("connection timed out", received)
			} else {
				t.Error("unexpected message received", received)
			}
		case <-time.After(testTimeoutUnauthorized + 20*time.Millisecond):
			return
		}
	})

}
