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
	Name     string `json:"name"`
	UserName string `json:"user"`
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
