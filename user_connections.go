package sockchat

import (
	"log"
	"sync"
	"time"

	"github.com/kacperf531/sockchat/api"
)

// ConnectedUsersPool is responsible for tracking all user handlers
type ConnectedUsersPool struct {
	handlers     map[string]api.SockchatUserHandler
	connections  map[api.SockchatWebsocketConnection]string
	lock         sync.RWMutex
	channelStore api.SockchatChannelStore
}

func NewConnectedUsersPool(channelStore api.SockchatChannelStore) *ConnectedUsersPool {
	manager := &ConnectedUsersPool{
		handlers:     make(map[string]api.SockchatUserHandler),
		connections:  make(map[api.SockchatWebsocketConnection]string),
		channelStore: channelStore,
	}
	return manager
}

func (m *ConnectedUsersPool) GetHandler(nick string) (api.SockchatUserHandler, bool) {
	m.lock.RLock()
	defer m.lock.RUnlock()
	handler, ok := m.handlers[nick]
	return handler, ok
}

func (m *ConnectedUsersPool) AddConnection(conn api.SockchatWebsocketConnection, nick string) {
	handler, ok := m.GetHandler(nick)
	if !ok {
		handler = m.addHandler(nick)
	}
	handler.AddConnection(conn)
	m.lock.Lock()
	defer m.lock.Unlock()
	m.connections[conn] = nick
}

func (m *ConnectedUsersPool) addHandler(nick string) api.SockchatUserHandler {
	handler := NewUserHandler(nick, m.channelStore)
	m.lock.Lock()
	defer m.lock.Unlock()
	m.handlers[nick] = handler
	return handler
}

func (m *ConnectedUsersPool) RemoveConnection(conn api.SockchatWebsocketConnection) {
	m.lock.RLock()
	nick := m.connections[conn]
	m.lock.RUnlock()
	handler, ok := m.GetHandler(nick)
	if !ok {
		log.Printf("could not drop connection - no handler found for %s", nick)
		return
	}

	if handler.GetActiveConnectionsCount() > 1 {
		handler.RemoveConnection(conn)
	} else {
		m.lock.Lock()
		defer m.lock.Unlock()
		delete(m.handlers, nick)
		delete(m.connections, conn)
		m.channelStore.DisconnectUser(handler)
	}
}

// UserHandler manages connections of a single connected user
type UserHandler struct {
	nick         string
	connections  map[api.SockchatWebsocketConnection]bool
	requests     chan *UserHandlerRequest
	lock         sync.RWMutex
	channelStore api.SockchatChannelStore
}

type UserHandlerRequest struct {
	action      string
	payload     any
	errCallback chan error
}

func NewUserHandler(nick string, store api.SockchatChannelStore) *UserHandler {
	handler := UserHandler{
		nick:         nick,
		connections:  make(map[api.SockchatWebsocketConnection]bool),
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
		case api.CreateAction:
			err := u.channelStore.CreateChannel(req.payload.(*api.ChannelRequest).Name)
			if err != nil {
				req.errCallback <- err
				continue
			}
			req.errCallback <- u.channelStore.AddUserToChannel(req.payload.(*api.ChannelRequest).Name, u)
		case api.JoinAction:
			err := u.channelStore.AddUserToChannel(req.payload.(*api.ChannelRequest).Name, u)
			req.errCallback <- err
		case api.LeaveAction:
			channelName := req.payload.(*api.ChannelRequest).Name
			err := u.channelStore.RemoveUserFromChannel(channelName, u)
			if err == nil {
				go u.Write(api.NewSocketMessage(api.YouLeftChannelEvent, api.ChannelUserChangeEvent{Channel: channelName, Nick: u.GetNick()}))
			}
			req.errCallback <- err
		case api.SendMessageAction:
			reqFields := req.payload.(*api.SendMessageRequest)
			if !u.channelStore.IsUserPresentIn(u, reqFields.Channel) {
				req.errCallback <- api.ErrUserNotInChannel
				continue
			}
			req.errCallback <- u.channelStore.MessageChannel(&api.MessageEvent{Text: reqFields.Text, Channel: reqFields.Channel, Author: u.GetNick(), Timestamp: time.Now().Unix()})
		}
	}
}

func (u *UserHandler) MakeRequest(action string, payload any) error {
	errCallback := make(chan error)
	u.requests <- &UserHandlerRequest{action, payload, errCallback}
	return <-errCallback
}

func (u *UserHandler) Write(msg api.SocketMessage) {
	u.lock.RLock()
	defer u.lock.RUnlock()
	wg := sync.WaitGroup{}
	wg.Add(len(u.connections))
	for conn := range u.connections {
		go func(conn api.SockchatWebsocketConnection) {
			conn.WriteSocketMsg(msg)
			wg.Done()
		}(conn)
	}
	wg.Wait()
}

func (u *UserHandler) AddConnection(conn api.SockchatWebsocketConnection) {
	u.lock.Lock()
	defer u.lock.Unlock()
	u.connections[conn] = true
}

func (u *UserHandler) RemoveConnection(conn api.SockchatWebsocketConnection) {
	u.lock.Lock()
	defer u.lock.Unlock()
	delete(u.connections, conn)
}

func (u *UserHandler) GetActiveConnectionsCount() int {
	u.lock.RLock()
	defer u.lock.RUnlock()
	return len(u.connections)
}

func (u *UserHandler) GetNick() string {
	return u.nick
}
