package sockchat

import "encoding/json"

type SocketMessage struct {
	Action  string          `json:"action"`
	Payload json.RawMessage `json:"payload"`
}

type Channel struct {
	Name  string `json:"name"`
	Users []*SockChatWS
}

type MessageEvent struct {
	Text    string `json:"text"`
	Channel string `json:"channel"`
}
