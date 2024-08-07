// Code generated by protoc-gen-go-grpc. DO NOT EDIT.
// versions:
// - protoc-gen-go-grpc v1.5.1
// - protoc             v4.25.3
// source: api/v1/detection_service.proto

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
	DetectionService_DetectBuildTime_FullMethodName          = "/v1.DetectionService/DetectBuildTime"
	DetectionService_DetectDeployTime_FullMethodName         = "/v1.DetectionService/DetectDeployTime"
	DetectionService_DetectDeployTimeFromYAML_FullMethodName = "/v1.DetectionService/DetectDeployTimeFromYAML"
)

// DetectionServiceClient is the client API for DetectionService service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://pkg.go.dev/google.golang.org/grpc/?tab=doc#ClientConn.NewStream.
//
// DetectionService APIs can be used to check for build and deploy time policy violations.
type DetectionServiceClient interface {
	// DetectBuildTime checks if any images violate build time policies.
	DetectBuildTime(ctx context.Context, in *BuildDetectionRequest, opts ...grpc.CallOption) (*BuildDetectionResponse, error)
	// DetectDeployTime checks if any deployments violate deploy time policies.
	DetectDeployTime(ctx context.Context, in *DeployDetectionRequest, opts ...grpc.CallOption) (*DeployDetectionResponse, error)
	// DetectDeployTimeFromYAML checks if the given deployment yaml violates any deploy time policies.
	DetectDeployTimeFromYAML(ctx context.Context, in *DeployYAMLDetectionRequest, opts ...grpc.CallOption) (*DeployDetectionResponse, error)
}

type detectionServiceClient struct {
	cc grpc.ClientConnInterface
}

func NewDetectionServiceClient(cc grpc.ClientConnInterface) DetectionServiceClient {
	return &detectionServiceClient{cc}
}

func (c *detectionServiceClient) DetectBuildTime(ctx context.Context, in *BuildDetectionRequest, opts ...grpc.CallOption) (*BuildDetectionResponse, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(BuildDetectionResponse)
	err := c.cc.Invoke(ctx, DetectionService_DetectBuildTime_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *detectionServiceClient) DetectDeployTime(ctx context.Context, in *DeployDetectionRequest, opts ...grpc.CallOption) (*DeployDetectionResponse, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(DeployDetectionResponse)
	err := c.cc.Invoke(ctx, DetectionService_DetectDeployTime_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *detectionServiceClient) DetectDeployTimeFromYAML(ctx context.Context, in *DeployYAMLDetectionRequest, opts ...grpc.CallOption) (*DeployDetectionResponse, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(DeployDetectionResponse)
	err := c.cc.Invoke(ctx, DetectionService_DetectDeployTimeFromYAML_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// DetectionServiceServer is the server API for DetectionService service.
// All implementations should embed UnimplementedDetectionServiceServer
// for forward compatibility.
//
// DetectionService APIs can be used to check for build and deploy time policy violations.
type DetectionServiceServer interface {
	// DetectBuildTime checks if any images violate build time policies.
	DetectBuildTime(context.Context, *BuildDetectionRequest) (*BuildDetectionResponse, error)
	// DetectDeployTime checks if any deployments violate deploy time policies.
	DetectDeployTime(context.Context, *DeployDetectionRequest) (*DeployDetectionResponse, error)
	// DetectDeployTimeFromYAML checks if the given deployment yaml violates any deploy time policies.
	DetectDeployTimeFromYAML(context.Context, *DeployYAMLDetectionRequest) (*DeployDetectionResponse, error)
}

// UnimplementedDetectionServiceServer should be embedded to have
// forward compatible implementations.
//
// NOTE: this should be embedded by value instead of pointer to avoid a nil
// pointer dereference when methods are called.
type UnimplementedDetectionServiceServer struct{}

func (UnimplementedDetectionServiceServer) DetectBuildTime(context.Context, *BuildDetectionRequest) (*BuildDetectionResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method DetectBuildTime not implemented")
}
func (UnimplementedDetectionServiceServer) DetectDeployTime(context.Context, *DeployDetectionRequest) (*DeployDetectionResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method DetectDeployTime not implemented")
}
func (UnimplementedDetectionServiceServer) DetectDeployTimeFromYAML(context.Context, *DeployYAMLDetectionRequest) (*DeployDetectionResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method DetectDeployTimeFromYAML not implemented")
}
func (UnimplementedDetectionServiceServer) testEmbeddedByValue() {}

// UnsafeDetectionServiceServer may be embedded to opt out of forward compatibility for this service.
// Use of this interface is not recommended, as added methods to DetectionServiceServer will
// result in compilation errors.
type UnsafeDetectionServiceServer interface {
	mustEmbedUnimplementedDetectionServiceServer()
}

func RegisterDetectionServiceServer(s grpc.ServiceRegistrar, srv DetectionServiceServer) {
	// If the following call pancis, it indicates UnimplementedDetectionServiceServer was
	// embedded by pointer and is nil.  This will cause panics if an
	// unimplemented method is ever invoked, so we test this at initialization
	// time to prevent it from happening at runtime later due to I/O.
	if t, ok := srv.(interface{ testEmbeddedByValue() }); ok {
		t.testEmbeddedByValue()
	}
	s.RegisterService(&DetectionService_ServiceDesc, srv)
}

func _DetectionService_DetectBuildTime_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(BuildDetectionRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(DetectionServiceServer).DetectBuildTime(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: DetectionService_DetectBuildTime_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(DetectionServiceServer).DetectBuildTime(ctx, req.(*BuildDetectionRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _DetectionService_DetectDeployTime_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(DeployDetectionRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(DetectionServiceServer).DetectDeployTime(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: DetectionService_DetectDeployTime_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(DetectionServiceServer).DetectDeployTime(ctx, req.(*DeployDetectionRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _DetectionService_DetectDeployTimeFromYAML_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(DeployYAMLDetectionRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(DetectionServiceServer).DetectDeployTimeFromYAML(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: DetectionService_DetectDeployTimeFromYAML_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(DetectionServiceServer).DetectDeployTimeFromYAML(ctx, req.(*DeployYAMLDetectionRequest))
	}
	return interceptor(ctx, in, info, handler)
}

// DetectionService_ServiceDesc is the grpc.ServiceDesc for DetectionService service.
// It's only intended for direct use with grpc.RegisterService,
// and not to be introspected or modified (even as a copy)
var DetectionService_ServiceDesc = grpc.ServiceDesc{
	ServiceName: "v1.DetectionService",
	HandlerType: (*DetectionServiceServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "DetectBuildTime",
			Handler:    _DetectionService_DetectBuildTime_Handler,
		},
		{
			MethodName: "DetectDeployTime",
			Handler:    _DetectionService_DetectDeployTime_Handler,
		},
		{
			MethodName: "DetectDeployTimeFromYAML",
			Handler:    _DetectionService_DetectDeployTimeFromYAML_Handler,
		},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "api/v1/detection_service.proto",
}
