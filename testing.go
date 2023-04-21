package sockchat

import (
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/kacperf531/sockchat/common"
)

const (
	ValidUserNick         = "SpecialTestUser"
	ValidUser2Nick        = "VerySpecialTestUser"
	ValidUserPassword     = "foo420"
	ValidUserPasswordHash = "$2a$10$Xl002E7Vj5qM1RHMiM06KOCHofpLcPTIj7LeyZgTf62txoOBvoyia"
	ChannelWithUser       = "channel_with_user"
	ChannelWithoutUser    = "channel_without_user"
)

// Test WS client
type TestWS struct {
	*websocket.Conn
	MessageStash chan SocketMessage
	writeLock    sync.Mutex
}

// Connects to provided URL and returns initialized TestWS
func NewTestWS(t *testing.T, url string) *TestWS {
	conn, _, err := websocket.DefaultDialer.Dial(url, nil)
	if err != nil {
		t.Fatalf("could not open a ws connection on %s %v", url, err)
	}
	ws := TestWS{Conn: conn, MessageStash: make(chan SocketMessage)}
	go ws.readIncomingMessages()
	return &ws
}

func (ws *TestWS) Write(t testing.TB, message SocketMessage) {
	payloadBytes, err := json.Marshal(message)
	if err != nil {
		t.Fatalf("could not marshal message before sending to the server %v", err)
	}
	ws.writeLock.Lock()
	defer ws.writeLock.Unlock()
	if err := ws.WriteMessage(websocket.TextMessage, payloadBytes); err != nil {
		t.Fatalf("could not send message over ws connection %v", err)
	}
}

func (ws *TestWS) readIncomingMessages() {
	for {
		receivedMessage := &SocketMessage{}
		if err := ws.ReadJSON(receivedMessage); err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("Test websocket read interrupted due to error: %v; closing now", err)
			}
			ws.Close()
		}
		ws.MessageStash <- *receivedMessage
	}
}

func (ws *TestWS) AssertEventReceivedWithin(t testing.TB, eventAction string, d time.Duration) {
	t.Helper()

	done := make(chan struct{}, 1)
	go func() {
		for {
			received := <-ws.MessageStash
			if received.Action == eventAction {
				done <- struct{}{}
				break
			}
		}
	}()

	select {
	case <-time.After(d):
		t.Errorf("assertion failed - timed out waiting for websocket event: %s", eventAction)
	case <-done:
	}
}

func GetWsURL(serverURL string) string {
	return "ws" + strings.TrimPrefix(serverURL, "http") + "/ws"
}

// StubChannelStore implements ChannelStore for testing purposes
type StubChannelStore struct {
	Channels map[string]*Channel
}

func (store *StubChannelStore) CreateChannel(name string) error {
	if name == "already_exists" {
		return fmt.Errorf("channel `%s` already exists", name)
	}
	return nil
}

func (store *StubChannelStore) DisconnectUser(user SockchatUserHandler) {
}

func (store *StubChannelStore) GetChannel(name string) (*Channel, error) {
	return &Channel{}, nil
}

func (store *StubChannelStore) AddUserToChannel(name string, user SockchatUserHandler) error {
	if name == ChannelWithUser {
		return ErrUserAlreadyInChannel
	}
	user.Write(NewSocketMessage(UserJoinedChannelEvent, ChannelUserChangeEvent{name, user.getNick()}))
	return nil
}

func (store *StubChannelStore) RemoveUserFromChannel(name string, user SockchatUserHandler) error {
	if name == ChannelWithoutUser {
		return ErrUserNotInChannel
	}
	user.Write(NewSocketMessage(UserLeftChannelEvent, ChannelUserChangeEvent{name, user.getNick()}))
	return nil
}

type messageStoreSpy struct {
	indexMessageCalls int
}

func (s *messageStoreSpy) GetMessagesByChannel(channel string) ([]*common.MessageEvent, error) {
	return nil, nil
}

func (s *messageStoreSpy) IndexMessage(*common.MessageEvent) (string, error) {
	s.indexMessageCalls++
	return "", nil
}

type messageStoreStub struct {
	messages []*common.MessageEvent
	lock     sync.Mutex
}

func (s *messageStoreStub) GetMessagesByChannel(channel string) ([]*common.MessageEvent, error) {
	s.lock.Lock()
	defer s.lock.Unlock()
	// simplified stub - channel filtering & sorting logic is in ES
	return s.messages, nil
}

func (s *messageStoreStub) IndexMessage(*common.MessageEvent) (string, error) {
	s.lock.Lock()
	defer s.lock.Unlock()
	s.messages = append(s.messages, &common.MessageEvent{})
	return "", nil
}
