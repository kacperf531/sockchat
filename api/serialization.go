package api

import (
	"encoding/json"
)

func UnmarshalChannelRequest(requestBytes json.RawMessage) (*ChannelRequest, error) {
	channelRequest := ChannelRequest{}
	if err := json.Unmarshal(requestBytes, &channelRequest); err != nil {
		return nil, ErrInvalidRequest
	}
	return &channelRequest, nil
}

func UnmarshalLoginRequest(requestBytes json.RawMessage) (*LoginRequest, error) {
	loginRequest := LoginRequest{}
	if err := json.Unmarshal(requestBytes, &loginRequest); err != nil {
		return nil, err
	}
	return &loginRequest, nil
}

func UnmarshalChannelUserChangeEvent(requestBytes json.RawMessage) (*ChannelUserChangeEvent, error) {
	channelUserChangeEvent := ChannelUserChangeEvent{}
	if err := json.Unmarshal(requestBytes, &channelUserChangeEvent); err != nil {
		return nil, err
	}
	return &channelUserChangeEvent, nil
}

func UnmarshalMessageRequest(requestBytes json.RawMessage) (*SendMessageRequest, error) {
	messageRequest := SendMessageRequest{}
	if err := json.Unmarshal(requestBytes, &messageRequest); err != nil {
		return nil, ErrInvalidRequest
	}
	return &messageRequest, nil
}

func UnmarshalMessageEvent(requestBytes json.RawMessage) (*MessageEvent, error) {
	messageEvent := MessageEvent{}
	if err := json.Unmarshal(requestBytes, &messageEvent); err != nil {
		return nil, err
	}
	return &messageEvent, nil
}

func UnmarshalCreateProfileRequest(requestBytes json.RawMessage) (*CreateProfileRequest, error) {
	createProfileRequest := CreateProfileRequest{}
	if err := json.Unmarshal(requestBytes, &createProfileRequest); err != nil {
		return nil, err
	}
	return &createProfileRequest, nil
}

func UnmarshalEditProfileRequest(requestBytes json.RawMessage) (*EditProfileRequest, error) {
	editProfileRequest := EditProfileRequest{}
	if err := json.Unmarshal(requestBytes, &editProfileRequest); err != nil {
		return nil, err
	}
	return &editProfileRequest, nil
}
