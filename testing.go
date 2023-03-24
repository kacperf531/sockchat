package sockchat

import (
	"encoding/json"
	"testing"

	"github.com/gorilla/websocket"
)

func mustDialWS(t *testing.T, url string) *websocket.Conn {
	ws, _, err := websocket.DefaultDialer.Dial(url, nil)

	if err != nil {
		t.Fatalf("could not open a ws connection on %s %v", url, err)
	}

	return ws
}

func mustWriteWSMessage(t testing.TB, conn *websocket.Conn, message SocketMessage) {
	t.Helper()
	payloadBytes, err := json.Marshal(message)
	if err != nil {
		t.Fatalf("could not marshal message before sending to the server %v", err)
	}
	if err := conn.WriteMessage(websocket.TextMessage, payloadBytes); err != nil {
		t.Fatalf("could not send message over ws connection %v", err)
	}
}

func mustReadWSMessage(t testing.TB, conn *websocket.Conn) SocketMessage {
	t.Helper()
	receivedMessage := &SocketMessage{}
	if err := conn.ReadJSON(receivedMessage); err != nil {
		t.Fatalf("could not parse message coming from ws %v", err)
	}
	return *receivedMessage
}