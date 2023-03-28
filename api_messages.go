package sockchat

import (
	"encoding/json"
	"log"
)

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
