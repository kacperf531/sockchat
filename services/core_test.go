package services_test

import (
	"context"
	"testing"

	"github.com/kacperf531/sockchat"
	"github.com/kacperf531/sockchat/api"
	"github.com/kacperf531/sockchat/services"
	"github.com/kacperf531/sockchat/test_utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSockChatCoreService(t *testing.T) {
	sampleMessage := api.MessageEvent{Text: "foo", Channel: "bar", Author: "baz"}
	messageStore := &test_utils.StubMessageStore{Messages: api.ChannelHistory{&sampleMessage}}
	userProfiles := &sockchat.ProfileService{Store: &test_utils.UserStoreDouble{}, Cache: test_utils.TestingRedisClient}

	oriDescription := test_utils.ValidUserDescription
	updatedDescription := "D3scription"

	core := &services.SockchatCoreService{UserProfiles: userProfiles, ChatChannels: &test_utils.StubChannelStore{}, Messages: messageStore}
	ctx := context.Background()

	t.Run("can register a new profile", func(t *testing.T) {
		_, err := core.RegisterProfile(&api.CreateProfileRequest{Nick: "Foo", Password: "Bar420", Description: oriDescription}, ctx)
		assert.NoError(t, err)
	})

	t.Run("can get a profile", func(t *testing.T) {
		profile, err := core.GetProfile(&api.GetProfileRequest{Nick: test_utils.ValidUserNick}, ctx)
		require.NoError(t, err)
		assert.Equal(t, test_utils.ValidUserNick, profile.Nick)
		assert.Equal(t, oriDescription, profile.Description)
	})

	t.Run("can edit a profile", func(t *testing.T) {
		_, err := core.EditProfile(&services.EditProfileWrapper{Nick: test_utils.ValidUserNick, Request: &api.EditProfileRequest{Description: updatedDescription}}, ctx)
		require.NoError(t, err)
	})

	t.Run("can get a profile after edit", func(t *testing.T) {
		profile, err := core.GetProfile(&api.GetProfileRequest{Nick: test_utils.ValidUserNick}, ctx)
		require.NoError(t, err)
		assert.Equal(t, updatedDescription, profile.Description)
	})

	t.Run("can get messages history of a channel", func(t *testing.T) {
		history, err := core.GetChannelHistory(&api.GetChannelHistoryRequest{Channel: test_utils.ChannelWithUser}, ctx)
		require.NoError(t, err)
		assert.Equal(t, api.ChannelHistory{&sampleMessage}, history)
	})

	t.Run("can filter messages history of a channel", func(t *testing.T) {
		history, err := core.GetChannelHistory(&api.GetChannelHistoryRequest{Channel: test_utils.ChannelWithUser, Search: "qux"}, ctx)
		require.NoError(t, err)
		assert.NotEqual(t, api.ChannelHistory{&sampleMessage}, history)
	})

	t.Run("can not register a new user with missing required data", func(t *testing.T) {
		missingDataTests := []*api.CreateProfileRequest{{Nick: "Foo"},
			{Password: "Bar42"}}
		for _, tt := range missingDataTests {
			_, err := core.RegisterProfile(tt, ctx)
			assert.Error(t, err)
		}
	})

	t.Run("can not register a new user with already existing nick", func(t *testing.T) {
		_, err := core.RegisterProfile(&api.CreateProfileRequest{Nick: "already_exists", Password: "Bar420"}, ctx)
		assert.Error(t, err)
	})

	t.Run("can not get a profile of non existing user", func(t *testing.T) {
		_, err := core.GetProfile(&api.GetProfileRequest{Nick: "not_exists"}, ctx)
		assert.Error(t, err)
	})

	t.Run("can not get history of non existing channel", func(t *testing.T) {
		_, err := core.GetChannelHistory(&api.GetChannelHistoryRequest{Channel: "not_exists"}, ctx)
		assert.Error(t, err)
	})
}
