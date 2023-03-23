package sockchat

import (
	"testing"
)

func TestSockChatStore(t *testing.T) {

	dummyUser := SockChatWS{}
	store := &SockChatStore{map[string]*Channel{"Foo420": {Users: []*SockChatWS{&dummyUser}}}}

	t.Run("can get an existing channel", func(t *testing.T) {
		_, err := store.GetChannel("Foo420")
		if err != nil {
			t.Errorf("unexpected issue while getting channel %v", err)
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
		err := store.CreateChannel("Foo420")
		if err == nil {
			t.Errorf("error should be returned but it was not")
		}
	})

	t.Run("Return correct value of user presence", func(t *testing.T) {
		got := store.ChannelHasUser("Foo420", &dummyUser)
		want := true
		if got != want {
			t.Error("User should be present in requested channel")
		}
	})

	t.Run("Return correct value of user presence for nonexistent channel", func(t *testing.T) {
		got := store.ChannelHasUser("noexistent", &dummyUser)
		want := false
		if got != want {
			t.Error("User should be present in requested channel")
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
