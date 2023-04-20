package sockchat

import (
	"log"
	"sync"

	"github.com/kacperf531/sockchat/common"
)

// SockchatUserHandler manages user actions
type SockchatUserHandler interface {
	MakeRequest(action string, payload any) error
	Write(msg SocketMessage)
	AddConnection(conn *SockChatWS)
	RemoveConnection(conn *SockChatWS)
	getActiveConnectionsCount() int
	getNick() string
}

// ConnectedUsersPool is responsible for tracking all user handlers
type ConnectedUsersPool struct {
	handlers     map[string]SockchatUserHandler
	lock         sync.RWMutex
	channelStore SockchatChannelStore
}

func NewConnectedUsersPool(channelStore SockchatChannelStore) *ConnectedUsersPool {
	manager := &ConnectedUsersPool{
		handlers:     make(map[string]SockchatUserHandler),
		channelStore: channelStore,
	}
	return manager
}

func (m *ConnectedUsersPool) GetHandler(nick string) (SockchatUserHandler, bool) {
	m.lock.RLock()
	defer m.lock.RUnlock()
	handler, ok := m.handlers[nick]
	return handler, ok
}

func (m *ConnectedUsersPool) AddConnection(conn *SockChatWS, nick string) {
	handler, ok := m.GetHandler(nick)
	if !ok {
		handler = NewUserHandler(conn, nick, m.channelStore)
		m.lock.Lock()
		defer m.lock.Unlock()
		m.handlers[nick] = handler
	}
	handler.AddConnection(conn)
	conn.userHandler = handler
}

func (m *ConnectedUsersPool) RemoveConnection(conn *SockChatWS) {
	nick := conn.userHandler.getNick()
	handler, ok := m.GetHandler(nick)
	if !ok {
		log.Printf("could not drop connection - no handler found for %s", nick)
		return
	}

	if handler.getActiveConnectionsCount() > 1 {
		handler.RemoveConnection(conn)
	} else {
		m.lock.Lock()
		defer m.lock.Unlock()
		delete(m.handlers, nick)
		m.channelStore.DisconnectUser(handler)
	}
}

// UserHandler manages connections of a single connected user
type UserHandler struct {
	nick         string
	connections  map[*SockChatWS]bool
	requests     chan *UserHandlerRequest
	lock         sync.RWMutex
	channelStore SockchatChannelStore
}

type UserHandlerRequest struct {
	action      string
	payload     any
	errCallback chan error
}

func NewUserHandler(conn *SockChatWS, nick string, store SockchatChannelStore) *UserHandler {
	handler := UserHandler{
		nick:         nick,
		connections:  make(map[*SockChatWS]bool),
		requests:     make(chan *UserHandlerRequest),
		channelStore: store,
	}
	go handler.HandleRequests()
	return &handler
}

func (u *UserHandler) HandleRequests() {
	for {
		req := <-u.requests
		switch req.action {
		case CreateAction:
			err := u.channelStore.CreateChannel(req.payload.(*ChannelRequest).Name)
			if err != nil {
				req.errCallback <- err
				continue
			}
			req.errCallback <- u.channelStore.AddUserToChannel(req.payload.(*ChannelRequest).Name, u)
		case JoinAction:
			err := u.channelStore.AddUserToChannel(req.payload.(*ChannelRequest).Name, u)
			req.errCallback <- err
		case LeaveAction:
			channelName := req.payload.(*ChannelRequest).Name
			err := u.channelStore.RemoveUserFromChannel(channelName, u)
			if err == nil {
				go u.Write(NewSocketMessage(YouLeftChannelEvent, ChannelUserChangeEvent{channelName, u.getNick()}))
			}
			req.errCallback <- err
		case SendMessageAction:
			channel, err := u.channelStore.GetChannel(req.payload.(*SendMessageRequest).Channel)
			if !channel.GetMembers()[u] {
				req.errCallback <- ErrUserNotInChannel
				continue
			}
			req.errCallback <- err
			channel.MessageMembers(NewSocketMessage(NewMessageEvent, common.MessageEvent{Text: req.payload.(*SendMessageRequest).Text, Channel: req.payload.(*SendMessageRequest).Channel, Author: u.getNick()}))
		}
	}
}

func (u *UserHandler) MakeRequest(action string, payload any) error {
	errCallback := make(chan error)
	u.requests <- &UserHandlerRequest{action, payload, errCallback}
	return <-errCallback
}

func (u *UserHandler) Write(msg SocketMessage) {
	u.lock.RLock()
	defer u.lock.RUnlock()
	wg := sync.WaitGroup{}
	wg.Add(len(u.connections))
	for conn := range u.connections {
		go func(conn *SockChatWS) {
			conn.WriteSocketMsg(msg)
			wg.Done()
		}(conn)
	}
	wg.Wait()
}

func (u *UserHandler) AddConnection(conn *SockChatWS) {
	u.lock.Lock()
	defer u.lock.Unlock()
	u.connections[conn] = true
}

func (u *UserHandler) RemoveConnection(conn *SockChatWS) {
	u.lock.Lock()
	defer u.lock.Unlock()
	delete(u.connections, conn)
}

func (u *UserHandler) getActiveConnectionsCount() int {
	u.lock.RLock()
	defer u.lock.RUnlock()
	return len(u.connections)
}

func (u *UserHandler) getNick() string {
	return u.nick
}
