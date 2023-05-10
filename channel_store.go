package sockchat

import (
	"log"
	"sync"

	"github.com/kacperf531/sockchat/api"
)

type ChannelStore struct {
	Channels     map[string]*Channel
	lock         sync.RWMutex
	messageStore api.SockchatMessageStore
}

func NewChannelStore(messageStore api.SockchatMessageStore) *ChannelStore {
	return &ChannelStore{Channels: make(map[string]*Channel), messageStore: messageStore}
}

func (s *ChannelStore) getChannel(name string) (*Channel, error) {
	if err := s.validateChannelName(name); err != nil {
		return nil, err
	}
	s.lock.RLock()
	defer s.lock.RUnlock()
	channel := s.Channels[name]
	if channel == nil {
		return nil, api.ErrChannelDoesNotExist
	}
	return channel, nil
}

func (s *ChannelStore) CreateChannel(channelName string) error {
	if err := s.validateChannelName(channelName); err != nil {
		return err
	}
	s.lock.Lock()
	defer s.lock.Unlock()
	if s.Channels[channelName] != nil {
		return api.ErrChannelAlreadyExists
	}
	s.Channels[channelName] = NewChannel()
	return nil
}

func (s *ChannelStore) AddUserToChannel(channelName string, user api.SockchatUserHandler) error {
	channel, err := s.getChannel(channelName)
	if err != nil {
		return err
	}
	if channel.HasMember(user) {
		return api.ErrUserAlreadyInChannel
	}
	channel.AddMember(user)
	channel.MessageMembers(api.NewSocketMessage(api.UserJoinedChannelEvent, api.ChannelUserChangeEvent{Channel: channelName, Nick: user.GetNick()}))
	return nil
}

func (s *ChannelStore) RemoveUserFromChannel(channelName string, user api.SockchatUserHandler) error {
	channel, err := s.getChannel(channelName)
	if err != nil {
		return err
	}
	if !channel.HasMember(user) {
		return api.ErrUserNotInChannel
	}
	channel.RemoveMember(user)
	channel.MessageMembers(api.NewSocketMessage(api.UserLeftChannelEvent, api.ChannelUserChangeEvent{Channel: channelName, Nick: user.GetNick()}))
	return nil
}

// Removes user from all channels
func (s *ChannelStore) DisconnectUser(user api.SockchatUserHandler) {
	s.lock.RLock()
	defer s.lock.RUnlock()
	for channelName := range s.Channels {
		go s.RemoveUserFromChannel(channelName, user)
	}
}

func (s *ChannelStore) ChannelExists(channelName string) bool {
	s.lock.RLock()
	defer s.lock.RUnlock()
	return s.Channels[channelName] != nil
}

func (s *ChannelStore) IsUserPresentIn(user api.SockchatUserHandler, channelName string) bool {
	channel, err := s.getChannel(channelName)
	if err != nil {
		return false
	}
	return channel.HasMember(user)
}

func (s *ChannelStore) MessageChannel(message *api.MessageEvent) error {
	channel, err := s.getChannel(message.Channel)
	if err != nil {
		return err
	}
	_, err = s.messageStore.IndexMessage(message)
	if err != nil {
		log.Printf("warning: failed to index message: %v", err)
		return api.ErrMessageNotSent
	}

	go channel.MessageMembers(api.NewSocketMessage(api.NewMessageEvent, message))
	return nil
}

func (s *ChannelStore) validateChannelName(channelName string) error {
	if channelName == "" {
		return api.ErrEmptyChannelName
	}
	return nil
}

type Channel struct {
	members map[api.SockchatUserHandler]bool
	lock    sync.RWMutex
}

func (c *Channel) AddMember(user api.SockchatUserHandler) {
	c.lock.Lock()
	defer c.lock.Unlock()
	if c.members == nil {
		c.members = make(map[api.SockchatUserHandler]bool)
	}
	c.members[user] = true
}

func (c *Channel) RemoveMember(user api.SockchatUserHandler) {
	c.lock.Lock()
	defer c.lock.Unlock()
	delete(c.members, user)
}

func (c *Channel) HasMember(user api.SockchatUserHandler) bool {
	c.lock.RLock()
	defer c.lock.RUnlock()
	return c.members[user]
}

func (c *Channel) MessageMembers(message api.SocketMessage) {
	c.lock.RLock()
	defer c.lock.RUnlock()
	for user := range c.members {
		go user.Write(message)
	}
}

func NewChannel() *Channel {
	return &Channel{members: make(map[api.SockchatUserHandler]bool)}
}
