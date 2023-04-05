package sockchat

import (
	"encoding/json"
	"log"
	"strings"
	"testing"

	"github.com/gorilla/websocket"
)

const (
	ValidUserNick         = "SpecialTestUser"
	ValidUserPassword     = "foo420"
	ValidUserPasswordHash = "$2a$10$Xl002E7Vj5qM1RHMiM06KOCHofpLcPTIj7LeyZgTf62txoOBvoyia"
)

// Test WS client
type TestWS struct {
	*websocket.Conn
	MessageStash chan SocketMessage
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
	if err := ws.WriteMessage(websocket.TextMessage, payloadBytes); err != nil {
		t.Fatalf("could not send message over ws connection %v", err)
	}
}

func (ws *TestWS) readIncomingMessages() {
	for {
		receivedMessage := &SocketMessage{}
		if err := ws.ReadJSON(receivedMessage); err != nil {
			log.Printf("Test websocket read interrupted due to error: %v; closing now", err)
			ws.Close()
		}
		ws.MessageStash <- *receivedMessage
	}
}

func GetWsURL(serverURL string) string {
	return "ws" + strings.TrimPrefix(serverURL, "http") + "/ws"
}
