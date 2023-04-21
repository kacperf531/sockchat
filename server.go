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
	"github.com/kacperf531/sockchat/common"
	"github.com/kacperf531/sockchat/storage"
)

const (
	ResponseDeadline = 3 * time.Second

	// The connections that haven't logged in successfully will be disconnected after this time
	defaultTimeoutUnauthorized = 1 * time.Minute
	defaultTimeoutAuthorized   = 10 * time.Minute
)

var wsUpgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

// SockchatChannelStore manages chat channels (rooms) and dispatches messages among their members
type SockchatChannelStore interface {
	GetChannel(name string) (*Channel, error)
	CreateChannel(name string) error
	AddUserToChannel(channel string, user SockchatUserHandler) error
	RemoveUserFromChannel(channel string, user SockchatUserHandler) error
	DisconnectUser(user SockchatUserHandler)
}

// SockchatProfileStore manages DB-stored user profiles
type SockchatProfileStore interface {
	Create(ctx context.Context, u *UserProfile) error
	Edit(ctx context.Context, u *UserProfile) error
	IsAuthValid(ctx context.Context, nick, password string) bool
}

// SockchatMessageStore manages messages in ES
type SockchatMessageStore interface {
	IndexMessage(msg *common.MessageEvent) (string, error)
	GetMessagesByChannel(channel string) ([]*common.MessageEvent, error)
}

// SockchatUserManager manages user handlers that store connections and send messages to them
type SockchatUserManager interface {
	AddConnection(conn *SockChatWS, nick string)
	RemoveConnection(conn *SockChatWS)
	GetHandler(nick string) (SockchatUserHandler, bool)
}

type SockchatServer struct {
	http.Handler
	channelStore        SockchatChannelStore
	userProfiles        SockchatProfileStore
	messageStore        SockchatMessageStore
	authorizedUsers     SockchatUserManager
	timeoutUnauthorized time.Duration
	timeoutAuthorized   time.Duration
}

func NewSocketMessage(action string, payload any) SocketMessage {
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		log.Printf("error while marshaling payload %v", err)
	}
	return SocketMessage{Action: action, Payload: payloadBytes}
}

func NewErrorMessage(details string) SocketMessage {
	return NewSocketMessage(InvalidRequestEvent, map[string]string{"details": details})
}

func NewSockChatServer(channelStore SockchatChannelStore, userStore storage.UserStore, messageStore SockchatMessageStore) *SockchatServer {
	s := new(SockchatServer)
	s.channelStore = channelStore
	s.userProfiles = &ProfileService{store: userStore}
	s.messageStore = messageStore
	s.authorizedUsers = NewConnectedUsersPool(channelStore)

	s.SetTimeoutValues(defaultTimeoutAuthorized, defaultTimeoutUnauthorized)

	router := http.NewServeMux()
	router.Handle("/ws", http.HandlerFunc(s.webSocket))
	router.Handle("/register", http.HandlerFunc(s.register))
	router.Handle("/edit_profile", http.HandlerFunc(s.editProfile))
	s.Handler = router

	return s
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
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("error: %v", err)
			}
			break
		}

		if !conn.authorized {
			err := s.authorizeConnection(*receivedMsg, conn)
			if err != nil {
				conn.WriteSocketMsg(NewErrorMessage(err.Error()))
			}
			continue
		}

		conn.SetReadDeadline(time.Now().Add(s.timeoutAuthorized))
		var req any
		switch receivedMsg.Action {
		case CreateAction, JoinAction, LeaveAction:
			req, err = UnmarshalChannelRequest(receivedMsg.Payload)
		case SendMessageAction:
			req, err = UnmarshalMessageRequest(receivedMsg.Payload)
		default:
			conn.WriteSocketMsg(NewErrorMessage(InvalidRequestEvent))
			continue
		}
		if err != nil {
			log.Print("error while unmarshaling request: ", err.Error())
			conn.WriteSocketMsg(NewErrorMessage(InvalidRequestEvent))
			continue
		}
		err = conn.userHandler.MakeRequest(receivedMsg.Action, req)
		if err != nil {
			conn.WriteSocketMsg(NewErrorMessage(err.Error()))
		}
	}
}

func (s *SockchatServer) register(w http.ResponseWriter, r *http.Request) {
	userData := UserProfile{}
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
	err = s.userProfiles.Create(ctx, &userData)
	if err != nil {
		if err == common.ErrResourceConflict {
			w.WriteHeader(http.StatusConflict)
		} else {
			w.WriteHeader(http.StatusUnprocessableEntity)
		}
		return
	}
	w.WriteHeader(http.StatusCreated)
}

func (s *SockchatServer) editProfile(w http.ResponseWriter, r *http.Request) {
	userData := UserProfile{}
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
	err = s.userProfiles.Edit(ctx, &userData)
	if err != nil {
		if err == common.ErrUnauthorized {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		w.WriteHeader(http.StatusUnprocessableEntity)
		return
	}
	w.WriteHeader(http.StatusOK)

}

func (s *SockchatServer) authorizeConnection(request SocketMessage, conn *SockChatWS) error {

	ctx, cancel := context.WithTimeout(context.Background(), ResponseDeadline)
	defer cancel()

	if request.Action != LoginAction {
		return fmt.Errorf("you must log in first using " + LoginAction + " action")
	}
	req := UnmarshalLoginRequest(request.Payload)
	if !s.userProfiles.IsAuthValid(ctx, req.Nick, req.Password) {
		return fmt.Errorf("login rejected: invalid credentials")
	}

	conn.authorized = true
	s.authorizedUsers.AddConnection(conn, req.Nick)
	conn.WriteSocketMsg(NewSocketMessage("logged_in:"+req.Nick, "{}"))
	conn.SetReadDeadline(time.Now().Add(s.timeoutAuthorized))
	return nil
}

func (s *SockchatServer) shutConnection(conn *SockChatWS) {
	if conn.authorized {
		s.authorizedUsers.RemoveConnection(conn)
	}
	conn.Close()
}

type SockChatWS struct {
	*websocket.Conn
	writeLock   sync.Mutex
	readLock    sync.Mutex
	authorized  bool
	userHandler SockchatUserHandler
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
