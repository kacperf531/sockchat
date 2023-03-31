package sockchat

import (
	"context"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/kacperf531/sockchat/storage"
)

const (
	JoinAction        = "join"
	CreateAction      = "create"
	LeaveAction       = "leave"
	SendMessageAction = "send_message"
	ResponseDeadline  = 3 * time.Second
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
	userService UserService
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

func (s *SockchatServer) webSocket(w http.ResponseWriter, r *http.Request) {
	conn := newSockChatWS(w, r)
	defer s.shutConnection(conn)
	for {
		receivedMsg, err := conn.ReadSocketMsg()
		if err != nil {
			log.Printf("error occured when listening for ws messages, closing connection %v", err)
			s.shutConnection(conn)
			break
		}

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
		w.WriteHeader(http.StatusUnprocessableEntity)
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
	// TODO: Handle authorization errors separately
	ctx, cancel := context.WithTimeout(r.Context(), ResponseDeadline)
	defer cancel()
	err = s.userService.CreateUser(ctx, &userData)
	if err != nil {
		w.WriteHeader(http.StatusUnprocessableEntity)
	}
	w.WriteHeader(http.StatusOK)

}

func (s *SockchatServer) shutConnection(conn *SockChatWS) {
	conn.Close()
	s.store.DisconnectUser(conn)
}

func (s *SockchatServer) sendMessageToChannel(request SocketMessage, conn *SockChatWS) {
	message := MessageEvent{}
	if err := json.Unmarshal(request.Payload, &message); err != nil {
		log.Printf("error while unmarshaling request for sending message: %v", err)
		return
	}
	if !s.store.ChannelHasUser(message.Channel, conn) {
		conn.WriteSocketMsg(NewErrorMessage("you are not member of this channel"))
		return
	}
	channel, _ := s.store.GetChannel(message.Channel)
	for user := range channel.Users {
		user.WriteSocketMsg(NewSocketMessage("new_message", message))
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
		conn.WriteSocketMsg(NewSocketMessage("channel_joined", ChannelUserChangeEvent{Name: channel.Name, UserName: conn.RemoteAddr().String()}))
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
		conn.WriteSocketMsg(NewSocketMessage("channel_left", ChannelUserChangeEvent{Name: channel.Name, UserName: conn.RemoteAddr().String()}))
	}
}

type SockChatWS struct {
	*websocket.Conn
	writeLock sync.Mutex
	readLock  sync.Mutex
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
