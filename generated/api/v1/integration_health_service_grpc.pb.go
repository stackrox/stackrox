// Code generated by protoc-gen-go-grpc. DO NOT EDIT.
// versions:
// - protoc-gen-go-grpc v1.5.1
// - protoc             v4.25.3
// source: api/v1/integration_health_service.proto

package v1

import (
	context "context"
	grpc "google.golang.org/grpc"
	codes "google.golang.org/grpc/codes"
	status "google.golang.org/grpc/status"
)

// This is a compile-time assertion to ensure that this generated file
// is compatible with the grpc package it is being compiled against.
// Requires gRPC-Go v1.64.0 or later.
const _ = grpc.SupportPackageIsVersion9

const (
	IntegrationHealthService_GetImageIntegrations_FullMethodName   = "/v1.IntegrationHealthService/GetImageIntegrations"
	IntegrationHealthService_GetNotifiers_FullMethodName           = "/v1.IntegrationHealthService/GetNotifiers"
	IntegrationHealthService_GetBackupPlugins_FullMethodName       = "/v1.IntegrationHealthService/GetBackupPlugins"
	IntegrationHealthService_GetDeclarativeConfigs_FullMethodName  = "/v1.IntegrationHealthService/GetDeclarativeConfigs"
	IntegrationHealthService_GetVulnDefinitionsInfo_FullMethodName = "/v1.IntegrationHealthService/GetVulnDefinitionsInfo"
)

// IntegrationHealthServiceClient is the client API for IntegrationHealthService service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://pkg.go.dev/google.golang.org/grpc/?tab=doc#ClientConn.NewStream.
type IntegrationHealthServiceClient interface {
	GetImageIntegrations(ctx context.Context, in *Empty, opts ...grpc.CallOption) (*GetIntegrationHealthResponse, error)
	GetNotifiers(ctx context.Context, in *Empty, opts ...grpc.CallOption) (*GetIntegrationHealthResponse, error)
	GetBackupPlugins(ctx context.Context, in *Empty, opts ...grpc.CallOption) (*GetIntegrationHealthResponse, error)
	GetDeclarativeConfigs(ctx context.Context, in *Empty, opts ...grpc.CallOption) (*GetIntegrationHealthResponse, error)
	GetVulnDefinitionsInfo(ctx context.Context, in *VulnDefinitionsInfoRequest, opts ...grpc.CallOption) (*VulnDefinitionsInfo, error)
}

type integrationHealthServiceClient struct {
	cc grpc.ClientConnInterface
}

func NewIntegrationHealthServiceClient(cc grpc.ClientConnInterface) IntegrationHealthServiceClient {
	return &integrationHealthServiceClient{cc}
}

