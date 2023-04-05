package sockchat

import (
	"errors"
	"fmt"
	"sync"
)

var ErrChannelDoesNotExist = errors.New("channel does not exist")

// FileSystemPlayerStore stores players in the filesystem.
type SockChatStore struct {
	Channels map[string]*Channel
	// A mutex is used to synchronize read/write access to the map
	lock sync.RWMutex
}

func NewSockChatStore() (*SockChatStore, error) {
	return &SockChatStore{Channels: make(map[string]*Channel)}, nil
}

func (s *SockChatStore) GetChannel(name string) (*Channel, error) {
	s.lock.RLock()
	defer s.lock.RUnlock()
	channel := s.Channels[name]
	if channel == nil {
		return nil, ErrChannelDoesNotExist
	}
	return channel, nil
}

func (s *SockChatStore) CreateChannel(channelName string) error {
	s.lock.Lock()
	defer s.lock.Unlock()
	if s.Channels[channelName] != nil {
		return fmt.Errorf("channel `%s` already exists", channelName)
	}
	s.Channels[channelName] = &Channel{Users: make(map[*SockChatWS]bool)}
	return nil
}

func (s *SockChatStore) AddUserToChannel(channelName string, conn *SockChatWS) error {
	s.lock.Lock()
	defer s.lock.Unlock()
	channel := s.Channels[channelName]
	if channel == nil {
		return ErrChannelDoesNotExist
	}
	channel.Users[conn] = true
	return nil
}

func (s *SockChatStore) RemoveUserFromChannel(channelName string, conn *SockChatWS) error {
	s.lock.Lock()
	defer s.lock.Unlock()
	channel := s.Channels[channelName]
	if channel == nil {
		return ErrChannelDoesNotExist
	}
	delete(channel.Users, conn)
	return nil
}

func (s *SockChatStore) ChannelHasUser(channelName string, conn *SockChatWS) bool {
	s.lock.RLock()
	defer s.lock.RUnlock()
	channel, err := s.GetChannel(channelName)
	if err != nil {
		return false
	}
	return channel.Users[conn]
}

// Removes user from all channels
func (s *SockChatStore) DisconnectUser(conn *SockChatWS) {
	for channelName := range s.Channels {
		s.RemoveUserFromChannel(channelName, conn)
	}
}
