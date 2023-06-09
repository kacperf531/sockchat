// Code generated by protoc-gen-go-grpc. DO NOT EDIT.
// versions:
// - protoc-gen-go-grpc v1.2.0
// - protoc             v3.21.12
// source: protobuf/sockchat.proto

package sockchat

import (
	context "context"
	grpc "google.golang.org/grpc"
	codes "google.golang.org/grpc/codes"
	status "google.golang.org/grpc/status"
	emptypb "google.golang.org/protobuf/types/known/emptypb"
)

// This is a compile-time assertion to ensure that this generated file
// is compatible with the grpc package it is being compiled against.
// Requires gRPC-Go v1.32.0 or later.
const _ = grpc.SupportPackageIsVersion7

// SockchatClient is the client API for Sockchat service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://pkg.go.dev/google.golang.org/grpc/?tab=doc#ClientConn.NewStream.
type SockchatClient interface {
	RegisterProfile(ctx context.Context, in *RegisterProfileRequest, opts ...grpc.CallOption) (*emptypb.Empty, error)
	GetProfile(ctx context.Context, in *GetProfileRequest, opts ...grpc.CallOption) (*Profile, error)
	EditProfile(ctx context.Context, in *EditProfileRequest, opts ...grpc.CallOption) (*emptypb.Empty, error)
	GetChannelHistory(ctx context.Context, in *GetChannelHistoryRequest, opts ...grpc.CallOption) (*GetChannelHistoryResponse, error)
	GetUserActivityReport(ctx context.Context, in *GetUserActivityReportRequest, opts ...grpc.CallOption) (*GetUserActivityReportResponse, error)
}

type sockchatClient struct {
	cc grpc.ClientConnInterface
}

func NewSockchatClient(cc grpc.ClientConnInterface) SockchatClient {
	return &sockchatClient{cc}
}

