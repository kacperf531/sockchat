package sockchat

import (
	"testing"
)

func TestSockChatStore(t *testing.T) {

	dummyUser := SockChatWS{}
	store, _ := NewSockChatStore()

	t.Run("returns error on nonexistent channel", func(t *testing.T) {
		_, err := store.GetChannel("Foo420")
		if err == nil {
			t.Error("error should be returned on nonexistent channel, got nil")
		}
	})

	t.Run("can create a new channel", func(t *testing.T) {
		err := store.CreateChannel("Foo")
		if err != nil {
			t.Errorf("unexpected issue with creating channel %v", err)
		}
		AssertChannelExists(t, store, "Foo")
	})

	t.Run("can not create channel with existing name", func(t *testing.T) {
		store.CreateChannel("Foo420") // create channel first
		err := store.CreateChannel("Foo420")
		if err == nil {
			t.Errorf("error should be returned but it was not")
		}
	})

	t.Run("can add user to channel", func(t *testing.T) {
		store.CreateChannel("Bar")
		store.AddUserToChannel("Bar", &dummyUser)

		got := store.ChannelHasUser("Bar", &dummyUser)
		want := true
		if got != want {
			t.Error("User should be present in requested channel")
		}
	})

	t.Run("can remove user from a channel", func(t *testing.T) {
		store.CreateChannel("Baz")
		store.AddUserToChannel("Baz", &dummyUser)
		store.RemoveUserFromChannel("Baz", &dummyUser)

		got := store.ChannelHasUser("Baz", &dummyUser)
		want := false
		if got != want {
			t.Error("User should not be present in requested channel (they were removed)")
		}
	})

	t.Run("Handle user's disconnection", func(t *testing.T) {
		store.CreateChannel("Baz")
		store.AddUserToChannel("Baz", &dummyUser)
		store.DisconnectUser(&dummyUser)

		got := store.ChannelHasUser("Bar", &dummyUser)
		want := false
		if got != want {
			t.Error("User should not be present in requested channel (they were disconnected)")
		}
	})

}

func AssertChannelExists(t *testing.T, store ChannelStore, channel string) {
	t.Helper()
	_, err := store.GetChannel(channel)
	if err != nil {
		t.Errorf("channel %s does not exist", channel)
	}
}
