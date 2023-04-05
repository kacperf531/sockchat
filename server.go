package sockchat

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/kacperf531/sockchat/errors"
	"github.com/kacperf531/sockchat/storage"
)

const (
	LoginAction       = "login"
	JoinAction        = "join"
	CreateAction      = "create"
	LeaveAction       = "leave"
	SendMessageAction = "send_message"
	ResponseDeadline  = 3 * time.Second

	// The connections that haven't logged in successfully will be disconnected after this time
	defaultTimeoutUnauthorized = 1 * time.Minute
	defaultTimeoutAuthorized   = 10 * time.Minute
)

// ChannelStore stores information about channels
type ChannelStore interface {
	GetChannel(name string) (*Channel, error)
	CreateChannel(name string) error
	AddUserToChannel(channelName string, conn *SockChatWS) error
	RemoveUserFromChannel(channelName string, conn *SockChatWS) error
	ChannelHasUser(channelName string, conn *SockChatWS) bool
	DisconnectUser(conn *SockChatWS)
}

type SockchatServer struct {
	store ChannelStore
	http.Handler
	userService         UserService
	timeoutUnauthorized time.Duration
	timeoutAuthorized   time.Duration
}

func NewSocketMessage(action string, payload any) SocketMessage {
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		log.Fatalf("error while marshaling payload %v", err)
	}
	return SocketMessage{Action: action, Payload: payloadBytes}
}

func NewErrorMessage(details string) SocketMessage {
	return NewSocketMessage("invalid_request_received", map[string]string{"details": details})
}

func NewSockChatServer(store ChannelStore, userStore storage.UserStore) *SockchatServer {
	s := new(SockchatServer)
	s.store = store
	s.userService = UserService{store: userStore}

	s.SetTimeoutValues(defaultTimeoutAuthorized, defaultTimeoutUnauthorized)

	router := http.NewServeMux()
	router.Handle("/ws", http.HandlerFunc(s.webSocket))
	router.Handle("/register", http.HandlerFunc(s.register))
	router.Handle("/edit_profile", http.HandlerFunc(s.editProfile))
	s.Handler = router

	return s
}

var wsUpgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

func (s *SockchatServer) SetTimeoutValues(authorized, unauthorized time.Duration) {
	s.timeoutAuthorized = authorized
	s.timeoutUnauthorized = unauthorized
}

func (s *SockchatServer) webSocket(w http.ResponseWriter, r *http.Request) {
	conn := newSockChatWS(w, r)
	defer s.shutConnection(conn)
	conn.SetReadDeadline(time.Now().Add(s.timeoutUnauthorized))
	for {
		receivedMsg, err := conn.ReadSocketMsg()
		if err != nil {
			if os.IsTimeout(err) {
				conn.WriteSocketMsg(NewSocketMessage("connection_timed_out", "{}"))
			}
			log.Printf("error occured when listening for ws messages, closing connection: %v", err)
			s.shutConnection(conn)
			break
		}
		if receivedMsg.Action == LoginAction {
			ctx, cancel := context.WithTimeout(r.Context(), ResponseDeadline)
			defer cancel()
			s.loginUser(ctx, *receivedMsg, conn)
		} else {
			if conn.nick == "" {
				conn.WriteSocketMsg(NewErrorMessage("Log in first using " + LoginAction))
			}
			conn.SetReadDeadline(time.Now().Add(s.timeoutAuthorized))
			switch receivedMsg.Action {
			case CreateAction:
				s.createNewChannel(*receivedMsg, conn)
			case JoinAction:
				s.joinChannel(*receivedMsg, conn)
			case LeaveAction:
				s.leaveChannel(*receivedMsg, conn)
			case SendMessageAction:
				s.sendMessageToChannel(*receivedMsg, conn)
			default:
				log.Printf("Unexpected request received: %s", receivedMsg.Action)
				conn.WriteSocketMsg(NewErrorMessage("action not supported"))
			}
		}

	}
}

