package services

import (
	"context"

	"github.com/kacperf531/sockchat/api"
)

type SockchatCoreService struct {
	UserProfiles   api.SockchatProfileStore
	Messages       api.SockchatMessageStore
	ChatChannels   api.SockchatChannelStore
	ConnectedUsers api.SockchatUserManager
}

type EditProfileWrapper struct {
	Nick    string
	Request *api.EditProfileRequest
}

func (s *SockchatCoreService) RegisterProfile(req *api.CreateProfileRequest, ctx context.Context) (*api.EmptyMessage, error) {
	if req.Nick == "" {
		return nil, api.ErrNickRequired
	}
	if req.Password == "" {
		return nil, api.ErrPasswordRequired
	}

	err := s.UserProfiles.Create(ctx, req)
	if err != nil {
		return nil, err
	}

	return &api.EmptyMessage{}, nil
}

func (s *SockchatCoreService) GetProfile(req *api.GetProfileRequest, ctx context.Context) (*api.PublicProfile, error) {
	profile, err := s.UserProfiles.GetProfile(ctx, req.Nick)
	if err != nil {
		return nil, err
	}
	return profile, nil
}

func (s *SockchatCoreService) EditProfile(req *EditProfileWrapper, ctx context.Context) (*api.EmptyMessage, error) {
	err := s.UserProfiles.Edit(ctx, req.Nick, req.Request)
	if err != nil {
		return nil, err
	}
	return &api.EmptyMessage{}, nil
}

func (s *SockchatCoreService) GetChannelHistory(req *api.GetChannelHistoryRequest, ctx context.Context) (api.ChannelHistory, error) {
	if !s.ChatChannels.ChannelExists(req.Channel) {
		return nil, api.ErrChannelNotFound
	}
	return s.Messages.FindMessages(ctx, req.Channel, req.Search)
}
