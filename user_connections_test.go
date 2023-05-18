package sockchat

import (
	"testing"

	"github.com/kacperf531/sockchat/services"
	"github.com/kacperf531/sockchat/test_utils"
	"github.com/stretchr/testify/assert"
)

func TestUserManager(t *testing.T) {
	t.Parallel()

	messageStore := &test_utils.StubMessageStore{}
	store := NewChannelStore(messageStore)
	userManager := NewConnectedUsersPool(store)

	t.Run("Resources (handlers) are cleaned up when user with 1 connection disconnects", func(t *testing.T) {
		dummyConn := &services.SockChatWS{}
		userManager.AddConnection(dummyConn, "dummy")

		userManager.RemoveConnection(dummyConn)
		_, handlerExists := userManager.GetHandler("dummy")
		assert.False(t, handlerExists)
	})

	t.Run("Connection remains when another user's connection closes", func(t *testing.T) {
		dummyConn := &services.SockChatWS{}
		otherDummyConn := &services.SockChatWS{}
		userManager.AddConnection(dummyConn, "dummy")
		userManager.AddConnection(otherDummyConn, "dummy")

		userManager.RemoveConnection(otherDummyConn)
		_, handlerExists := userManager.GetHandler("dummy")
		assert.True(t, handlerExists)
	})
}
