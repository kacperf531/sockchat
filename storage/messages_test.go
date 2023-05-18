package storage

import (
	"context"
	"testing"
	"time"

	"github.com/elastic/go-elasticsearch/v7"
	"github.com/joho/godotenv"
	"github.com/kacperf531/sockchat/api"
	"github.com/stretchr/testify/require"
)

func TestMessageStore(t *testing.T) {
	godotenv.Load("../.env")

	es := mustSetUpES(t)
	store := &MessageStore{es, "test_messages"}

	t.Run("can index new message into ES", func(t *testing.T) {
		_, err := store.IndexMessage(&api.MessageEvent{Channel: "Foo", Author: "Bar", Text: "FooBarBaz", Timestamp: time.Now().Unix()})
		require.NoError(t, err)
	})

	t.Run("can get messages by channel", func(t *testing.T) {
		messages, err := store.FindMessages(context.Background(), "Foo", "")
		require.NoError(t, err)
		require.NotEmpty(t, messages)
	})

	t.Run("can search messages in channel by phrase", func(t *testing.T) {
		// positive case
		messages, err := store.FindMessages(context.Background(), "Foo", "FooBarBax")
		require.NoError(t, err)
		require.NotEmpty(t, messages)

		// negative case
		messages, err = store.FindMessages(context.Background(), "Foo", "FooBarQux")
		require.NoError(t, err)
		require.Empty(t, messages)
	})
}

func mustSetUpES(t *testing.T) *elasticsearch.Client {
	es, err := elasticsearch.NewDefaultClient()
	if err != nil {
		t.Errorf("error creating the client: %s", err)
	}
	return es
}