func (s *SockchatServer) register(w http.ResponseWriter, r *http.Request) {
	userData := UserRequest{}
	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		log.Printf("error reading register request: %v", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	err = json.Unmarshal(bodyBytes, &userData)
	if err != nil {
		log.Printf("error unmarshaling register request: %v", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), ResponseDeadline)
	defer cancel()
	err = s.userService.CreateUser(ctx, &userData)
	if err != nil {
		if err == errors.ResourceConflict {
			w.WriteHeader(http.StatusConflict)
		} else {
			w.WriteHeader(http.StatusUnprocessableEntity)
		}
		return
	}
	w.WriteHeader(http.StatusCreated)
}

func (s *SockchatServer) editProfile(w http.ResponseWriter, r *http.Request) {
	userData := UserRequest{}
	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		log.Printf("error reading register request: %v", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	err = json.Unmarshal(bodyBytes, &userData)
	if err != nil {
		log.Printf("error unmarshaling register request: %v", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), ResponseDeadline)
	defer cancel()
	err = s.userService.EditUser(ctx, &userData)
	if err != nil {
		if err == errors.Unauthorized {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		w.WriteHeader(http.StatusUnprocessableEntity)
		return
	}
	w.WriteHeader(http.StatusOK)

}

func (s *SockchatServer) shutConnection(conn *SockChatWS) {
	s.store.DisconnectUser(conn)
	conn.Close()
}

func (s *SockchatServer) sendMessageToChannel(request SocketMessage, conn *SockChatWS) {
	req := SendMessageRequest{}
	if err := json.Unmarshal(request.Payload, &req); err != nil {
		log.Printf("error while unmarshaling request for sending message: %v", err)
		return
	}
	if !s.store.ChannelHasUser(req.Channel, conn) {
		conn.WriteSocketMsg(NewErrorMessage("you are not member of this channel"))
		return
	}
	channel, _ := s.store.GetChannel(req.Channel)
	for user := range channel.Users {
		user.WriteSocketMsg(NewSocketMessage("new_message", MessageEvent{Channel: req.Channel, Author: conn.nick, Text: req.Text}))
	}

}

func (s *SockchatServer) loginUser(ctx context.Context, request SocketMessage, conn *SockChatWS) {
	req := UnmarshalLoginRequest(request.Payload)
	if err := s.userService.LoginUser(ctx, req.Nick, req.Password); err != nil {
		conn.WriteSocketMsg(NewErrorMessage(err.Error()))
	} else {
		conn.nick = req.Nick
		conn.WriteSocketMsg(NewSocketMessage(fmt.Sprintf("logged_in:%s", req.Nick), "{}"))
		conn.SetReadDeadline(time.Now().Add(s.timeoutAuthorized))
	}
}

func (s *SockchatServer) createNewChannel(request SocketMessage, conn *SockChatWS) {
	channel := UnmarshalChannelRequest(request.Payload)
	if err := s.store.CreateChannel(channel.Name); err != nil {
		conn.WriteSocketMsg(NewErrorMessage(err.Error()))
	} else {
		conn.WriteSocketMsg(NewSocketMessage("channel_created", channel))
	}
}

func (s *SockchatServer) joinChannel(request SocketMessage, conn *SockChatWS) {
	channel := UnmarshalChannelRequest(request.Payload)
	if err := s.store.AddUserToChannel(channel.Name, conn); err != nil {
		conn.WriteSocketMsg(NewErrorMessage(err.Error()))
	} else {
		conn.WriteSocketMsg(NewSocketMessage("channel_joined", ChannelUserChangeEvent{Channel: channel.Name, Nick: conn.nick}))
	}
}

func (s *SockchatServer) leaveChannel(request SocketMessage, conn *SockChatWS) {
	channel := ChannelRequest{}
	if err := json.Unmarshal(request.Payload, &channel); err != nil {
		log.Printf("error while unmarshaling request for leaving channel: %v", err)
	}
	if err := s.store.RemoveUserFromChannel(channel.Name, conn); err != nil {
		conn.WriteSocketMsg(NewErrorMessage(err.Error()))
	} else {
		conn.WriteSocketMsg(NewSocketMessage("channel_left", ChannelUserChangeEvent{Channel: channel.Name, Nick: conn.nick}))
	}
}

type SockChatWS struct {
	*websocket.Conn
	writeLock sync.Mutex
	readLock  sync.Mutex
	nick      string
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

func (w *SockChatWS) ReadSocketMsg() (*SocketMessage, error) {
	msgBytes, err := w.ReadMsg()
	msg := &SocketMessage{}
	json.Unmarshal(msgBytes, &msg)
	return msg, err
}

func (w *SockChatWS) WriteSocketMsg(m SocketMessage) {
	w.writeLock.Lock()
	defer w.writeLock.Unlock()
	err := w.WriteJSON(m)
	if err != nil {
		log.Printf("Error writing message %s with payload %s to websocket: %v", m.Action, string(m.Payload), err)
	}
}
