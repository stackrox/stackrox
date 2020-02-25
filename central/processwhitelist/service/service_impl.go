package service

import (
	"context"
	"errors"

	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"github.com/stackrox/rox/central/processwhitelist/datastore"
	"github.com/stackrox/rox/central/reprocessor"
	"github.com/stackrox/rox/central/role/resources"
	"github.com/stackrox/rox/central/sensor/service/connection"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/auth/permissions"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/grpc/authz"
	"github.com/stackrox/rox/pkg/grpc/authz/perrpc"
	"github.com/stackrox/rox/pkg/grpc/authz/user"
	"github.com/stackrox/rox/pkg/set"
	"github.com/stackrox/rox/pkg/stringutils"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var (
	authorizer = perrpc.FromMap(map[authz.Authorizer][]string{
		user.With(permissions.View(resources.ProcessWhitelist)): {
			"/v1.ProcessWhitelistService/GetProcessWhitelist",
		},
		user.With(permissions.Modify(resources.ProcessWhitelist)): {
			"/v1.ProcessWhitelistService/UpdateProcessWhitelists",
			"/v1.ProcessWhitelistService/LockProcessWhitelists",
		},
	})
)

type serviceImpl struct {
	dataStore         datastore.DataStore
	reprocessor       reprocessor.Loop
	connectionManager connection.Manager
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

func validateKeyNotEmpty(key *storage.ProcessWhitelistKey) error {
	if stringutils.AtLeastOneEmpty(
		key.GetDeploymentId(),
		key.GetContainerName(),
		key.GetClusterId(),
		key.GetNamespace(),
	) {
		return errors.New("invalid key: must specify both deployment id and container name")
	}
	return nil
}

func (s *serviceImpl) GetProcessWhitelist(ctx context.Context, request *v1.GetProcessWhitelistRequest) (*storage.ProcessWhitelist, error) {
	if err := validateKeyNotEmpty(request.GetKey()); err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	whitelist, exists, err := s.dataStore.GetProcessWhitelist(ctx, request.GetKey())
	if err != nil {
		return nil, err
	}
	if !exists {
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

func (s *serviceImpl) sendWhitelistToSensor(pw *storage.ProcessWhitelist) {
	if features.SensorBasedDetection.Enabled() {
		err := s.connectionManager.SendMessage(pw.GetKey().GetClusterId(), &central.MsgToSensor{
			Msg: &central.MsgToSensor_WhitelistSync{
				WhitelistSync: &central.WhitelistSync{
					Whitelists: []*storage.ProcessWhitelist{pw},
				}},
		})
		if err != nil {
			log.Errorf("Error sending process whitelist to cluster %q: %v", pw.GetKey().GetClusterId(), err)
		}
	}
}

func (s *serviceImpl) reprocessDeploymentRisks(keys []*storage.ProcessWhitelistKey) {
	deploymentIDs := set.NewStringSet()
	for _, key := range keys {
		deploymentIDs.Add(key.GetDeploymentId())
	}
	s.reprocessor.ReprocessRiskForDeployments(deploymentIDs.AsSlice()...)
}

func (s *serviceImpl) UpdateProcessWhitelists(ctx context.Context, request *v1.UpdateProcessWhitelistsRequest) (*v1.UpdateProcessWhitelistsResponse, error) {
	updateFunc := func(key *storage.ProcessWhitelistKey) (*storage.ProcessWhitelist, error) {
		return s.dataStore.UpdateProcessWhitelistElements(ctx, key, request.GetAddElements(), request.GetRemoveElements(), false)
	}
	defer s.reprocessDeploymentRisks(request.GetKeys())
	resp := bulkUpdate(request.GetKeys(), updateFunc)
	for _, w := range resp.GetWhitelists() {
		s.sendWhitelistToSensor(w)
	}
	return resp, nil
}

func (s *serviceImpl) LockProcessWhitelists(ctx context.Context, request *v1.LockProcessWhitelistsRequest) (*v1.UpdateProcessWhitelistsResponse, error) {
	updateFunc := func(key *storage.ProcessWhitelistKey) (*storage.ProcessWhitelist, error) {
		return s.dataStore.UserLockProcessWhitelist(ctx, key, request.GetLocked())
	}

	defer s.reprocessDeploymentRisks(request.GetKeys())
	resp := bulkUpdate(request.GetKeys(), updateFunc)
	for _, w := range resp.GetWhitelists() {
		s.sendWhitelistToSensor(w)
	}
	return resp, nil
}
