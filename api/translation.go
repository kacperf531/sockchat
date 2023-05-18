package api

import (
	pb "github.com/kacperf531/sockchat/protobuf"
)

func CreateProfileRequestFromProto(in *pb.RegisterProfileRequest) *CreateProfileRequest {
	return &CreateProfileRequest{Nick: in.Nick, Password: in.Password}
}

func GetProfileRequestFromProto(in *pb.GetProfileRequest) *GetProfileRequest {
	return &GetProfileRequest{Nick: in.Nick}
}

func EditProfileRequestFromProto(in *pb.EditProfileRequest) *EditProfileRequest {
	return &EditProfileRequest{Description: in.Description}
}

func GetChannelHistoryRequestFromProto(in *pb.GetChannelHistoryRequest) *GetChannelHistoryRequest {
	return &GetChannelHistoryRequest{Channel: in.Channel, Search: in.Search}
}

func MessageEventToProto(in *MessageEvent) *pb.ChatMessage {
	return &pb.ChatMessage{
		Channel:   in.Channel,
		Text:      in.Text,
		Author:    in.Author,
		Timestamp: in.Timestamp,
	}
}

func ChannelHistoryToProto(in ChannelHistory) []*pb.ChatMessage {
	out := make([]*pb.ChatMessage, len(in))
	for i, v := range in {
		out[i] = MessageEventToProto(v)
	}
	return out
}

func ProfileToProto(in *PublicProfile) *pb.Profile {
	return &pb.Profile{
		Nick:        in.Nick,
		Description: in.Description,
	}
}

func UserActivityReportToProto(in *UserActivityReport) *pb.GetUserActivityReportResponse {
	response := &pb.GetUserActivityReportResponse{
		Channels: make(map[string]*pb.ChannelData),
		From:     in.From.Format(ReportsDateLayout),
		To:       in.To.Format(ReportsDateLayout),
	}

	for channelID, activity := range (*in).ChannelActivity {
		response.Channels[channelID] = &pb.ChannelData{
			TotalMessages:            int32(activity.TotalMessages),
			MessageCountDistribution: messageCountDistributionToProto(activity.MessageCountDistribution),
		}
	}

	return response
}

func messageCountDistributionToProto(in []DistributionEntry) []*pb.MessageCount {
	out := make([]*pb.MessageCount, len(in))
	for i, v := range in {
		out[i] = &pb.MessageCount{
			PeriodStart:      v.PeriodStart,
			MessagesInPeriod: int32(v.MessagesInPeriod),
		}
	}
	return out
}
