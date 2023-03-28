package sockchat

import "encoding/json"

type SocketMessage struct {
	Action  string          `json:"action"`
	Payload json.RawMessage `json:"payload"`
}

type Channel struct {
	Users map[*SockChatWS]bool
}

type MessageEvent struct {
	Text    string `json:"text"`
	Channel string `json:"channel"`
}
