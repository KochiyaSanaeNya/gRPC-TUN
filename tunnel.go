package main

import (
	"context"

	"google.golang.org/grpc"
)

type Chunk struct {
	Data []byte `json:"data,omitempty"`
	Fin  bool   `json:"fin,omitempty"`
}

type TunnelServiceClient interface {
	Tunnel(ctx context.Context, opts ...grpc.CallOption) (Tunnel_TunnelClient, error)
}

type tunnelServiceClient struct {
	cc grpc.ClientConnInterface
}

func NewTunnelServiceClient(cc grpc.ClientConnInterface) TunnelServiceClient {
	return &tunnelServiceClient{cc}
}

func (c *tunnelServiceClient) Tunnel(ctx context.Context, opts ...grpc.CallOption) (Tunnel_TunnelClient, error) {
	stream, err := c.cc.NewStream(ctx, &TunnelService_ServiceDesc.Streams[0], "/gRPC.TunnelService/Tunnel", opts...)
	if err != nil {
		return nil, err
	}
	return &tunnelServiceTunnelClient{stream}, nil
}

type Tunnel_TunnelClient interface {
	Send(*Chunk) error
	Recv() (*Chunk, error)
	CloseSend() error
	Context() context.Context
}

type tunnelServiceTunnelClient struct {
	grpc.ClientStream
}

func (x *tunnelServiceTunnelClient) Send(m *Chunk) error {
	return x.ClientStream.SendMsg(m)
}

func (x *tunnelServiceTunnelClient) Recv() (*Chunk, error) {
	m := new(Chunk)
	if err := x.ClientStream.RecvMsg(m); err != nil {
		return nil, err
	}
	return m, nil
}

type TunnelServiceServer interface {
	Tunnel(Tunnel_TunnelServer) error
}

type Tunnel_TunnelServer interface {
	Send(*Chunk) error
	Recv() (*Chunk, error)
	Context() context.Context
}

type tunnelServiceTunnelServer struct {
	grpc.ServerStream
}

func (x *tunnelServiceTunnelServer) Send(m *Chunk) error {
	return x.ServerStream.SendMsg(m)
}

func (x *tunnelServiceTunnelServer) Recv() (*Chunk, error) {
	m := new(Chunk)
	if err := x.ServerStream.RecvMsg(m); err != nil {
		return nil, err
	}
	return m, nil
}

func RegisterTunnelServiceServer(s *grpc.Server, srv TunnelServiceServer) {
	s.RegisterService(&TunnelService_ServiceDesc, srv)
}

func _TunnelService_Tunnel_Handler(srv any, stream grpc.ServerStream) error {
	return srv.(TunnelServiceServer).Tunnel(&tunnelServiceTunnelServer{stream})
}

var TunnelService_ServiceDesc = grpc.ServiceDesc{
	ServiceName: "gRPC.TunnelService",
	HandlerType: (*TunnelServiceServer)(nil),
	Streams: []grpc.StreamDesc{
		{
			StreamName:    "Tunnel",
			Handler:       _TunnelService_Tunnel_Handler,
			ServerStreams: true,
			ClientStreams: true,
		},
	},
	Metadata: "tunnel.proto",
}
