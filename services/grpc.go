package services

import (
	"context"
	"fmt"
	"log"
	"net"
	"strings"
	"time"

	"github.com/kacperf531/sockchat/api"
	pb "github.com/kacperf531/sockchat/protobuf"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	emptypb "google.golang.org/protobuf/types/known/emptypb"
)

var GRPCCodes = map[error]codes.Code{
	api.ErrNickAlreadyUsed:       codes.AlreadyExists,
	api.ErrNickRequired:          codes.InvalidArgument,
	api.ErrPasswordRequired:      codes.InvalidArgument,
	api.ErrInvalidRequest:        codes.InvalidArgument,
	api.ErrMetadataNotProvided:   codes.InvalidArgument,
	api.ErrChannelNotFound:       codes.NotFound,
	api.ErrInternal:              codes.Internal,
	api.ErrUnauthorized:          codes.Unauthenticated,
	api.ErrAuthorizationRequired: codes.Unauthenticated,
	api.ErrBasicTokenRequired:    codes.Unauthenticated,
	api.ErrCouldNotDecodeToken:   codes.Unauthenticated,
	api.ErrInvalidGroupBy:        codes.InvalidArgument,
	api.ErrInvalidDateFormat:     codes.InvalidArgument,
	api.ErrFromMissing:           codes.InvalidArgument,
	api.ErrToMissing:             codes.InvalidArgument,
	api.ErrMaxReportSizeExceeded: codes.OutOfRange,
}

func NewGRPCError(err error) error {
	code, found := GRPCCodes[err]
	if !found {
		log.Printf("GRPC error code not found for error: %v", err)
		return status.Errorf(codes.Internal, api.ErrInternal.Error())
	}
	return status.Errorf(code, err.Error())
}

type GrpcAPI struct {
	pb.UnimplementedSockchatServer
	core        *SockchatCoreService
	authService *SockchatAuthService
	userReports api.SockchatReportsService
}

func NewSockchatGRPCServer(core *SockchatCoreService, authService *SockchatAuthService, reportsService api.SockchatReportsService) *grpc.Server {
	grpcApi := &GrpcAPI{core: core, authService: authService, userReports: reportsService}
	server := grpc.NewServer(grpc.UnaryInterceptor(grpcApi.AuthInterceptor))
	pb.RegisterSockchatServer(server, grpcApi)
	return server
}

var methodAuthorizationRequired = map[string]bool{
	"RegisterProfile":       false,
	"GetProfile":            true,
	"EditProfile":           true,
	"GetChannelHistory":     true,
	"GetUserActivityReport": true,
}

func isProtected(fullMethodName string) bool {
	methodName := strings.TrimPrefix(fullMethodName, "/sockchat.Sockchat/")
	requires_authorization, exists := methodAuthorizationRequired[methodName]
	if !exists {
		log.Fatalf("gRPC method %s not defined in methodAuthorizationRequired map", methodName)
	}
	return requires_authorization
}

func (s *GrpcAPI) AuthInterceptor(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
	if isProtected(info.FullMethod) {
		token, err := tokenFromCtx(ctx)
		if err != nil {
			return nil, NewGRPCError(err)
		}
		authenticationOK, err := s.authService.AuthenticateFromBasicToken(ctx, token)
		if err != nil {
			return nil, NewGRPCError(err)
		}
		if !authenticationOK {
			err := api.ErrUnauthorized
			return nil, NewGRPCError(err)
		}
	}
	return handler(ctx, req)
}

func (s *GrpcAPI) RegisterProfile(ctx context.Context, in *pb.RegisterProfileRequest) (*emptypb.Empty, error) {
	_, err := s.core.RegisterProfile(api.CreateProfileRequestFromProto(in), ctx)
	if err != nil {
		return nil, NewGRPCError(err)
	}
	return &emptypb.Empty{}, nil
}

func (s *GrpcAPI) GetProfile(ctx context.Context, in *pb.GetProfileRequest) (*pb.Profile, error) {
	res, err := s.core.GetProfile(api.GetProfileRequestFromProto(in), ctx)
	if err != nil {
		return nil, NewGRPCError(err)
	}
	return api.ProfileToProto(res), nil
}

func (s *GrpcAPI) EditProfile(ctx context.Context, in *pb.EditProfileRequest) (*emptypb.Empty, error) {
	token, err := tokenFromCtx(ctx)
	if err != nil {
		return nil, NewGRPCError(err)
	}
	authData, err := decodeToken(token)
	if err != nil {
		return nil, NewGRPCError(err)
	}
	_, err = s.core.EditProfile(&EditProfileWrapper{Nick: authData.Username, Request: api.EditProfileRequestFromProto(in)}, ctx)
	if err != nil {
		return nil, NewGRPCError(err)
	}
	return &emptypb.Empty{}, nil
}

func (s *GrpcAPI) GetChannelHistory(ctx context.Context, in *pb.GetChannelHistoryRequest) (*pb.GetChannelHistoryResponse, error) {
	res, err := s.core.GetChannelHistory(api.GetChannelHistoryRequestFromProto(in), ctx)
	if err != nil {
		return nil, NewGRPCError(err)
	}
	return &pb.GetChannelHistoryResponse{
		Messages: api.ChannelHistoryToProto(res),
	}, nil
}

func (s *GrpcAPI) GetUserActivityReport(ctx context.Context, in *pb.GetUserActivityReportRequest) (*pb.GetUserActivityReportResponse, error) {
	err := validateGetUserActivityReportOpts(in)
	if err != nil {
		return nil, NewGRPCError(err)
	}
	parsedIn, err := parseReportsDate(in.From)
	if err != nil {
		return nil, NewGRPCError(err)
	}
	parsedOut, err := parseReportsDate(in.To)
	if err != nil {
		return nil, NewGRPCError(err)
	}
	opts := &api.UserActivityReportOptions{
		Author:  in.Author,
		GroupBy: api.GroupBy(in.GroupBy),
		From:    *parsedIn,
		To:      *parsedOut,
	}
	res, err := s.userReports.GetUserActivityReport(opts)
	if err != nil {
		return nil, err
	}
	return api.UserActivityReportToProto(res), nil
}

func tokenFromCtx(ctx context.Context) (string, error) {
	meta, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return "", api.ErrMetadataNotProvided
	}
	if len(meta["authorization"]) != 1 {
		return "", api.ErrAuthorizationRequired
	}
	return meta["authorization"][0], nil
}

func ServeGRPC(server *grpc.Server, grpcPort int) {
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", grpcPort))
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	go func() {
		if err := server.Serve(lis); err != nil {
			log.Fatalf("failed to serve gRPC: %v", err)
		}
	}()
}

func validateGetUserActivityReportOpts(in *pb.GetUserActivityReportRequest) error {
	if !isGroupByOK(in.GroupBy) {
		return api.ErrInvalidGroupBy
	}
	if in.From == "" {
		return api.ErrFromMissing
	}
	if in.To == "" {
		return api.ErrToMissing
	}
	return nil
}

func isGroupByOK(groupBy string) bool {
	allowedGroupBy := []api.GroupBy{api.GroupByHour, api.GroupByDay, api.GroupByMinute, ""}
	for _, b := range allowedGroupBy {
		if groupBy == string(b) {
			return true
		}
	}
	return false
}

func parseReportsDate(date string) (*time.Time, error) {
	if date == "" {
		return &time.Time{}, nil
	}
	parsed, err := time.Parse(api.ReportsDateLayout, date)
	if err != nil {
		return nil, api.ErrInvalidDateFormat
	}
	return &parsed, nil
}
