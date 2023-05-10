package test_utils

import (
	"encoding/json"
	"log"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/kacperf531/sockchat/api"
)

// Test WS client
type TestWS struct {
	*websocket.Conn
	MessageStash chan api.SocketMessage
	writeLock    sync.Mutex
}

// Connects to provided URL and returns initialized TestWS
func NewTestWS(t *testing.T, url string) *TestWS {
	conn, _, err := websocket.DefaultDialer.Dial(url, nil)
	if err != nil {
		t.Fatalf("could not open a ws connection on %s %v", url, err)
	}
	ws := TestWS{Conn: conn, MessageStash: make(chan api.SocketMessage)}
	go ws.readIncomingMessages()
	return &ws
}

func (ws *TestWS) Write(t testing.TB, message api.SocketMessage) {
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
		receivedMessage := &api.SocketMessage{}
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
