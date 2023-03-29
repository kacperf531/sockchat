package sockchat

import (
	"context"
	"testing"

	"github.com/kacperf531/sockchat/storage"
	"github.com/stretchr/testify/assert"
)

type userStoreSpy struct {
	calls []*storage.User
}

func (s *userStoreSpy) InsertUser(ctx context.Context, u *storage.User) error {
	s.calls = append(s.calls, u)
	return nil
}

func TestCreateUser(t *testing.T) {

	store := userStoreSpy{}
	service := UserService{&store}

	t.Run("Calls to insert new user when request is OK", func(t *testing.T) {
		newUser := NewUser{Nick: "Foo", Password: "Bar"}
		err := service.CreateUser(context.TODO(), &newUser)
		assert.NoError(t, err)
		assert.Equal(t, store.calls[0].Nick, newUser.Nick)
		// ensure password is hashed
		assert.NotEqual(t, store.calls[0].PwHash, newUser.Password)
	})

	// TODO
	// t.Run("Returns error on already existing nick", func(t *testing.T) {

	// })

}
