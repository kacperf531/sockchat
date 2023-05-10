package sockchat

import (
	"context"
	"testing"

	"github.com/kacperf531/sockchat/api"
	"github.com/kacperf531/sockchat/test_utils"
	"github.com/stretchr/testify/assert"
)

func TestUserProfile(t *testing.T) {

	store := test_utils.UserStoreDouble{}
	service := ProfileService{&store, test_utils.TestingRedisClient}

	t.Run("Calls to insert new user when request is OK", func(t *testing.T) {
		newUser := &api.CreateProfileRequest{Nick: "x69", Password: "foo420", Description: "description goes here"}
		err := service.Create(context.TODO(), newUser)
		assert.NoError(t, err)
		assert.Equal(t, store.CreateCalls[0].Nick, newUser.Nick)
		assert.Equal(t, store.CreateCalls[0].Description, newUser.Description)
		// ensure password is hashed
		assert.NotEqual(t, store.CreateCalls[0].PwHash, newUser.Password)
	})

	t.Run("Returns error on already existing nick", func(t *testing.T) {
		duplicateNickUser := &api.CreateProfileRequest{Nick: "already_exists", Password: "foo420", Description: "description goes here"}
		err := service.Create(context.TODO(), duplicateNickUser)
		assert.EqualError(t, err, api.ErrNickAlreadyUsed.Error())
	})

	t.Run("Returns error on empty nick/password", func(t *testing.T) {
		missingDataTests := []api.CreateProfileRequest{{Nick: "Foo"},
			{Password: "Bar42"}}
		for _, tt := range missingDataTests {
			assert.Error(t, service.Create(context.TODO(), &tt))
		}
	})

	t.Run("Calls to update existing user when edit request is OK", func(t *testing.T) {
		req := &api.EditProfileRequest{Description: "Bar"}
		err := service.Edit(context.TODO(), "dummy", req)
		assert.NoError(t, err)
		assert.Equal(t, req.Description, store.UpdateCalls[0].Description)
	})

	t.Run("GetProfile returns error for non-existing user", func(t *testing.T) {
		_, err := service.GetProfile(context.TODO(), "NonExistingUser")
		assert.Error(t, err)
	})

}
