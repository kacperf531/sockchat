package services_test

import (
	"context"
	"encoding/base64"
	"flag"
	"fmt"
	"log"
	"net"
	"testing"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	"github.com/kacperf531/sockchat"
	"github.com/kacperf531/sockchat/api"
	pb "github.com/kacperf531/sockchat/protobuf"
	"github.com/kacperf531/sockchat/services"
	"github.com/kacperf531/sockchat/test_utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var (
	addr = flag.String("addr", "localhost:50052", "the address to connect to")
	port = flag.Int("port", 50052, "The server port")
)

func TestSockChatGRPC(t *testing.T) {
	validToken := "Basic " + base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("%s:%s", test_utils.ValidUserNick, test_utils.ValidUserPassword)))
	invalidToken := "Basic rhweufdsf420"
	sampleMessage := api.MessageEvent{Text: "foo", Channel: "bar", Author: "baz"}
	messageStore := &test_utils.StubMessageStore{Messages: api.ChannelHistory{&sampleMessage}}

	// Set up test grpc server
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", *port))
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	userProfiles := &sockchat.ProfileService{Store: &test_utils.UserStoreDouble{}, Cache: test_utils.TestingRedisClient}
	core := &services.SockchatCoreService{UserProfiles: userProfiles, ChatChannels: &test_utils.StubChannelStore{}, Messages: messageStore}
	server := services.NewSockchatGRPCServer(core, &services.SockchatAuthService{UserProfiles: userProfiles})
	go func() {
		if err := server.Serve(lis); err != nil {
			log.Fatalf("failed to serve: %v", err)
		}
	}()

	// Set up a connection to the server.
	conn, err := grpc.Dial(*addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		t.Errorf("could not connect: %v", err)
	}
	defer conn.Close()
	client := pb.NewSockchatClient(conn)
	ctx := metadata.NewOutgoingContext(context.Background(), metadata.Pairs("authorization", invalidToken))

	t.Run("can register over grpc", func(t *testing.T) {
		_, err := client.RegisterProfile(ctx, &pb.RegisterProfileRequest{Nick: test_utils.ValidUserNick, Password: test_utils.ValidUserPassword})
		require.NoError(t, err)
	})

	t.Run("returns error for unauthorized request to channel history", func(t *testing.T) {
		_, err := client.GetChannelHistory(ctx, &pb.GetChannelHistoryRequest{Channel: test_utils.ChannelWithUser})
		require.ErrorContains(t, err, api.ErrBasicTokenRequired.Error())
		assert.Equal(t, codes.Unauthenticated, status.Code(err))
	})

	t.Run("returns history for authorized request", func(t *testing.T) {
		ctx := metadata.NewOutgoingContext(context.Background(), metadata.Pairs("authorization", validToken))
		resp, err := client.GetChannelHistory(ctx, &pb.GetChannelHistoryRequest{Channel: test_utils.ChannelWithUser})
		require.NoError(t, err)
		require.Len(t, resp.Messages, 1)
		require.Equal(t, resp.Messages[0].Text, sampleMessage.Text)
	})

	t.Run("returns error for unauthorized request to edit profile", func(t *testing.T) {
		_, err := client.EditProfile(ctx, &pb.EditProfileRequest{Description: "foo"})
		require.ErrorContains(t, err, api.ErrBasicTokenRequired.Error())
		assert.Equal(t, codes.Unauthenticated, status.Code(err))
	})

	t.Run("returns error for unauthorized request to get profile", func(t *testing.T) {
		_, err := client.GetProfile(ctx, &pb.GetProfileRequest{Nick: test_utils.ValidUserNick})
		require.ErrorContains(t, err, api.ErrBasicTokenRequired.Error())
		assert.Equal(t, codes.Unauthenticated, status.Code(err))
	})

	t.Run("returns correct error code for empty call to get profile", func(t *testing.T) {
		ctx := metadata.NewOutgoingContext(context.Background(), metadata.Pairs("authorization", validToken))
		_, err := client.GetProfile(ctx, &pb.GetProfileRequest{})
		require.ErrorContains(t, err, api.ErrNickRequired.Error())
		assert.Equal(t, codes.InvalidArgument, status.Code(err))
	})

}
