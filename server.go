package sockchat

import (
	"encoding/json"
	"log"
	"net/http"
	"sync"

	"github.com/gorilla/websocket"
)

// ChannelStore stores information about channels
type ChannelStore interface {
	GetChannel(name string) (*Channel, error)
	CreateChannel(name string) error
	JoinChannel(channelName string, conn *SockChatWS) error
	ChannelHasUser(channelName string, conn *SockChatWS) bool
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
	for {
		receivedMsg, err := conn.WaitForMsg()
		if err != nil {
			log.Print("error occured when listening for ws messages, closing connection")
			conn.Close()
			break
		}

		switch receivedMsg.Action {
		case "create":
			s.CreateNewChannel(*receivedMsg, conn)
		case "join":
			s.JoinChannel(*receivedMsg, conn)
		case "send_message":
			s.SendMessageToChannel(*receivedMsg, conn)
		default:
			conn.WriteSocketMsg(NewErrorMessage("action not supported"))
		}

	}

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
	for _, conn := range channel.Users {
		conn.WriteSocketMsg(NewSocketMessage("new_message", message))
	}

}

func (s *SockchatServer) CreateNewChannel(request SocketMessage, conn *SockChatWS) {
	channel := CreateChannel{}
	if err := json.Unmarshal(request.Payload, &channel); err != nil {
		log.Printf("error while unmarshaling request for creating channel: %v", err)
	}
	if err := s.store.CreateChannel(channel.Name); err != nil {
		conn.WriteSocketMsg(NewErrorMessage(err.Error()))
	} else {
		conn.WriteSocketMsg(NewSocketMessage("channel_created", channel))
	}
}

func (s *SockchatServer) JoinChannel(request SocketMessage, conn *SockChatWS) {
	channel := JoinChannel{}
	if err := json.Unmarshal(request.Payload, &channel); err != nil {
		log.Printf("error while unmarshaling request for joining channel: %v", err)
	}
	if err := s.store.JoinChannel(channel.Name, conn); err != nil {
		conn.WriteSocketMsg(NewErrorMessage(err.Error()))
	} else {
		conn.WriteSocketMsg(NewSocketMessage("channel_joined", ChannelJoined{Name: channel.Name, UserName: conn.RemoteAddr().String()}))
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

func (w *SockChatWS) WaitForMsg() (*SocketMessage, error) {
	w.readLock.Lock()
	defer w.readLock.Unlock()
	_, msgBytes, err := w.ReadMessage()
	if err != nil {
		return nil, err
	}
	msg := &SocketMessage{}

	json.Unmarshal(msgBytes, &msg)
	return msg, nil
}

func (w *SockChatWS) WriteSocketMsg(m SocketMessage) error {
	w.writeLock.Lock()
	defer w.writeLock.Unlock()
	return w.WriteJSON(m)
}
