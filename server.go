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
	"github.com/redis/go-redis/v9"
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
	CreateChannel(name string) error
	AddUserToChannel(channel string, user SockchatUserHandler) error
	RemoveUserFromChannel(channel string, user SockchatUserHandler) error
	MessageChannel(msg *common.MessageEvent) error
	DisconnectUser(user SockchatUserHandler)
	IsUserPresentIn(user SockchatUserHandler, channel string) bool
	ChannelExists(name string) bool
}

// SockchatProfileStore manages DB-stored user profiles
type SockchatProfileStore interface {
	Create(ctx context.Context, u *CreateProfileRequest) error
	Edit(ctx context.Context, nick string, u *EditProfileRequest) error
	IsAuthValid(ctx context.Context, nick, password string) bool
	GetProfile(ctx context.Context, nick string) (*common.PublicProfile, error)
}

// SockchatMessageStore manages messages in ES
type SockchatMessageStore interface {
	IndexMessage(msg *common.MessageEvent) (string, error)
	FindMessages(channel, query string) ([]*common.MessageEvent, error)
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

func NewSockChatServer(channelStore SockchatChannelStore, userStore storage.UserStore, messageStore SockchatMessageStore, userCache *redis.Client) *SockchatServer {
	s := new(SockchatServer)
	s.channelStore = channelStore
	s.userProfiles = &ProfileService{store: userStore, cache: userCache}
	s.messageStore = messageStore
	s.authorizedUsers = NewConnectedUsersPool(channelStore)

	s.SetTimeoutValues(defaultTimeoutAuthorized, defaultTimeoutUnauthorized)

	router := http.NewServeMux()
	authenticate := newAuthMiddleware(s.userProfiles)
	router.Handle("/ws", http.HandlerFunc(s.webSocket))
	router.Handle("/register", http.HandlerFunc(s.register))
	router.Handle("/edit_profile", authenticate(s.editProfile))
	router.Handle("/history", authenticate(s.getChannelHistory))
	router.Handle("/profile", authenticate(s.getProfile))
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
			break
		}

		if conn.authorized {
			conn.SetReadDeadline(time.Now().Add(s.timeoutAuthorized))
			req, err := parseWebsocketMessage(*receivedMsg)
			if err != nil {
				conn.WriteSocketMsg(NewErrorMessage(err.Error()))
				continue
			}
			err = conn.userHandler.MakeRequest(receivedMsg.Action, req)
			if err != nil {
				conn.WriteSocketMsg(NewErrorMessage(err.Error()))
			}
			continue
		}

		err = s.authorizeWebsocket(*receivedMsg, conn)
		if err != nil {
			conn.WriteSocketMsg(NewErrorMessage(err.Error()))
		}
		conn.SetReadDeadline(time.Now().Add(s.timeoutAuthorized))
	}
}

func (s *SockchatServer) register(w http.ResponseWriter, r *http.Request) {
	userData := readCreateProfileRequest(w, r)
	ctx, cancel := context.WithTimeout(r.Context(), ResponseDeadline)
	defer cancel()
	err := s.userProfiles.Create(ctx, userData)
	if err != nil {
		if err == common.ErrResourceConflict {
			writeJsonHttpResponse(w, http.StatusConflict, NewErrorMessage("user already exists"))
		} else {
			writeJsonHttpResponse(w, http.StatusUnprocessableEntity, NewErrorMessage(err.Error()))
		}
		return
	}
	writeJsonHttpResponse(w, http.StatusCreated, "")
}

