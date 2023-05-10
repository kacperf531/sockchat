package test_utils

import (
	"context"
	"sync"

	"github.com/kacperf531/sockchat/api"
	"github.com/kacperf531/sockchat/storage"
	"github.com/redis/go-redis/v9"
)

var TestingRedisClient = redis.NewClient(
	&redis.Options{
		Addr:     "localhost:6379",
		Password: "",
		DB:       1,
	},
)

// StubChannelStore implements ChannelStore for testing purposes
type StubChannelStore struct{}

func (store *StubChannelStore) CreateChannel(name string) error {
	if name == "already_exists" {
		return api.ErrChannelAlreadyExists
	}
	return nil
}

func (store *StubChannelStore) DisconnectUser(user api.SockchatUserHandler) {
}

func (store *StubChannelStore) ChannelExists(name string) bool {
	return name != "not_exists"
}

func (s *StubChannelStore) AddUserToChannel(name string, user api.SockchatUserHandler) error {
	if name == ChannelWithUser {
		return api.ErrUserAlreadyInChannel
	}
	user.Write(api.NewSocketMessage(api.UserJoinedChannelEvent, api.ChannelUserChangeEvent{Channel: name, Nick: user.GetNick()}))
	return nil
}

func (s *StubChannelStore) RemoveUserFromChannel(name string, user api.SockchatUserHandler) error {
	if name == ChannelWithoutUser {
		return api.ErrUserNotInChannel
	}
	user.Write(api.NewSocketMessage(api.UserLeftChannelEvent, api.ChannelUserChangeEvent{Channel: name, Nick: user.GetNick()}))
	return nil
}

func (s *StubChannelStore) IsUserPresentIn(user api.SockchatUserHandler, channel string) bool {
	return false
}

func (store *StubChannelStore) MessageChannel(message *api.MessageEvent) error {
	return nil
}

type StubMessageStore struct {
	Messages api.ChannelHistory
	lock     sync.Mutex
}

func (s *StubMessageStore) FindMessages(ctx context.Context, channel, soughtPhrase string) (api.ChannelHistory, error) {
	if soughtPhrase != "" {
		// just assume that the results are filtered out
		return nil, nil
	}
	s.lock.Lock()
	defer s.lock.Unlock()
	// simplified stub - channel filtering & sorting logic is in ES
	return s.Messages, nil
}

func (s *StubMessageStore) IndexMessage(*api.MessageEvent) (string, error) {
	s.lock.Lock()
	defer s.lock.Unlock()
	s.Messages = append(s.Messages, &api.MessageEvent{})
	return "", nil
}

// Test double which spies create/update calls and stubs select request
type UserStoreDouble struct {
	CreateCalls []*storage.User
	UpdateCalls []*api.PublicProfile
}

func (s *UserStoreDouble) InsertUser(ctx context.Context, u *storage.User) error {
	s.CreateCalls = append(s.CreateCalls, u)
	if u.Nick == "already_exists" {
		return api.ErrNickAlreadyUsed
	}
	return nil
}

func (s *UserStoreDouble) UpdatePublicProfile(ctx context.Context, u *api.PublicProfile) error {
	s.UpdateCalls = append(s.UpdateCalls, u)
	return nil
}

func (s *UserStoreDouble) SelectUser(ctx context.Context, nick string) (*storage.User, error) {
	var description string
	if len(s.UpdateCalls) > 0 {
		description = s.UpdateCalls[0].Description
	} else {
		description = ValidUserDescription
	}
	if nick == ValidUserNick {
		return &storage.User{Nick: ValidUserNick, PwHash: ValidUserPasswordHash, Description: description}, nil
	}
	if nick == ValidUser2Nick {
		return &storage.User{Nick: ValidUser2Nick, PwHash: ValidUserPasswordHash, Description: description}, nil
	}
	return nil, api.ErrUserNotFound

}
