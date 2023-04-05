package sockchat

import (
	"encoding/json"
	"log"
)

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
	Author  string `json:"author"`
}

func UnmarshalMessageEvent(requestBytes json.RawMessage) *MessageEvent {
	messageEvent := MessageEvent{}
	if err := json.Unmarshal(requestBytes, &messageEvent); err != nil {
		log.Printf("error while unmarshaling request for sending message: %v", err)
		return nil
	}
	return &messageEvent

}
