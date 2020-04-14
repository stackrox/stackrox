package service

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/gogo/protobuf/types"
	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"github.com/stackrox/rox/central/cve/datastore"
	"github.com/stackrox/rox/central/reprocessor"
	"github.com/stackrox/rox/central/role/resources"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/auth/permissions"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/dackbox/utils/queue"
	"github.com/stackrox/rox/pkg/grpc/authz"
	"github.com/stackrox/rox/pkg/grpc/authz/perrpc"
	"github.com/stackrox/rox/pkg/grpc/authz/user"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/set"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var (
	// TODO: Change the resource to CVE once SAC is in place
	authorizer = perrpc.FromMap(map[authz.Authorizer][]string{
		user.With(permissions.Modify(permissions.WithLegacyAuthForSAC(resources.Image, true))): {
			"/v1.CVEService/SuppressCVEs",
			"/v1.CVEService/UnsuppressCVEs",
		},
	})
)

// serviceImpl provides APIs for cves.
type serviceImpl struct {
	cves        datastore.DataStore
	indexQ      queue.WaitableQueue
	reprocessor reprocessor.Loop
}

// RegisterServiceServer registers this service with the given gRPC Server.
func (s *serviceImpl) RegisterServiceServer(grpcServer *grpc.Server) {
	v1.RegisterCVEServiceServer(grpcServer, s)
}

// RegisterServiceHandler registers this service with the given gRPC Gateway endpoint.
func (s *serviceImpl) RegisterServiceHandler(ctx context.Context, mux *runtime.ServeMux, conn *grpc.ClientConn) error {
	return v1.RegisterCVEServiceHandler(ctx, mux, conn)
}

// AuthFuncOverride specifies the auth criteria for this API.
func (s *serviceImpl) AuthFuncOverride(ctx context.Context, fullMethodName string) (context.Context, error) {
	return ctx, authorizer.Authorized(ctx, fullMethodName)
}

// SuppressCVE suppresses cves for specific duration or indefinitely.
func (s *serviceImpl) SuppressCVEs(ctx context.Context, request *v1.SuppressCVERequest) (*v1.Empty, error) {
	activation := types.TimestampNow()
	if err := s.validateCVEsExist(ctx, request.GetIds()...); err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	if err := s.cves.Suppress(ctx, activation, request.GetDuration(), request.GetIds()...); err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	if err := s.waitForCVEToBeIndexed(ctx); err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	s.reprocessDeployments()

	return &v1.Empty{}, nil
}

// UnsuppressCVE unsuppresses given cves indefinitely.
func (s *serviceImpl) UnsuppressCVEs(ctx context.Context, request *v1.UnsuppressCVERequest) (*v1.Empty, error) {
	if err := s.validateCVEsExist(ctx, request.GetIds()...); err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	if err := s.cves.Unsuppress(ctx, request.GetIds()...); err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	if err := s.waitForCVEToBeIndexed(ctx); err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	s.reprocessDeployments()

	return &v1.Empty{}, nil
}

func (s *serviceImpl) waitForCVEToBeIndexed(ctx context.Context) error {
	cveSynchronized := concurrency.NewSignal()
	s.indexQ.PushSignal(&cveSynchronized)

	select {
	case <-ctx.Done():
		return errors.New("timed out waiting for indexing")
	case <-cveSynchronized.Done():
		return nil
	}
}

func (s *serviceImpl) reprocessDeployments() {
	s.reprocessor.ShortCircuit()
}

func (s *serviceImpl) validateCVEsExist(ctx context.Context, ids ...string) error {
	result, err := s.cves.Search(ctx, search.NewQueryBuilder().AddDocIDs(ids...).ProtoQuery())
	if err != nil {
		return status.Error(codes.Internal, err.Error())
	}

	if len(result) < len(ids) {
		missingIds := set.NewStringSet(ids...).Difference(search.ResultsToIDSet(result))
		return status.Error(codes.NotFound, fmt.Sprintf("Following CVEs not found: %s", strings.Join(missingIds.AsSlice(), ", ")))
	}
	return nil
}
