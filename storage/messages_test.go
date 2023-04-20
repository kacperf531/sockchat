package storage

import (
	"log"
	"testing"

	"github.com/elastic/go-elasticsearch/v7"
	"github.com/joho/godotenv"
	"github.com/kacperf531/sockchat/common"
	"github.com/stretchr/testify/require"
)

func TestMessageStore(t *testing.T) {
	godotenv.Load("../.env")

	es, err := elasticsearch.NewDefaultClient()
	if err != nil {
		log.Fatalf("Error creating the client: %s", err)
	}

	store := &MessageStore{es, "test_messages"}

	t.Run("can index new message into ES", func(t *testing.T) {
		_, err := store.IndexMessage(&common.MessageEvent{Channel: "Foo", Author: "Bar", Text: "Baz"})
		require.NoError(t, err)
	})

	t.Run("can get a messages by channel", func(t *testing.T) {
		messages, err := store.GetMessagesByChannel("Foo")
		require.NoError(t, err)
		require.NotEmpty(t, messages)
	})
}
