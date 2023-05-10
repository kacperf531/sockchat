package services

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/kacperf531/sockchat/api"
)

var wsUpgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

type MessagingAPI struct {
	TimeoutAuthorized   time.Duration
	TimeoutUnauthorized time.Duration
	ConnectedUsers      api.SockchatUserManager
	UserProfiles        api.SockchatProfileStore
}

func (s *MessagingAPI) HandleRequests(router *http.ServeMux) {
	router.Handle("/ws", http.HandlerFunc(s.ServeSession))
}

func (s *MessagingAPI) ServeSession(w http.ResponseWriter, r *http.Request) {
	conn := newSockChatWS(w, r)
	defer s.shutConnection(conn)
	conn.SetReadDeadline(time.Now().Add(s.TimeoutUnauthorized))
	var nick string
	for {
		receivedMsg, err := conn.ReadSocketMsg()
		if err != nil {
			break
		}

		if conn.authorized {
			err = s.serveAuthorizedConnection(conn, nick, *receivedMsg)
			if err != nil {
				log.Printf("error serving authorized connection: %v", err)
				break
			}
			continue
		}
		nick, err = s.authorizeConnection(*receivedMsg, conn)
		if err != nil {
			conn.WriteSocketMsg(api.NewSocketError(err.Error()))
		}
		conn.SetReadDeadline(time.Now().Add(s.TimeoutAuthorized))
	}
}

func (s *MessagingAPI) authorizeConnection(request api.SocketMessage, conn *SockChatWS) (string, error) {
	if request.Action == api.LoginAction {
		req, err := api.UnmarshalLoginRequest(request.Payload)
		if err != nil {
			return "", api.ErrInvalidRequest
		}
		u, err := s.loginUser(req)
		if err == nil {
			conn.authorized = true
			s.ConnectedUsers.AddConnection(conn, u.Nick)
			conn.WriteSocketMsg(api.NewSocketMessage("logged_in:"+u.Nick, "{}"))
			conn.SetReadDeadline(time.Now().Add(s.TimeoutAuthorized))
			return req.Nick, nil
		}
		return "", err
	}
	return "", fmt.Errorf("you must log in first using " + api.LoginAction + " action")
}

func (s *MessagingAPI) loginUser(req *api.LoginRequest) (*api.PublicProfile, error) {
	ctx, cancel := context.WithTimeout(context.Background(), ResponseDeadline)
	defer cancel()
	if s.UserProfiles.IsAuthValid(ctx, req.Nick, req.Password) {
		return &api.PublicProfile{Nick: req.Nick}, nil
	}
	return nil, fmt.Errorf("login rejected: invalid credentials")
}

func (s *MessagingAPI) serveAuthorizedConnection(conn *SockChatWS, nick string, receivedMsg api.SocketMessage) error {
	conn.SetReadDeadline(time.Now().Add(s.TimeoutAuthorized))
	req, err := parseWebsocketMessage(receivedMsg)
	if err != nil {
		conn.WriteSocketMsg(api.NewSocketError(err.Error()))
		return nil
	}
	handler, ok := s.ConnectedUsers.GetHandler(nick)
	if !ok {
		return fmt.Errorf("handler not found for user `%s`", nick)
	}
	err = handler.MakeRequest(receivedMsg.Action, req)
	if err != nil {
		conn.WriteSocketMsg(api.NewSocketError(err.Error()))
	}
	return nil
}

func (s *MessagingAPI) shutConnection(conn *SockChatWS) {
	if conn.authorized {
		s.ConnectedUsers.RemoveConnection(conn)
	}
	conn.Close()
}

func parseWebsocketMessage(msg api.SocketMessage) (interface{}, error) {
	switch msg.Action {
	case api.CreateAction, api.JoinAction, api.LeaveAction:
		return api.UnmarshalChannelRequest(msg.Payload)
	case api.SendMessageAction:
		return api.UnmarshalMessageRequest(msg.Payload)
	default:
		return nil, fmt.Errorf(api.ErrInvalidRequest.Error())
	}
}

type SockChatWS struct {
	*websocket.Conn
	writeLock  sync.Mutex
	readLock   sync.Mutex
	authorized bool
}

func newSockChatWS(w http.ResponseWriter, r *http.Request) *SockChatWS {
	conn, err := wsUpgrader.Upgrade(w, r, nil)

	if err != nil {
		log.Printf("problem upgrading connection to WebSockets %v\n", err)
	}

	return &SockChatWS{Conn: conn}
}

func (w *SockChatWS) ReadMsg() ([]byte, error) {
	w.readLock.Lock()
	defer w.readLock.Unlock()
	_, msgBytes, err := w.ReadMessage()
	return msgBytes, err
}

func (w *SockChatWS) ReadSocketMsg() (*api.SocketMessage, error) {
	msgBytes, err := w.ReadMsg()
	if err != nil {
		if os.IsTimeout(err) {
			w.WriteSocketMsg(api.NewSocketMessage("connection_timed_out", "{}"))
		}
		if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
			log.Printf("error while reading messages from websocket: %v", err)
		}
		return nil, err
	}
	msg := &api.SocketMessage{}
	json.Unmarshal(msgBytes, &msg)
	return msg, nil
}

func (w *SockChatWS) WriteSocketMsg(m api.SocketMessage) {
	w.writeLock.Lock()
	defer w.writeLock.Unlock()
	err := w.WriteJSON(m)
	if err != nil {
		log.Printf("Error writing message %s with payload %s to websocket: %v", m.Action, string(m.Payload), err)
	}
}
