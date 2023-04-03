package sockchat

import (
	"context"
	"fmt"
	"testing"

	"github.com/kacperf531/sockchat/errors"
	"github.com/kacperf531/sockchat/storage"
	"github.com/stretchr/testify/assert"
)

// Test double which spies create/update calls and stubs select request
type userStoreDouble struct {
	createCalls []*storage.User
	updateCalls []*storage.User
}

func (s *userStoreDouble) InsertUser(ctx context.Context, u *storage.User) error {
	s.createCalls = append(s.createCalls, u)
	if u.Nick == "already_exists" {
		return errors.ResourceConflict
	}
	return nil
}

func (s *userStoreDouble) UpdateUser(ctx context.Context, u *storage.User) error {
	s.updateCalls = append(s.updateCalls, u)
	return nil
}

func (s *userStoreDouble) SelectUser(ctx context.Context, nick string) (*storage.User, error) {
	if nick == validUserNick {
		return &storage.User{Nick: validUserNick, PwHash: validUserPasswordHash, Description: "desc"}, nil
	} else {
		return nil, fmt.Errorf("user not found")
	}
}

func TestCreateUser(t *testing.T) {

	store := userStoreDouble{}
	service := UserService{&store}

	t.Run("Calls to insert new user when request is OK", func(t *testing.T) {
		newUser := &UserRequest{Nick: "x69", Password: "foo420", Description: "description goes here"}
		err := service.CreateUser(context.TODO(), newUser)
		assert.NoError(t, err)
		assert.Equal(t, store.createCalls[0].Nick, newUser.Nick)
		assert.Equal(t, store.createCalls[0].Description, newUser.Description)
		// ensure password is hashed
		assert.NotEqual(t, store.createCalls[0].PwHash, newUser.Password)
	})

	t.Run("Returns error on already existing nick", func(t *testing.T) {
		duplicateNickUser := &UserRequest{Nick: "already_exists", Password: "foo420", Description: "description goes here"}
		err := service.CreateUser(context.TODO(), duplicateNickUser)
		assert.EqualError(t, err, errors.ResourceConflict.Error())
	})

	t.Run("Calls to update existing user when edit request is OK", func(t *testing.T) {
		req := &UserRequest{Nick: validUserNick, Description: "Bar", Password: validUserPassword}
		err := service.EditUser(context.TODO(), req)
		assert.NoError(t, err)
		assert.Equal(t, req.Nick, store.updateCalls[0].Nick)
		assert.Equal(t, req.Description, store.updateCalls[0].Description)
	})

	t.Run("Returns error on invalid password in edit request", func(t *testing.T) {
		initialCallsCount := len(store.updateCalls)
		req := &UserRequest{Nick: "Foo", Description: "this should not be set", Password: "boo420"}
		err := service.EditUser(context.TODO(), req)
		assert.Error(t, err)
		// count of calls to update should not increment
		assert.Equal(t, initialCallsCount, len(store.updateCalls))
	})
}
