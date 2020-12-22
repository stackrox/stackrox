package service

import (
	"context"

	"github.com/golang/protobuf/proto"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// TODO(ROX-6194): The code below implements a deprecated `ProcessWhitelistServiceServer`
//   and shall be removed after the deprecation cycle started with the 54.0 release.
//
// The implementation converts protobufs using Marshal/Unmarshal trick
// and relays the calls to the "real" methods.

func convertProto(in, out proto.Message) error {
	bytes, err := proto.Marshal(in)
	if err != nil {
		return err
	}

	return proto.Unmarshal(bytes, out)
}

func (s *serviceImpl) GetProcessWhitelist(ctx context.Context, request *v1.GetProcessWhitelistRequest) (*storage.ProcessBaseline, error) {
	// Here the conversion is straightforward so no calls to `convertProto()`.
	r := &v1.GetProcessBaselineRequest{Key: request.GetKey()}
	return s.GetProcessBaseline(ctx, r)
}

func (s *serviceImpl) UpdateProcessWhitelists(ctx context.Context, request *v1.UpdateProcessWhitelistsRequest) (*v1.UpdateProcessWhitelistsResponse, error) {
	var realRequest v1.UpdateProcessBaselinesRequest
	if err := convertProto(request, &realRequest); err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	realResponse, err := s.UpdateProcessBaselines(ctx, &realRequest)
	if err != nil {
		return nil, err
	}

	var response v1.UpdateProcessWhitelistsResponse
	if err := convertProto(realResponse, &response); err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &response, nil
}

func (s *serviceImpl) LockProcessWhitelists(ctx context.Context, request *v1.LockProcessWhitelistsRequest) (*v1.UpdateProcessWhitelistsResponse, error) {
	var realRequest v1.LockProcessBaselinesRequest
	if err := convertProto(request, &realRequest); err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	realResponse, err := s.LockProcessBaselines(ctx, &realRequest)
	if err != nil {
		return nil, err
	}

	var response v1.UpdateProcessWhitelistsResponse
	if err := convertProto(realResponse, &response); err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &response, nil
}

func (s *serviceImpl) DeleteProcessWhitelists(ctx context.Context, request *v1.DeleteProcessWhitelistsRequest) (*v1.DeleteProcessWhitelistsResponse, error) {
	var realRequest v1.DeleteProcessBaselinesRequest
	if err := convertProto(request, &realRequest); err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	realResponse, err := s.DeleteProcessBaselines(ctx, &realRequest)
	if err != nil {
		return nil, err
	}

	var response v1.DeleteProcessWhitelistsResponse
	if err := convertProto(realResponse, &response); err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &response, nil
}
