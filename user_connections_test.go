package sockchat

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestUserManager(t *testing.T) {

	messageStore := &StubMessageStore{}
	store, _ := NewChannelStore(messageStore)
	userManager := NewConnectedUsersPool(store)

	t.Run("Resources (handlers) are cleaned up when user with 1 connection disconnects", func(t *testing.T) {
		dummyConn := &SockChatWS{}
		userManager.AddConnection(dummyConn, "dummy")

		userManager.RemoveConnection(dummyConn)
		_, handlerExists := userManager.GetHandler("dummy")
		assert.False(t, handlerExists)
	})

	t.Run("Connection remains when another user's connection closes", func(t *testing.T) {
		dummyConn := &SockChatWS{}
		otherDummyConn := &SockChatWS{}
		userManager.AddConnection(dummyConn, "dummy")
		userManager.AddConnection(otherDummyConn, "dummy")

		userManager.RemoveConnection(otherDummyConn)
		_, handlerExists := userManager.GetHandler("dummy")
		assert.True(t, handlerExists)
	})
}
