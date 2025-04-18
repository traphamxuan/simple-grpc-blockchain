// Code generated by protoc-gen-go-grpc. DO NOT EDIT.
// versions:
// - protoc-gen-go-grpc v1.5.1
// - protoc             v5.29.3
// source: proto/block.proto

package proto

import (
	context "context"
	grpc "google.golang.org/grpc"
	codes "google.golang.org/grpc/codes"
	status "google.golang.org/grpc/status"
	emptypb "google.golang.org/protobuf/types/known/emptypb"
)

// This is a compile-time assertion to ensure that this generated file
// is compatible with the grpc package it is being compiled against.
// Requires gRPC-Go v1.64.0 or later.
const _ = grpc.SupportPackageIsVersion9

const (
	Blockchain_GetBlock_FullMethodName        = "/blockchain.Blockchain/GetBlock"
	Blockchain_GetHighestBlock_FullMethodName = "/blockchain.Blockchain/GetHighestBlock"
	Blockchain_AddNewBlock_FullMethodName     = "/blockchain.Blockchain/AddNewBlock"
)

// BlockchainClient is the client API for Blockchain service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://pkg.go.dev/google.golang.org/grpc/?tab=doc#ClientConn.NewStream.
type BlockchainClient interface {
	GetBlock(ctx context.Context, in *GetBlockRequest, opts ...grpc.CallOption) (*Block, error)
	GetHighestBlock(ctx context.Context, in *emptypb.Empty, opts ...grpc.CallOption) (*Block, error)
	AddNewBlock(ctx context.Context, in *Block, opts ...grpc.CallOption) (*Block, error)
}

type blockchainClient struct {
	cc grpc.ClientConnInterface
}

func NewBlockchainClient(cc grpc.ClientConnInterface) BlockchainClient {
	return &blockchainClient{cc}
}

func (c *blockchainClient) GetBlock(ctx context.Context, in *GetBlockRequest, opts ...grpc.CallOption) (*Block, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(Block)
	err := c.cc.Invoke(ctx, Blockchain_GetBlock_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *blockchainClient) GetHighestBlock(ctx context.Context, in *emptypb.Empty, opts ...grpc.CallOption) (*Block, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(Block)
	err := c.cc.Invoke(ctx, Blockchain_GetHighestBlock_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *blockchainClient) AddNewBlock(ctx context.Context, in *Block, opts ...grpc.CallOption) (*Block, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(Block)
	err := c.cc.Invoke(ctx, Blockchain_AddNewBlock_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// BlockchainServer is the server API for Blockchain service.
// All implementations must embed UnimplementedBlockchainServer
// for forward compatibility.
type BlockchainServer interface {
	GetBlock(context.Context, *GetBlockRequest) (*Block, error)
	GetHighestBlock(context.Context, *emptypb.Empty) (*Block, error)
	AddNewBlock(context.Context, *Block) (*Block, error)
	mustEmbedUnimplementedBlockchainServer()
}

// UnimplementedBlockchainServer must be embedded to have
// forward compatible implementations.
//
// NOTE: this should be embedded by value instead of pointer to avoid a nil
// pointer dereference when methods are called.
type UnimplementedBlockchainServer struct{}

func (UnimplementedBlockchainServer) GetBlock(context.Context, *GetBlockRequest) (*Block, error) {
	return nil, status.Errorf(codes.Unimplemented, "method GetBlock not implemented")
}
func (UnimplementedBlockchainServer) GetHighestBlock(context.Context, *emptypb.Empty) (*Block, error) {
	return nil, status.Errorf(codes.Unimplemented, "method GetHighestBlock not implemented")
}
func (UnimplementedBlockchainServer) AddNewBlock(context.Context, *Block) (*Block, error) {
	return nil, status.Errorf(codes.Unimplemented, "method AddNewBlock not implemented")
}
func (UnimplementedBlockchainServer) mustEmbedUnimplementedBlockchainServer() {}
func (UnimplementedBlockchainServer) testEmbeddedByValue()                    {}

// UnsafeBlockchainServer may be embedded to opt out of forward compatibility for this service.
// Use of this interface is not recommended, as added methods to BlockchainServer will
// result in compilation errors.
type UnsafeBlockchainServer interface {
	mustEmbedUnimplementedBlockchainServer()
}

func RegisterBlockchainServer(s grpc.ServiceRegistrar, srv BlockchainServer) {
	// If the following call pancis, it indicates UnimplementedBlockchainServer was
	// embedded by pointer and is nil.  This will cause panics if an
	// unimplemented method is ever invoked, so we test this at initialization
	// time to prevent it from happening at runtime later due to I/O.
	if t, ok := srv.(interface{ testEmbeddedByValue() }); ok {
		t.testEmbeddedByValue()
	}
	s.RegisterService(&Blockchain_ServiceDesc, srv)
}

func _Blockchain_GetBlock_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(GetBlockRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(BlockchainServer).GetBlock(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: Blockchain_GetBlock_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(BlockchainServer).GetBlock(ctx, req.(*GetBlockRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _Blockchain_GetHighestBlock_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(emptypb.Empty)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(BlockchainServer).GetHighestBlock(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: Blockchain_GetHighestBlock_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(BlockchainServer).GetHighestBlock(ctx, req.(*emptypb.Empty))
	}
	return interceptor(ctx, in, info, handler)
}

func _Blockchain_AddNewBlock_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(Block)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(BlockchainServer).AddNewBlock(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: Blockchain_AddNewBlock_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(BlockchainServer).AddNewBlock(ctx, req.(*Block))
	}
	return interceptor(ctx, in, info, handler)
}

// Blockchain_ServiceDesc is the grpc.ServiceDesc for Blockchain service.
// It's only intended for direct use with grpc.RegisterService,
// and not to be introspected or modified (even as a copy)
var Blockchain_ServiceDesc = grpc.ServiceDesc{
	ServiceName: "blockchain.Blockchain",
	HandlerType: (*BlockchainServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "GetBlock",
			Handler:    _Blockchain_GetBlock_Handler,
		},
		{
			MethodName: "GetHighestBlock",
			Handler:    _Blockchain_GetHighestBlock_Handler,
		},
		{
			MethodName: "AddNewBlock",
			Handler:    _Blockchain_AddNewBlock_Handler,
		},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "proto/block.proto",
}
