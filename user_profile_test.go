package sockchat

import (
	"context"
	"fmt"
	"testing"

	"github.com/kacperf531/sockchat/common"
	"github.com/kacperf531/sockchat/storage"
	"github.com/stretchr/testify/assert"
)

// Test double which spies create/update calls and stubs select request
type userStoreDouble struct {
	createCalls []*storage.User
	updateCalls []*common.PublicProfile
}

func (s *userStoreDouble) InsertUser(ctx context.Context, u *storage.User) error {
	s.createCalls = append(s.createCalls, u)
	if u.Nick == "already_exists" {
		return common.ErrResourceConflict
	}
	return nil
}

func (s *userStoreDouble) UpdatePublicProfile(ctx context.Context, u *common.PublicProfile) error {
	s.updateCalls = append(s.updateCalls, u)

	return nil
}

func (s *userStoreDouble) SelectUser(ctx context.Context, nick string) (*storage.User, error) {
	var description string
	if len(s.updateCalls) > 0 {
		description = s.updateCalls[0].Description
	} else {
		description = ValidUserDescription
	}
	if nick == ValidUserNick {
		return &storage.User{Nick: ValidUserNick, PwHash: ValidUserPasswordHash, Description: description}, nil
	}
	if nick == ValidUser2Nick {
		return &storage.User{Nick: ValidUser2Nick, PwHash: ValidUserPasswordHash, Description: description}, nil
	}
	return nil, fmt.Errorf("user not found")

}

func TestUserProfile(t *testing.T) {

	store := userStoreDouble{}
	service := ProfileService{&store, TestingRedisClient}

	t.Run("Calls to insert new user when request is OK", func(t *testing.T) {
		newUser := &CreateProfileRequest{Nick: "x69", Password: "foo420", Description: "description goes here"}
		err := service.Create(context.TODO(), newUser)
		assert.NoError(t, err)
		assert.Equal(t, store.createCalls[0].Nick, newUser.Nick)
		assert.Equal(t, store.createCalls[0].Description, newUser.Description)
		// ensure password is hashed
		assert.NotEqual(t, store.createCalls[0].PwHash, newUser.Password)
	})

	t.Run("Returns error on already existing nick", func(t *testing.T) {
		duplicateNickUser := &CreateProfileRequest{Nick: "already_exists", Password: "foo420", Description: "description goes here"}
		err := service.Create(context.TODO(), duplicateNickUser)
		assert.EqualError(t, err, common.ErrResourceConflict.Error())
	})

	t.Run("Returns error on empty nick/password", func(t *testing.T) {
		missingDataTests := []CreateProfileRequest{{Nick: "Foo"},
			{Password: "Bar42"}}
		for _, tt := range missingDataTests {
			assert.Error(t, service.Create(context.TODO(), &tt))
		}
	})

	t.Run("Calls to update existing user when edit request is OK", func(t *testing.T) {
		req := &EditProfileRequest{Description: "Bar"}
		err := service.Edit(context.TODO(), "dummy", req)
		assert.NoError(t, err)
		assert.Equal(t, req.Description, store.updateCalls[0].Description)
	})

	t.Run("GetProfile returns error for non-existing user", func(t *testing.T) {
		_, err := service.GetProfile(context.TODO(), "NonExistingUser")
		assert.Error(t, err)
	})

}
