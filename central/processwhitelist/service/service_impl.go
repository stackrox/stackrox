package service

import (
	"context"
	"errors"

	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"github.com/stackrox/rox/central/processwhitelist/datastore"
	"github.com/stackrox/rox/central/role/resources"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/auth/permissions"
	"github.com/stackrox/rox/pkg/grpc/authz"
	"github.com/stackrox/rox/pkg/grpc/authz/perrpc"
	"github.com/stackrox/rox/pkg/grpc/authz/user"
	"github.com/stackrox/rox/pkg/stringutils"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var (
	authorizer = perrpc.FromMap(map[authz.Authorizer][]string{
		user.With(permissions.View(resources.ProcessWhitelist)): {
			"/v1.ProcessWhitelistService/GetProcessWhitelists",
			"/v1.ProcessWhitelistService/GetProcessWhitelist",
		},
		user.With(permissions.Modify(resources.ProcessWhitelist)): {
			"/v1.ProcessWhitelistService/UpdateProcessWhitelists",
			"/v1.ProcessWhitelistService/LockProcessWhitelists",
		},
	})
)

type serviceImpl struct {
	dataStore datastore.DataStore
}

func (s *serviceImpl) RegisterServiceServer(server *grpc.Server) {
	v1.RegisterProcessWhitelistServiceServer(server, s)
}

func (s *serviceImpl) RegisterServiceHandler(ctx context.Context, mux *runtime.ServeMux, conn *grpc.ClientConn) error {
	return v1.RegisterProcessWhitelistServiceHandler(ctx, mux, conn)
}

func (s *serviceImpl) AuthFuncOverride(ctx context.Context, fullMethodName string) (context.Context, error) {
	return ctx, authorizer.Authorized(ctx, fullMethodName)
}

func (s *serviceImpl) GetProcessWhitelists(context.Context, *v1.Empty) (*v1.ProcessWhitelistsResponse, error) {
	whitelists, err := s.dataStore.GetProcessWhitelists()
	if err != nil {
		return nil, err
	}
	return &v1.ProcessWhitelistsResponse{Whitelists: whitelists}, nil
}

func validateKeyNotEmpty(key *storage.ProcessWhitelistKey) error {
	if stringutils.AtLeastOneEmpty(
		key.GetDeploymentId(),
		key.GetContainerName(),
	) {
		return errors.New("invalid key: must specify both deployment id and container name")
	}
	return nil
}

func (s *serviceImpl) GetProcessWhitelist(ctx context.Context, request *v1.GetProcessWhitelistRequest) (*storage.ProcessWhitelist, error) {
	if err := validateKeyNotEmpty(request.GetKey()); err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	whitelist, err := s.dataStore.GetProcessWhitelist(request.GetKey())
	if err != nil {
		return nil, err
	}
	if whitelist == nil {
		return nil, status.Errorf(codes.NotFound, "No process whitelist with key %+v found", request.GetKey())
	}
	return whitelist, nil
}

func bulkUpdate(keys []*storage.ProcessWhitelistKey, parallelFunc func(*storage.ProcessWhitelistKey) (*storage.ProcessWhitelist, error)) *v1.UpdateProcessWhitelistsResponse {
	chanLen := len(keys)
	whitelists := make([]*storage.ProcessWhitelist, 0, chanLen)
	errorList := make([]*v1.ProcessWhitelistUpdateError, 0, chanLen)
	for _, key := range keys {
		whitelist, err := parallelFunc(key)
		if err != nil {
			errorList = append(errorList, &v1.ProcessWhitelistUpdateError{Error: err.Error(), Key: key})
		} else {
			whitelists = append(whitelists, whitelist)
		}
	}
	response := &v1.UpdateProcessWhitelistsResponse{
		Whitelists: whitelists,
		Errors:     errorList,
	}
	return response
}

func (s *serviceImpl) UpdateProcessWhitelists(ctx context.Context, request *v1.UpdateProcessWhitelistsRequest) (*v1.UpdateProcessWhitelistsResponse, error) {
	updateFunc := func(key *storage.ProcessWhitelistKey) (*storage.ProcessWhitelist, error) {
		return s.dataStore.UpdateProcessWhitelistElements(key, request.GetAddElements(), request.GetRemoveElements(), false)
	}
	return bulkUpdate(request.GetKeys(), updateFunc), nil
}

func (s *serviceImpl) LockProcessWhitelists(ctx context.Context, request *v1.LockProcessWhitelistsRequest) (*v1.UpdateProcessWhitelistsResponse, error) {
	updateFunc := func(key *storage.ProcessWhitelistKey) (*storage.ProcessWhitelist, error) {
		return s.dataStore.UserLockProcessWhitelist(key, request.GetLocked())
	}
	return bulkUpdate(request.GetKeys(), updateFunc), nil
}
