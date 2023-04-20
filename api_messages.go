package sockchat

import (
	"encoding/json"
	"fmt"
	"log"

	"github.com/kacperf531/sockchat/common"
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
	InvalidRequestEvent    = "request is invalid"
)

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

// For create & edit profile web API requests
type UserProfile struct {
	Nick        string `json:"nick"`
	Password    string `json:"password"`
	Description string `json:"description"`
}

func UnmarshalChannelRequest(requestBytes json.RawMessage) (*ChannelRequest, error) {
	channelRequest := ChannelRequest{}
	if err := json.Unmarshal(requestBytes, &channelRequest); err != nil {
		log.Printf("error while unmarshaling request for joining channel: %v", err)
		return nil, fmt.Errorf(InvalidRequestEvent)
	}
	return &channelRequest, nil
}

func UnmarshalLoginRequest(requestBytes json.RawMessage) *LoginRequest {
	loginRequest := LoginRequest{}
	if err := json.Unmarshal(requestBytes, &loginRequest); err != nil {
		log.Printf("error while unmarshaling request for login: %v", err)
		return nil
	}
	return &loginRequest
}

func UnmarshalChannelUserChangeEvent(requestBytes json.RawMessage) *ChannelUserChangeEvent {
	channelUserChangeEvent := ChannelUserChangeEvent{}
	if err := json.Unmarshal(requestBytes, &channelUserChangeEvent); err != nil {
		log.Printf("error while unmarshaling request for channel user change event: %v", err)
		return nil
	}
	return &channelUserChangeEvent
}

func UnmarshalMessageRequest(requestBytes json.RawMessage) (*SendMessageRequest, error) {
	messageRequest := SendMessageRequest{}
	if err := json.Unmarshal(requestBytes, &messageRequest); err != nil {
		log.Printf("error while unmarshaling request for message: %v", err)
		return nil, fmt.Errorf(InvalidRequestEvent)
	}
	return &messageRequest, nil
}

func UnmarshalMessageEvent(requestBytes json.RawMessage) *common.MessageEvent {
	messageEvent := common.MessageEvent{}
	if err := json.Unmarshal(requestBytes, &messageEvent); err != nil {
		log.Printf("error while unmarshaling request for sending message: %v", err)
		return nil
	}
	return &messageEvent
}
