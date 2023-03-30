package sockchat

import (
	"context"
	"fmt"
	"testing"

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
	return nil
}

func (s *userStoreDouble) UpdateUser(ctx context.Context, u *storage.User) error {
	s.updateCalls = append(s.updateCalls, u)
	return nil
}

func (s *userStoreDouble) SelectUser(ctx context.Context, nick string) (*storage.User, error) {
	// Password hash for `foo420` password
	PasswordHash := "$2a$10$Xl002E7Vj5qM1RHMiM06KOCHofpLcPTIj7LeyZgTf62txoOBvoyia"
	if nick == "Foo" {
		return &storage.User{Nick: "Foo", PwHash: PasswordHash, Description: "desc"}, nil
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
		duplicateNickUser := &UserRequest{Nick: "Foo", Password: "foo420", Description: "description goes here"}
		err := service.CreateUser(context.TODO(), duplicateNickUser)
		assert.Error(t, err)
	})

	t.Run("Calls to update existing user when edit request is OK", func(t *testing.T) {
		req := &UserRequest{Nick: "Foo", Description: "Bar", Password: "foo420"}
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
