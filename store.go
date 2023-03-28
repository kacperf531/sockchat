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
	return &SockChatStore{Channels: map[string]*Channel{}}, nil
}

func (store *SockChatStore) GetChannel(name string) (*Channel, error) {
	store.lock.RLock()
	defer store.lock.RUnlock()
	channel := store.Channels[name]
	if channel == nil {
		return nil, ErrChannelDoesNotExist
	}
	return channel, nil
}

func (store *SockChatStore) CreateChannel(channelName string) error {
	store.lock.Lock()
	defer store.lock.Unlock()
	if store.Channels[channelName] != nil {
		return fmt.Errorf("channel `%s` already exists", channelName)
	}
	store.Channels[channelName] = &Channel{Users: make(map[*SockChatWS]bool)}
	return nil
}

func (store *SockChatStore) AddUserToChannel(channelName string, conn *SockChatWS) error {
	store.lock.Lock()
	defer store.lock.Unlock()
	channel := store.Channels[channelName]
	if channel == nil {
		return ErrChannelDoesNotExist
	}
	channel.Users[conn] = true
	return nil
}

func (store *SockChatStore) RemoveUserFromChannel(channelName string, conn *SockChatWS) error {
	store.lock.Lock()
	defer store.lock.Unlock()
	channel := store.Channels[channelName]
	if channel == nil {
		return ErrChannelDoesNotExist
	}
	delete(channel.Users, conn)
	return nil
}

func (store *SockChatStore) ChannelHasUser(channelName string, conn *SockChatWS) bool {
	store.lock.RLock()
	defer store.lock.RUnlock()
	channel, err := store.GetChannel(channelName)
	if err != nil {
		return false
	}
	return channel.Users[conn]
}

// Removes user from all channels
func (store *SockChatStore) DisconnectUser(conn *SockChatWS) {
	for channelName := range store.Channels {
		store.RemoveUserFromChannel(channelName, conn)
	}
}
