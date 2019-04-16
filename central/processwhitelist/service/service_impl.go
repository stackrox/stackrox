package service

import (
	"context"

	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"github.com/stackrox/rox/central/processwhitelist/datastore"
	"github.com/stackrox/rox/central/role/resources"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/auth/permissions"
	"github.com/stackrox/rox/pkg/grpc/authz"
	"github.com/stackrox/rox/pkg/grpc/authz/perrpc"
	"github.com/stackrox/rox/pkg/grpc/authz/user"
	"github.com/stackrox/rox/pkg/sync"
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

func (s *serviceImpl) GetProcessWhitelist(ctx context.Context, request *v1.GetProcessWhitelistByIdRequest) (*storage.ProcessWhitelist, error) {
	if request.GetWhitelistId() == "" {
		return nil, status.Error(codes.InvalidArgument, "Id must be specified when requesting a process whitelist")
	}
	whitelist, err := s.dataStore.GetProcessWhitelist(request.GetWhitelistId())
	if err != nil {
		return nil, err
	}
	if whitelist == nil {
		return nil, status.Errorf(codes.NotFound, "No process whitelist with id %q found", request.GetWhitelistId())
	}
	return whitelist, nil
}

func parallelizeUpdate(ids []string, parallelFunc func(string) (*storage.ProcessWhitelist, error)) *v1.UpdateProcessWhitelistsResponse {
	wg := sync.WaitGroup{}
	chanLen := len(ids)
	successChan := make(chan *storage.ProcessWhitelist, chanLen)
	errorChan := make(chan *v1.ProcessWhitelistUpdateError, chanLen)
	for _, id := range ids {
		wg.Add(1)
		go func(wlID string) {
			defer wg.Done()
			whitelist, err := parallelFunc(wlID)
			if err != nil {
				errorChan <- &v1.ProcessWhitelistUpdateError{Error: err.Error(), Id: wlID}
				return
			}
			successChan <- whitelist
		}(id)
	}
	wg.Wait()
	close(successChan)
	close(errorChan)
	response := &v1.UpdateProcessWhitelistsResponse{
		Whitelists: make([]*storage.ProcessWhitelist, 0, len(successChan)),
		Errors:     make([]*v1.ProcessWhitelistUpdateError, 0, len(errorChan)),
	}
	for whitelist := range successChan {
		response.Whitelists = append(response.Whitelists, whitelist)
	}
	for err := range errorChan {
		response.Errors = append(response.Errors, err)
	}
	return response
}

func (s *serviceImpl) UpdateProcessWhitelists(ctx context.Context, request *v1.UpdateProcessWhitelistsRequest) (*v1.UpdateProcessWhitelistsResponse, error) {
	updateFunc := func(wlID string) (*storage.ProcessWhitelist, error) {
		return s.dataStore.UpdateProcessWhitelist(wlID, request.GetAddProcessNames(), request.GetRemoveProcessNames())
	}
	return parallelizeUpdate(request.WhitelistIds, updateFunc), nil
}

func (s *serviceImpl) LockProcessWhitelists(ctx context.Context, request *v1.LockProcessWhitelistsRequest) (*v1.UpdateProcessWhitelistsResponse, error) {
	updateFunc := func(wlID string) (*storage.ProcessWhitelist, error) {
		return s.dataStore.UserLockProcessWhitelist(wlID, request.GetLocked())
	}
	return parallelizeUpdate(request.GetWhitelistIds(), updateFunc), nil
}