func (s *SockchatServer) getProfile(w http.ResponseWriter, r *http.Request) {
	// No authorization required for getting user profiles (public info)
	ctx, cancel := context.WithTimeout(r.Context(), ResponseDeadline)
	defer cancel()
	profile, err := s.userProfiles.GetProfile(ctx, r.URL.Query().Get("nick"))
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}
	profileBytes, err := json.Marshal(profile)
	if err != nil {
		log.Printf("error marshaling profile: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.Write(profileBytes)
}

func (s *SockchatServer) editProfile(w http.ResponseWriter, r *http.Request) {
	userData := readEditProfileRequest(w, r)
	ctx, cancel := context.WithTimeout(r.Context(), ResponseDeadline)
	defer cancel()
	username, _, _ := r.BasicAuth()
	err := s.userProfiles.Edit(ctx, username, userData)
	if err != nil {
		writeJsonHttpResponse(w, http.StatusUnprocessableEntity, NewErrorMessage(err.Error()))
		return
	}
	writeJsonHttpResponse(w, http.StatusOK, "")
}

func (s *SockchatServer) getChannelHistory(w http.ResponseWriter, r *http.Request) {
	channelName := r.URL.Query().Get("channel")
	if !s.channelStore.ChannelExists(channelName) {
		writeJsonHttpResponse(w, http.StatusNotFound, NewErrorMessage("channel not found"))
		return
	}
	soughtPhrase := r.URL.Query().Get("search")
	messages, err := s.messageStore.FindMessages(channelName, soughtPhrase)
	if err != nil {
		writeJsonHttpResponse(w, http.StatusInternalServerError, NewErrorMessage("Server error, please try again later"))
		return
	}
	writeJsonHttpResponse(w, http.StatusOK, messages)
}

func (s *SockchatServer) authorizeWebsocket(request SocketMessage, conn *SockChatWS) error {
	if request.Action == LoginAction {
		u, err := s.loginUser(request)
		if err == nil {
			conn.authorized = true
			s.authorizedUsers.AddConnection(conn, u.Nick)
			conn.WriteSocketMsg(NewSocketMessage("logged_in:"+u.Nick, "{}"))
			conn.SetReadDeadline(time.Now().Add(s.timeoutAuthorized))
		}
		return err
	}
	return fmt.Errorf("you must log in first using " + LoginAction + " action")
}

func (s *SockchatServer) loginUser(request SocketMessage) (*common.PublicProfile, error) {
	ctx, cancel := context.WithTimeout(context.Background(), ResponseDeadline)
	defer cancel()
	req := UnmarshalLoginRequest(request.Payload)
	if s.userProfiles.IsAuthValid(ctx, req.Nick, req.Password) {
		return &common.PublicProfile{Nick: req.Nick}, nil
	}
	return nil, fmt.Errorf("login rejected: invalid credentials")
}

func (s *SockchatServer) shutConnection(conn *SockChatWS) {
	if conn.authorized {
		s.authorizedUsers.RemoveConnection(conn)
	}
	conn.Close()
}

func newAuthMiddleware(profiles SockchatProfileStore) func(next http.HandlerFunc) http.HandlerFunc {
	return func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			username, password, ok := r.BasicAuth()
			if !ok {
				writeJsonHttpResponse(w, http.StatusUnauthorized, NewErrorMessage("unauthorized"))
				return
			}
			ctx, cancel := context.WithTimeout(r.Context(), ResponseDeadline)
			defer cancel()
			if !profiles.IsAuthValid(ctx, username, password) {
				writeJsonHttpResponse(w, http.StatusUnauthorized, NewErrorMessage("unauthorized"))
				return
			}
			next(w, r)
		}
	}
}

func writeJsonHttpResponse(w http.ResponseWriter, statusCode int, data interface{}) error {
	output, err := json.Marshal(data)
	if err != nil {
		return err
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	_, err = w.Write(output)
	if err != nil {
		log.Print(err)
		return nil
	}
	return nil
}

func readCreateProfileRequest(w http.ResponseWriter, r *http.Request) *CreateProfileRequest {
	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		writeJsonHttpResponse(w, http.StatusBadRequest, NewErrorMessage("invalid request"))
		return nil
	}
	return UnmarshalCreateProfileRequest(bodyBytes)
}

func readEditProfileRequest(w http.ResponseWriter, r *http.Request) *EditProfileRequest {
	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		writeJsonHttpResponse(w, http.StatusBadRequest, NewErrorMessage("invalid request"))
		return nil
	}
	return UnmarshalEditProfileRequest(bodyBytes)
}

func parseWebsocketMessage(msg SocketMessage) (interface{}, error) {
	switch msg.Action {
	case CreateAction, JoinAction, LeaveAction:
		return UnmarshalChannelRequest(msg.Payload)
	case SendMessageAction:
		return UnmarshalMessageRequest(msg.Payload)
	default:
		return nil, fmt.Errorf(InvalidRequestEvent)
	}
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
	if err != nil {
		if os.IsTimeout(err) {
			w.WriteSocketMsg(NewSocketMessage("connection_timed_out", "{}"))
		}
		if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
			log.Printf("error while reading messages from websocket: %v", err)
		}
		return nil, err
	}
	msg := &SocketMessage{}
	json.Unmarshal(msgBytes, &msg)
	return msg, nil
}

func (w *SockChatWS) WriteSocketMsg(m SocketMessage) {
	w.writeLock.Lock()
	defer w.writeLock.Unlock()
	err := w.WriteJSON(m)
	if err != nil {
		log.Printf("Error writing message %s with payload %s to websocket: %v", m.Action, string(m.Payload), err)
	}
}
