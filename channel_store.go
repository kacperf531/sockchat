package sockchat

import (
	"errors"
	"fmt"
	"sync"
)

var ErrChannelDoesNotExist = errors.New("channel does not exist")
var ErrUserNotInChannel = errors.New("user is not member of this channel")
var ErrUserAlreadyInChannel = errors.New("user is already member of this channel")
var ErrEmptyChannelName = errors.New("channel's `name` is missing")

type ChannelStore struct {
	Channels map[string]*Channel
	lock     sync.RWMutex
}

func NewChannelStore() (*ChannelStore, error) {
	return &ChannelStore{Channels: make(map[string]*Channel)}, nil
}

func (s *ChannelStore) GetChannel(name string) (*Channel, error) {
	s.lock.RLock()
	defer s.lock.RUnlock()
	channel := s.Channels[name]
	if channel == nil {
		return nil, ErrChannelDoesNotExist
	}
	return channel, nil
}

func (s *ChannelStore) CreateChannel(channelName string) error {
	if channelName == "" {
		return ErrEmptyChannelName
	}
	s.lock.Lock()
	defer s.lock.Unlock()
	if s.Channels[channelName] != nil {
		return fmt.Errorf("channel `%s` already exists", channelName)
	}
	s.Channels[channelName] = &Channel{members: make(map[SockchatUserHandler]bool)}
	return nil
}

func (s *ChannelStore) AddUserToChannel(channelName string, user SockchatUserHandler) error {
	if channelName == "" {
		return ErrEmptyChannelName
	}
	s.lock.Lock()
	defer s.lock.Unlock()
	channel := s.Channels[channelName]
	if channel == nil {
		return ErrChannelDoesNotExist
	}
	if channel.HasMember(user) {
		return ErrUserAlreadyInChannel
	}
	channel.AddMember(user)
	channel.MessageMembers(NewSocketMessage(UserJoinedChannelEvent, ChannelUserChangeEvent{channelName, user.getNick()}))
	return nil
}

func (s *ChannelStore) RemoveUserFromChannel(channelName string, user SockchatUserHandler) error {
	if channelName == "" {
		return ErrEmptyChannelName
	}
	channel, err := s.GetChannel(channelName)
	if err != nil {
		return err
	}
	s.lock.Lock()
	defer s.lock.Unlock()
	if !channel.HasMember(user) {
		return ErrUserNotInChannel
	}
	channel.RemoveMember(user)
	channel.MessageMembers(NewSocketMessage(UserLeftChannelEvent, ChannelUserChangeEvent{channelName, user.getNick()}))
	return nil
}

// Removes user from all channels
func (s *ChannelStore) DisconnectUser(user SockchatUserHandler) {
	s.lock.RLock()
	defer s.lock.RUnlock()
	for channelName := range s.Channels {
		go s.RemoveUserFromChannel(channelName, user)
	}
}

type Channel struct {
	members map[SockchatUserHandler]bool
	lock    sync.RWMutex
}

func (c *Channel) AddMember(user SockchatUserHandler) {
	c.lock.Lock()
	defer c.lock.Unlock()
	if c.members == nil {
		c.members = make(map[SockchatUserHandler]bool)
	}
	c.members[user] = true
}

func (c *Channel) RemoveMember(user SockchatUserHandler) {
	c.lock.Lock()
	defer c.lock.Unlock()
	delete(c.members, user)
}

func (c *Channel) GetMembers() map[SockchatUserHandler]bool {
	c.lock.RLock()
	defer c.lock.RUnlock()
	return c.members
}

func (c *Channel) HasMember(user SockchatUserHandler) bool {
	c.lock.RLock()
	defer c.lock.RUnlock()
	return c.members[user]
}

func (c *Channel) MessageMembers(message SocketMessage) {
	// TODO write message to ES here
	c.lock.RLock()
	defer c.lock.RUnlock()
	for user := range c.members {
		go user.Write(message)
	}
}
