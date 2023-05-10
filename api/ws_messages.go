package api

import (
	"encoding/json"
	"log"
)

const (
	LoginAction       = "login"
	JoinAction        = "join"
	CreateAction      = "create"
	LeaveAction       = "leave"
	SendMessageAction = "send_message"

	UserJoinedChannelEvent = "user has joined the channel"
	UserLeftChannelEvent   = "user has left the channel"
	YouLeftChannelEvent    = "you have left the channel"
	NewMessageEvent        = "new message in channel"
)

func NewSocketMessage(action string, payload any) SocketMessage {
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		log.Printf("error while marshaling payload %v", err)
	}
	return SocketMessage{Action: action, Payload: payloadBytes}
}

func NewSocketError(description string) SocketMessage {
	type ErrorDetails struct {
		Description string `json:"description"`
	}
	return NewSocketMessage(ErrInvalidRequest.Error(), ErrorDetails{Description: description})
}

type SocketMessage struct {
	Action  string          `json:"action"`
	Payload json.RawMessage `json:"payload"`
}

type LoginRequest struct {
	Nick     string `json:"nick"`
	Password string `json:"password"`
}

// For create, join & leave requests
type ChannelRequest struct {
	Name string `json:"name"`
}

type ChannelUserChangeEvent struct {
	Channel string `json:"channel"`
	Nick    string `json:"nick"`
}

// For messages sent to server
type SendMessageRequest struct {
	Channel string `json:"channel"`
	Text    string `json:"text"`
}
