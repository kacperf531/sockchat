package sockchat

import (
	"context"
	"testing"

	"github.com/kacperf531/sockchat/storage"
	"github.com/stretchr/testify/assert"
)

type userStoreSpy struct {
	createCalls []*storage.User
	updateCalls []*storage.User
}

func (s *userStoreSpy) InsertUser(ctx context.Context, u *storage.User) error {
	s.createCalls = append(s.createCalls, u)
	return nil
}

func (s *userStoreSpy) UpdateUser(ctx context.Context, u *storage.User) error {
	s.updateCalls = append(s.updateCalls, u)
	return nil
}

func TestCreateUser(t *testing.T) {

	store := userStoreSpy{}
	service := UserService{&store}

	t.Run("Calls to insert new user when request is OK", func(t *testing.T) {
		newUser := &UserRequest{Nick: "Foo", Password: "Bar", Description: "description goes here"}
		err := service.CreateUser(context.TODO(), newUser)
		assert.NoError(t, err)
		assert.Equal(t, store.createCalls[0].Nick, newUser.Nick)
		assert.Equal(t, store.createCalls[0].Description, newUser.Description)
		// ensure password is hashed
		assert.NotEqual(t, store.createCalls[0].PwHash, newUser.Password)
	})

	// TODO
	// t.Run("Returns error on already existing nick", func(t *testing.T) {

	// })

	t.Run("Calls to update existing user when edit request is OK", func(t *testing.T) {
		updatedUser := &UserRequest{Nick: "Foo", Description: "Bar"}
		err := service.EditUser(context.TODO(), updatedUser)
		assert.NoError(t, err)
		assert.Equal(t, updatedUser.Nick, store.updateCalls[0].Nick)
		assert.Equal(t, updatedUser.Description, store.updateCalls[0].Description)
	})

	// TODO
	// t.Run("Returns error on invalid authorization in edit request", func(t *testing.T) {

	// })

}
