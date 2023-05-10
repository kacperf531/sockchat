package api

import (
	"context"
)

// SockchatChannelStore manages chat channels (rooms) and dispatches messages among their members
type SockchatChannelStore interface {
	CreateChannel(name string) error
	AddUserToChannel(channel string, user SockchatUserHandler) error
	RemoveUserFromChannel(channel string, user SockchatUserHandler) error
	MessageChannel(msg *MessageEvent) error
	DisconnectUser(user SockchatUserHandler)
	IsUserPresentIn(user SockchatUserHandler, channel string) bool
	ChannelExists(name string) bool
}

// SockchatProfileStore manages DB-stored user profiles
type SockchatProfileStore interface {
	Create(ctx context.Context, u *CreateProfileRequest) error
	Edit(ctx context.Context, nick string, u *EditProfileRequest) error
	IsAuthValid(ctx context.Context, nick, password string) bool
	GetProfile(ctx context.Context, nick string) (*PublicProfile, error)
}

// SockchatMessageStore manages messages in ES
type SockchatMessageStore interface {
	IndexMessage(msg *MessageEvent) (string, error)
	FindMessages(ctx context.Context, channel, query string) (ChannelHistory, error)
}

// SockchatUserManager manages user handlers that store connections and send messages to them
type SockchatUserManager interface {
	AddConnection(conn SockchatWebsocketConnection, nick string)
	RemoveConnection(conn SockchatWebsocketConnection)
	GetHandler(nick string) (SockchatUserHandler, bool)
}

// SockchatUserHandler manages user actions from multiple connections
type SockchatUserHandler interface {
	MakeRequest(action string, payload any) error
	Write(msg SocketMessage)
	AddConnection(conn SockchatWebsocketConnection)
	RemoveConnection(conn SockchatWebsocketConnection)
	GetActiveConnectionsCount() int
	GetNick() string
}

// SockchatWebsocketConnection represents single websocket connection
type SockchatWebsocketConnection interface {
	WriteSocketMsg(m SocketMessage)
	ReadSocketMsg() (*SocketMessage, error)
	ReadMsg() ([]byte, error)
}