func (c *integrationHealthServiceClient) GetImageIntegrations(ctx context.Context, in *Empty, opts ...grpc.CallOption) (*GetIntegrationHealthResponse, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(GetIntegrationHealthResponse)
	err := c.cc.Invoke(ctx, IntegrationHealthService_GetImageIntegrations_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *integrationHealthServiceClient) GetNotifiers(ctx context.Context, in *Empty, opts ...grpc.CallOption) (*GetIntegrationHealthResponse, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(GetIntegrationHealthResponse)
	err := c.cc.Invoke(ctx, IntegrationHealthService_GetNotifiers_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *integrationHealthServiceClient) GetBackupPlugins(ctx context.Context, in *Empty, opts ...grpc.CallOption) (*GetIntegrationHealthResponse, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(GetIntegrationHealthResponse)
	err := c.cc.Invoke(ctx, IntegrationHealthService_GetBackupPlugins_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *integrationHealthServiceClient) GetDeclarativeConfigs(ctx context.Context, in *Empty, opts ...grpc.CallOption) (*GetIntegrationHealthResponse, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(GetIntegrationHealthResponse)
	err := c.cc.Invoke(ctx, IntegrationHealthService_GetDeclarativeConfigs_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *integrationHealthServiceClient) GetVulnDefinitionsInfo(ctx context.Context, in *VulnDefinitionsInfoRequest, opts ...grpc.CallOption) (*VulnDefinitionsInfo, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(VulnDefinitionsInfo)
	err := c.cc.Invoke(ctx, IntegrationHealthService_GetVulnDefinitionsInfo_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// IntegrationHealthServiceServer is the server API for IntegrationHealthService service.
// All implementations should embed UnimplementedIntegrationHealthServiceServer
// for forward compatibility.
type IntegrationHealthServiceServer interface {
	GetImageIntegrations(context.Context, *Empty) (*GetIntegrationHealthResponse, error)
	GetNotifiers(context.Context, *Empty) (*GetIntegrationHealthResponse, error)
	GetBackupPlugins(context.Context, *Empty) (*GetIntegrationHealthResponse, error)
	GetDeclarativeConfigs(context.Context, *Empty) (*GetIntegrationHealthResponse, error)
	GetVulnDefinitionsInfo(context.Context, *VulnDefinitionsInfoRequest) (*VulnDefinitionsInfo, error)
}

// UnimplementedIntegrationHealthServiceServer should be embedded to have
// forward compatible implementations.
//
// NOTE: this should be embedded by value instead of pointer to avoid a nil
// pointer dereference when methods are called.
type UnimplementedIntegrationHealthServiceServer struct{}

func (UnimplementedIntegrationHealthServiceServer) GetImageIntegrations(context.Context, *Empty) (*GetIntegrationHealthResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method GetImageIntegrations not implemented")
}
func (UnimplementedIntegrationHealthServiceServer) GetNotifiers(context.Context, *Empty) (*GetIntegrationHealthResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method GetNotifiers not implemented")
}
func (UnimplementedIntegrationHealthServiceServer) GetBackupPlugins(context.Context, *Empty) (*GetIntegrationHealthResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method GetBackupPlugins not implemented")
}
func (UnimplementedIntegrationHealthServiceServer) GetDeclarativeConfigs(context.Context, *Empty) (*GetIntegrationHealthResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method GetDeclarativeConfigs not implemented")
}
func (UnimplementedIntegrationHealthServiceServer) GetVulnDefinitionsInfo(context.Context, *VulnDefinitionsInfoRequest) (*VulnDefinitionsInfo, error) {
	return nil, status.Errorf(codes.Unimplemented, "method GetVulnDefinitionsInfo not implemented")
}
func (UnimplementedIntegrationHealthServiceServer) testEmbeddedByValue() {}

// UnsafeIntegrationHealthServiceServer may be embedded to opt out of forward compatibility for this service.
// Use of this interface is not recommended, as added methods to IntegrationHealthServiceServer will
// result in compilation errors.
type UnsafeIntegrationHealthServiceServer interface {
	mustEmbedUnimplementedIntegrationHealthServiceServer()
}

func RegisterIntegrationHealthServiceServer(s grpc.ServiceRegistrar, srv IntegrationHealthServiceServer) {
	// If the following call pancis, it indicates UnimplementedIntegrationHealthServiceServer was
	// embedded by pointer and is nil.  This will cause panics if an
	// unimplemented method is ever invoked, so we test this at initialization
	// time to prevent it from happening at runtime later due to I/O.
	if t, ok := srv.(interface{ testEmbeddedByValue() }); ok {
		t.testEmbeddedByValue()
	}
	s.RegisterService(&IntegrationHealthService_ServiceDesc, srv)
}

func _IntegrationHealthService_GetImageIntegrations_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(Empty)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(IntegrationHealthServiceServer).GetImageIntegrations(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: IntegrationHealthService_GetImageIntegrations_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(IntegrationHealthServiceServer).GetImageIntegrations(ctx, req.(*Empty))
	}
	return interceptor(ctx, in, info, handler)
}

func _IntegrationHealthService_GetNotifiers_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(Empty)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(IntegrationHealthServiceServer).GetNotifiers(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: IntegrationHealthService_GetNotifiers_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(IntegrationHealthServiceServer).GetNotifiers(ctx, req.(*Empty))
	}
	return interceptor(ctx, in, info, handler)
}

func _IntegrationHealthService_GetBackupPlugins_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(Empty)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(IntegrationHealthServiceServer).GetBackupPlugins(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: IntegrationHealthService_GetBackupPlugins_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(IntegrationHealthServiceServer).GetBackupPlugins(ctx, req.(*Empty))
	}
	return interceptor(ctx, in, info, handler)
}

func _IntegrationHealthService_GetDeclarativeConfigs_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(Empty)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(IntegrationHealthServiceServer).GetDeclarativeConfigs(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: IntegrationHealthService_GetDeclarativeConfigs_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(IntegrationHealthServiceServer).GetDeclarativeConfigs(ctx, req.(*Empty))
	}
	return interceptor(ctx, in, info, handler)
}

func _IntegrationHealthService_GetVulnDefinitionsInfo_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(VulnDefinitionsInfoRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(IntegrationHealthServiceServer).GetVulnDefinitionsInfo(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: IntegrationHealthService_GetVulnDefinitionsInfo_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(IntegrationHealthServiceServer).GetVulnDefinitionsInfo(ctx, req.(*VulnDefinitionsInfoRequest))
	}
	return interceptor(ctx, in, info, handler)
}

// IntegrationHealthService_ServiceDesc is the grpc.ServiceDesc for IntegrationHealthService service.
// It's only intended for direct use with grpc.RegisterService,
// and not to be introspected or modified (even as a copy)
var IntegrationHealthService_ServiceDesc = grpc.ServiceDesc{
	ServiceName: "v1.IntegrationHealthService",
	HandlerType: (*IntegrationHealthServiceServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "GetImageIntegrations",
			Handler:    _IntegrationHealthService_GetImageIntegrations_Handler,
		},
		{
			MethodName: "GetNotifiers",
			Handler:    _IntegrationHealthService_GetNotifiers_Handler,
		},
		{
			MethodName: "GetBackupPlugins",
			Handler:    _IntegrationHealthService_GetBackupPlugins_Handler,
		},
		{
			MethodName: "GetDeclarativeConfigs",
			Handler:    _IntegrationHealthService_GetDeclarativeConfigs_Handler,
		},
		{
			MethodName: "GetVulnDefinitionsInfo",
			Handler:    _IntegrationHealthService_GetVulnDefinitionsInfo_Handler,
		},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "api/v1/integration_health_service.proto",
}
