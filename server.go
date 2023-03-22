package sockchat

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/gorilla/websocket"
)

// ChannelStore stores information about channels
type ChannelStore interface {
	GetChannel(name string) *Channel
	CreateChannel(name string) error
}

type SockchatServer struct {
	store ChannelStore
	http.Handler
}

type WSMsg struct {
	Action  string          `json:"action"`
	Payload json.RawMessage `json:"payload"`
}

type Channel struct {
	Name string `json:"name"`
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
	receivedMsg := conn.WaitForMsg()
	switch receivedMsg.Action {
	case "create":
		channel := Channel{}
		if err := json.Unmarshal(receivedMsg.Payload, &channel); err != nil {
			log.Printf("error while unmarshaling request for creating channel: %v", err)
		}
		if err := s.store.CreateChannel(channel.Name); err != nil {
			log.Printf("could not create new channel due to error %v", err)
		}
	}

}

type SockChatWS struct {
	*websocket.Conn
}

func newSockChatWS(w http.ResponseWriter, r *http.Request) *SockChatWS {
	conn, err := wsUpgrader.Upgrade(w, r, nil)

	if err != nil {
		log.Printf("problem upgrading connection to WebSockets %v\n", err)
	}

	return &SockChatWS{conn}
}

func (w *SockChatWS) WaitForMsg() WSMsg {
	_, msgBytes, err := w.ReadMessage()
	if err != nil {
		log.Printf("error reading from websocket %v\n", err)
	}
	msg := WSMsg{}

	json.Unmarshal(msgBytes, &msg)
	return msg
}
