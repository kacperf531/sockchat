package sockchat

import (
	"fmt"
)

// FileSystemPlayerStore stores players in the filesystem.
type SockChatStore struct {
	Channels map[string]*Channel
}

func (store *SockChatStore) GetChannel(name string) (*Channel, error) {
	channel := store.Channels[name]
	if channel == nil {
		return nil, fmt.Errorf("channel %s does not exist", name)
	}
	return channel, nil
}

func (store *SockChatStore) CreateChannel(name string) error {
	channel, _ := store.GetChannel(name)
	if channel != nil {
		return fmt.Errorf("channel `%s` already exists", name)
	}
	store.Channels[name] = &Channel{}
	return nil
}

func (store *SockChatStore) JoinChannel(channelName string, conn *SockChatWS) error {
	channel, err := store.GetChannel(channelName)
	if err != nil {
		return err
	}
	channel.Users = append(channel.Users, conn)
	return nil
}

func NewSockChatStore() (*SockChatStore, error) {
	return &SockChatStore{Channels: map[string]*Channel{}}, nil
}
