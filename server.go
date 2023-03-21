package sockchat

import (
	"log"
	"net/http"

	"github.com/gorilla/websocket"
)

// ChannelStore stores information about channels
type ChannelStore interface {
	GetChannel(name string) int
	CreateChannel(name string) int
}

type SockchatServer struct {
	store ChannelStore
	http.Handler
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
	channelName := conn.WaitForMsg()
	s.store.CreateChannel(string(channelName))
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

func (w *SockChatWS) WaitForMsg() string {
	_, msg, err := w.ReadMessage()
	if err != nil {
		log.Printf("error reading from websocket %v\n", err)
	}
	return string(msg)
}
