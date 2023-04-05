package sockchat

import (
	"encoding/json"
	"log"
)

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

type SendMessageRequest struct {
	Channel string `json:"channel"`
	Text    string `json:"text"`
}

// For create & edit profile web API requests
type UserRequest struct {
	Nick        string `json:"nick"`
	Password    string `json:"password"`
	Description string `json:"description"`
}

func UnmarshalChannelRequest(requestBytes json.RawMessage) *ChannelRequest {
	channelRequest := ChannelRequest{}
	if err := json.Unmarshal(requestBytes, &channelRequest); err != nil {
		log.Printf("error while unmarshaling request for joining channel: %v", err)
		return nil
	}
	return &channelRequest

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
