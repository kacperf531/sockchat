package sockchat

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestChannelStore(t *testing.T) {

	dummyUser := UserHandler{}
	store, _ := NewChannelStore()

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

	t.Run("can not create channel without name", func(t *testing.T) {
		err := store.CreateChannel("")
		assert.EqualError(t, err, ErrEmptyChannelName.Error())
	})

	t.Run("can not join channel without providing name", func(t *testing.T) {
		err := store.AddUserToChannel("", &dummyUser)
		assert.EqualError(t, err, ErrEmptyChannelName.Error())
	})

	t.Run("can not leave channel without providing name", func(t *testing.T) {
		err := store.RemoveUserFromChannel("", &dummyUser)
		assert.EqualError(t, err, ErrEmptyChannelName.Error())
	})

	t.Run("can add user to channel", func(t *testing.T) {
		store.CreateChannel("Bar")
		store.AddUserToChannel("Bar", &dummyUser)

		assert.True(t, ChannelHasMember(store, "Bar", &dummyUser))
	})

	t.Run("can remove user from a channel", func(t *testing.T) {
		store.CreateChannel("Baz")
		store.AddUserToChannel("Baz", &dummyUser)
		store.RemoveUserFromChannel("Baz", &dummyUser)

		assert.True(t, ChannelHasMember(store, "Bar", &dummyUser))
	})

}

func AssertChannelExists(t *testing.T, store *ChannelStore, channel string) {
	t.Helper()
	_, err := store.GetChannel(channel)
	if err != nil {
		t.Errorf("channel %s does not exist", channel)
	}
}

func ChannelHasMember(store *ChannelStore, channelName string, member *UserHandler) bool {
	channel, _ := store.GetChannel(channelName)
	_, exists := channel.members[member]
	return exists
}