func (c *sockchatClient) RegisterProfile(ctx context.Context, in *RegisterProfileRequest, opts ...grpc.CallOption) (*emptypb.Empty, error) {
	out := new(emptypb.Empty)
	err := c.cc.Invoke(ctx, "/sockchat.Sockchat/RegisterProfile", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *sockchatClient) GetProfile(ctx context.Context, in *GetProfileRequest, opts ...grpc.CallOption) (*Profile, error) {
	out := new(Profile)
	err := c.cc.Invoke(ctx, "/sockchat.Sockchat/GetProfile", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *sockchatClient) EditProfile(ctx context.Context, in *EditProfileRequest, opts ...grpc.CallOption) (*emptypb.Empty, error) {
	out := new(emptypb.Empty)
	err := c.cc.Invoke(ctx, "/sockchat.Sockchat/EditProfile", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *sockchatClient) GetChannelHistory(ctx context.Context, in *GetChannelHistoryRequest, opts ...grpc.CallOption) (*GetChannelHistoryResponse, error) {
	out := new(GetChannelHistoryResponse)
	err := c.cc.Invoke(ctx, "/sockchat.Sockchat/GetChannelHistory", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *sockchatClient) GetUserActivityReport(ctx context.Context, in *GetUserActivityReportRequest, opts ...grpc.CallOption) (*GetUserActivityReportResponse, error) {
	out := new(GetUserActivityReportResponse)
	err := c.cc.Invoke(ctx, "/sockchat.Sockchat/GetUserActivityReport", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// SockchatServer is the server API for Sockchat service.
// All implementations must embed UnimplementedSockchatServer
// for forward compatibility
type SockchatServer interface {
	RegisterProfile(context.Context, *RegisterProfileRequest) (*emptypb.Empty, error)
	GetProfile(context.Context, *GetProfileRequest) (*Profile, error)
	EditProfile(context.Context, *EditProfileRequest) (*emptypb.Empty, error)
	GetChannelHistory(context.Context, *GetChannelHistoryRequest) (*GetChannelHistoryResponse, error)
	GetUserActivityReport(context.Context, *GetUserActivityReportRequest) (*GetUserActivityReportResponse, error)
	mustEmbedUnimplementedSockchatServer()
}

// UnimplementedSockchatServer must be embedded to have forward compatible implementations.
type UnimplementedSockchatServer struct {
}

func (UnimplementedSockchatServer) RegisterProfile(context.Context, *RegisterProfileRequest) (*emptypb.Empty, error) {
	return nil, status.Errorf(codes.Unimplemented, "method RegisterProfile not implemented")
}
func (UnimplementedSockchatServer) GetProfile(context.Context, *GetProfileRequest) (*Profile, error) {
	return nil, status.Errorf(codes.Unimplemented, "method GetProfile not implemented")
}
func (UnimplementedSockchatServer) EditProfile(context.Context, *EditProfileRequest) (*emptypb.Empty, error) {
	return nil, status.Errorf(codes.Unimplemented, "method EditProfile not implemented")
}
func (UnimplementedSockchatServer) GetChannelHistory(context.Context, *GetChannelHistoryRequest) (*GetChannelHistoryResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method GetChannelHistory not implemented")
}
func (UnimplementedSockchatServer) GetUserActivityReport(context.Context, *GetUserActivityReportRequest) (*GetUserActivityReportResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method GetUserActivityReport not implemented")
}
func (UnimplementedSockchatServer) mustEmbedUnimplementedSockchatServer() {}

// UnsafeSockchatServer may be embedded to opt out of forward compatibility for this service.
// Use of this interface is not recommended, as added methods to SockchatServer will
// result in compilation errors.
type UnsafeSockchatServer interface {
	mustEmbedUnimplementedSockchatServer()
}

func RegisterSockchatServer(s grpc.ServiceRegistrar, srv SockchatServer) {
	s.RegisterService(&Sockchat_ServiceDesc, srv)
}

func _Sockchat_RegisterProfile_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(RegisterProfileRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(SockchatServer).RegisterProfile(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/sockchat.Sockchat/RegisterProfile",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(SockchatServer).RegisterProfile(ctx, req.(*RegisterProfileRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _Sockchat_GetProfile_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(GetProfileRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(SockchatServer).GetProfile(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/sockchat.Sockchat/GetProfile",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(SockchatServer).GetProfile(ctx, req.(*GetProfileRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _Sockchat_EditProfile_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(EditProfileRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(SockchatServer).EditProfile(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/sockchat.Sockchat/EditProfile",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(SockchatServer).EditProfile(ctx, req.(*EditProfileRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _Sockchat_GetChannelHistory_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(GetChannelHistoryRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(SockchatServer).GetChannelHistory(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/sockchat.Sockchat/GetChannelHistory",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(SockchatServer).GetChannelHistory(ctx, req.(*GetChannelHistoryRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _Sockchat_GetUserActivityReport_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(GetUserActivityReportRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(SockchatServer).GetUserActivityReport(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/sockchat.Sockchat/GetUserActivityReport",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(SockchatServer).GetUserActivityReport(ctx, req.(*GetUserActivityReportRequest))
	}
	return interceptor(ctx, in, info, handler)
}

// Sockchat_ServiceDesc is the grpc.ServiceDesc for Sockchat service.
// It's only intended for direct use with grpc.RegisterService,
// and not to be introspected or modified (even as a copy)
var Sockchat_ServiceDesc = grpc.ServiceDesc{
	ServiceName: "sockchat.Sockchat",
	HandlerType: (*SockchatServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "RegisterProfile",
			Handler:    _Sockchat_RegisterProfile_Handler,
		},
		{
			MethodName: "GetProfile",
			Handler:    _Sockchat_GetProfile_Handler,
		},
		{
			MethodName: "EditProfile",
			Handler:    _Sockchat_EditProfile_Handler,
		},
		{
			MethodName: "GetChannelHistory",
			Handler:    _Sockchat_GetChannelHistory_Handler,
		},
		{
			MethodName: "GetUserActivityReport",
			Handler:    _Sockchat_GetUserActivityReport_Handler,
		},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "protobuf/sockchat.proto",
}
