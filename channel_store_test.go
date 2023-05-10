package sockchat

import (
	"context"
	"testing"
	"time"

	"github.com/kacperf531/sockchat/api"
	"github.com/kacperf531/sockchat/test_utils"

	"github.com/stretchr/testify/assert"
)

func TestChannelStore(t *testing.T) {

	dummyUser := UserHandler{}
	messageStore := &test_utils.StubMessageStore{}
	store := NewChannelStore(messageStore)

	t.Run("returns error on nonexistent channel", func(t *testing.T) {
		_, err := store.getChannel("Foo420")
		if err == nil {
			t.Error("error should be returned on nonexistent channel, got nil")
		}
	})

	t.Run("returns error on empty channel", func(t *testing.T) {
		_, err := store.getChannel("")
		if err == nil {
			t.Error("error should be returned on empty channel, got nil")
		}
	})

	t.Run("can create a new channel", func(t *testing.T) {
		err := store.CreateChannel("Foo")
		if err != nil {
			t.Errorf("unexpected issue with creating channel %v", err)
		}
		_, err = store.getChannel("Foo")
		assert.NoError(t, err)
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
		assert.EqualError(t, err, api.ErrEmptyChannelName.Error())
	})

	t.Run("can not join channel without providing name", func(t *testing.T) {
		err := store.AddUserToChannel("", &dummyUser)
		assert.EqualError(t, err, api.ErrEmptyChannelName.Error())
	})

	t.Run("can not leave channel without providing name", func(t *testing.T) {
		err := store.RemoveUserFromChannel("", &dummyUser)
		assert.EqualError(t, err, api.ErrEmptyChannelName.Error())
	})

	t.Run("can add user to channel", func(t *testing.T) {
		store.CreateChannel("Bar")
		store.AddUserToChannel("Bar", &dummyUser)

		assert.True(t, store.IsUserPresentIn(&dummyUser, "Bar"))
	})

	t.Run("can remove user from a channel", func(t *testing.T) {
		store.CreateChannel("Baz")
		store.AddUserToChannel("Baz", &dummyUser)
		store.RemoveUserFromChannel("Baz", &dummyUser)

		assert.True(t, store.IsUserPresentIn(&dummyUser, "Bar"))
	})

	t.Run("Channel stores messages from users", func(t *testing.T) {
		store.CreateChannel("Qux")
		store.MessageChannel(&api.MessageEvent{Channel: "Qux", Author: "Foo", Text: "Bar", Timestamp: 0})
		ctx := context.Background()

		messageFound := make(chan bool, 1)
		go func() {
			for {
				msgs, _ := messageStore.FindMessages(ctx, "Qux", "")
				if len(msgs) == 1 {
					messageFound <- true
					return
				}
			}
		}()
		select {
		case <-messageFound:
			return
		case <-time.After(200 * time.Millisecond):
			t.Error("message was not stored in channel")
		}
	})

}
