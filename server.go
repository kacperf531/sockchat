package sockchat

import (
	"net/http"
)

// ChannelStore stores information about channels
type ChannelStore interface {
	GetChannel(name string) int
}

type SockchatServer struct {
	store ChannelStore
	http.Handler
}

func NewSockChatServer(store ChannelStore) *SockchatServer {
	s := new(SockchatServer)

	s.store = store

	router := http.NewServeMux()
	router.Handle("/channels", http.HandlerFunc(s.channelsHandler))

	s.Handler = router

	return s
}

func (p *SockchatServer) channelsHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
}
