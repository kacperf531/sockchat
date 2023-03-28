package sockchat

import (
	"encoding/json"
	"log"
	"net/http"
	"sync"

	"github.com/gorilla/websocket"
)

const (
	JoinAction        = "join"
	CreateAction      = "create"
	LeaveAction       = "leave"
	SendMessageAction = "send_message"
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

func NewSockChatServer(store ChannelStore) *SockchatServer {
	s := new(SockchatServer)
	s.store = store

	router := http.NewServeMux()
	router.Handle("/ws", http.HandlerFunc(s.webSocket))
	s.Handler = router

	return s
}

var wsUpgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

func (s *SockchatServer) webSocket(w http.ResponseWriter, r *http.Request) {
	conn := newSockChatWS(w, r)
	defer s.ShutConnection(conn)
	for {
		receivedMsg, err := conn.ReadSocketMsg()
		if err != nil {
			log.Printf("error occured when listening for ws messages, closing connection %v", err)
			s.ShutConnection(conn)
			break
		}

		switch receivedMsg.Action {
		case CreateAction:
			s.CreateNewChannel(*receivedMsg, conn)
		case JoinAction:
			s.JoinChannel(*receivedMsg, conn)
		case LeaveAction:
			s.LeaveChannel(*receivedMsg, conn)
		case SendMessageAction:
			s.SendMessageToChannel(*receivedMsg, conn)
		default:
			log.Printf("Unexpected request received: %s", receivedMsg.Action)
			conn.WriteSocketMsg(NewErrorMessage("action not supported"))
		}

	}

}

func (s *SockchatServer) ShutConnection(conn *SockChatWS) {
	conn.Close()
	s.store.DisconnectUser(conn)
}

func (s *SockchatServer) SendMessageToChannel(request SocketMessage, conn *SockChatWS) {
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

func (s *SockchatServer) CreateNewChannel(request SocketMessage, conn *SockChatWS) {
	channel := UnmarshalChannelRequest(request.Payload)
	if err := s.store.CreateChannel(channel.Name); err != nil {
		conn.WriteSocketMsg(NewErrorMessage(err.Error()))
	} else {
		conn.WriteSocketMsg(NewSocketMessage("channel_created", channel))
	}
}

func (s *SockchatServer) JoinChannel(request SocketMessage, conn *SockChatWS) {
	channel := UnmarshalChannelRequest(request.Payload)
	if err := s.store.AddUserToChannel(channel.Name, conn); err != nil {
		conn.WriteSocketMsg(NewErrorMessage(err.Error()))
	} else {
		conn.WriteSocketMsg(NewSocketMessage("channel_joined", ChannelUserChangeEvent{Name: channel.Name, UserName: conn.RemoteAddr().String()}))
	}
}

func (s *SockchatServer) LeaveChannel(request SocketMessage, conn *SockChatWS) {
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
