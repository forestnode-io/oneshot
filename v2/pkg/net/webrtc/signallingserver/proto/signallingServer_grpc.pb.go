// Code generated by protoc-gen-go-grpc. DO NOT EDIT.
// versions:
// - protoc-gen-go-grpc v1.3.0
// - protoc             v3.21.12
// source: signallingServer.proto

package proto

import (
	context "context"
	grpc "google.golang.org/grpc"
	codes "google.golang.org/grpc/codes"
	status "google.golang.org/grpc/status"
)

// This is a compile-time assertion to ensure that this generated file
// is compatible with the grpc package it is being compiled against.
// Requires gRPC-Go v1.32.0 or later.
const _ = grpc.SupportPackageIsVersion7

const (
	SignallingServer_Connect_FullMethodName = "/signallingserverproto.SignallingServer/Connect"
)

// SignallingServerClient is the client API for SignallingServer service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://pkg.go.dev/google.golang.org/grpc/?tab=doc#ClientConn.NewStream.
type SignallingServerClient interface {
	Connect(ctx context.Context, opts ...grpc.CallOption) (SignallingServer_ConnectClient, error)
}

type signallingServerClient struct {
	cc grpc.ClientConnInterface
}

func NewSignallingServerClient(cc grpc.ClientConnInterface) SignallingServerClient {
	return &signallingServerClient{cc}
}

func (c *signallingServerClient) Connect(ctx context.Context, opts ...grpc.CallOption) (SignallingServer_ConnectClient, error) {
	stream, err := c.cc.NewStream(ctx, &SignallingServer_ServiceDesc.Streams[0], SignallingServer_Connect_FullMethodName, opts...)
	if err != nil {
		return nil, err
	}
	x := &signallingServerConnectClient{stream}
	return x, nil
}

type SignallingServer_ConnectClient interface {
	Send(*Envelope) error
	Recv() (*Envelope, error)
	grpc.ClientStream
}

type signallingServerConnectClient struct {
	grpc.ClientStream
}

func (x *signallingServerConnectClient) Send(m *Envelope) error {
	return x.ClientStream.SendMsg(m)
}

func (x *signallingServerConnectClient) Recv() (*Envelope, error) {
	m := new(Envelope)
	if err := x.ClientStream.RecvMsg(m); err != nil {
		return nil, err
	}
	return m, nil
}

// SignallingServerServer is the server API for SignallingServer service.
// All implementations must embed UnimplementedSignallingServerServer
// for forward compatibility
type SignallingServerServer interface {
	Connect(SignallingServer_ConnectServer) error
	mustEmbedUnimplementedSignallingServerServer()
}

// UnimplementedSignallingServerServer must be embedded to have forward compatible implementations.
type UnimplementedSignallingServerServer struct {
}

func (UnimplementedSignallingServerServer) Connect(SignallingServer_ConnectServer) error {
	return status.Errorf(codes.Unimplemented, "method Connect not implemented")
}
func (UnimplementedSignallingServerServer) mustEmbedUnimplementedSignallingServerServer() {}

// UnsafeSignallingServerServer may be embedded to opt out of forward compatibility for this service.
// Use of this interface is not recommended, as added methods to SignallingServerServer will
// result in compilation errors.
type UnsafeSignallingServerServer interface {
	mustEmbedUnimplementedSignallingServerServer()
}

func RegisterSignallingServerServer(s grpc.ServiceRegistrar, srv SignallingServerServer) {
	s.RegisterService(&SignallingServer_ServiceDesc, srv)
}

func _SignallingServer_Connect_Handler(srv interface{}, stream grpc.ServerStream) error {
	return srv.(SignallingServerServer).Connect(&signallingServerConnectServer{stream})
}

type SignallingServer_ConnectServer interface {
	Send(*Envelope) error
	Recv() (*Envelope, error)
	grpc.ServerStream
}

type signallingServerConnectServer struct {
	grpc.ServerStream
}

func (x *signallingServerConnectServer) Send(m *Envelope) error {
	return x.ServerStream.SendMsg(m)
}

func (x *signallingServerConnectServer) Recv() (*Envelope, error) {
	m := new(Envelope)
	if err := x.ServerStream.RecvMsg(m); err != nil {
		return nil, err
	}
	return m, nil
}

// SignallingServer_ServiceDesc is the grpc.ServiceDesc for SignallingServer service.
// It's only intended for direct use with grpc.RegisterService,
// and not to be introspected or modified (even as a copy)
var SignallingServer_ServiceDesc = grpc.ServiceDesc{
	ServiceName: "signallingserverproto.SignallingServer",
	HandlerType: (*SignallingServerServer)(nil),
	Methods:     []grpc.MethodDesc{},
	Streams: []grpc.StreamDesc{
		{
			StreamName:    "Connect",
			Handler:       _SignallingServer_Connect_Handler,
			ServerStreams: true,
			ClientStreams: true,
		},
	},
	Metadata: "signallingServer.proto",
}
