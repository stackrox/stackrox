package service

import (
	"context"
	"strings"

	"github.com/gogo/protobuf/types"
	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/cve/datastore"
	"github.com/stackrox/rox/central/role/resources"
	vulnReqMgr "github.com/stackrox/rox/central/vulnerabilityrequest/manager/requestmgr"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/auth/permissions"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/dackbox/utils/queue"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/grpc/authz"
	"github.com/stackrox/rox/pkg/grpc/authz/and"
	"github.com/stackrox/rox/pkg/grpc/authz/perrpc"
	"github.com/stackrox/rox/pkg/grpc/authz/user"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/set"
	"google.golang.org/grpc"
)

var (
	authorizer = func() authz.Authorizer {
		return perrpc.FromMap(map[authz.Authorizer][]string{
			and.And(
				user.With(permissions.Modify(resources.VulnerabilityManagementRequests)),
				user.With(permissions.Modify(resources.VulnerabilityManagementApprovals))): {
				"/v1.CVEService/SuppressCVEs",
				"/v1.CVEService/UnsuppressCVEs",
			},
		})
	}()
)

// serviceImpl provides APIs for CVEs.
type serviceImpl struct {
	cves       datastore.DataStore
	vulnReqMgr vulnReqMgr.Manager
	indexQ     queue.WaitableQueue
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

// SuppressCVEs suppresses CVEs for specific duration or indefinitely.
func (s *serviceImpl) SuppressCVEs(ctx context.Context, request *v1.SuppressCVERequest) (*v1.Empty, error) {
	createdAt := types.TimestampNow()
	if err := s.validateCVEsExist(ctx, request.GetIds()...); err != nil {
		return nil, err
	}

	if err := s.cves.Suppress(ctx, createdAt, request.GetDuration(), request.GetIds()...); err != nil {
		return nil, err
	}

	if err := s.waitForCVEToBeIndexed(ctx); err != nil {
		return nil, err
	}

	// This handles updating image-cve edges and reprocessing affected deployments.
	if err := s.vulnReqMgr.SnoozeVulnerabilityOnRequest(ctx, suppressCVEReqToVulnReq(request, createdAt)); err != nil {
		log.Error(err)
	}
	return &v1.Empty{}, nil
}

// UnsuppressCVEs unsuppresses given CVEs indefinitely.
func (s *serviceImpl) UnsuppressCVEs(ctx context.Context, request *v1.UnsuppressCVERequest) (*v1.Empty, error) {
	if err := s.validateCVEsExist(ctx, request.GetIds()...); err != nil {
		return nil, err
	}

	if err := s.cves.Unsuppress(ctx, request.GetIds()...); err != nil {
		return nil, err
	}

	if err := s.waitForCVEToBeIndexed(ctx); err != nil {
		return nil, err
	}

	// This handles updating image-cve edges and reprocessing affected deployments.
	if err := s.vulnReqMgr.UnSnoozeVulnerabilityOnRequest(ctx, unSuppressCVEReqToVulnReq(request)); err != nil {
		log.Error(err)
	}
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

func (s *serviceImpl) validateCVEsExist(ctx context.Context, ids ...string) error {
	result, err := s.cves.Search(ctx, search.NewQueryBuilder().AddDocIDs(ids...).ProtoQuery())
	if err != nil {
		return err
	}

	if len(result) < len(ids) {
		missingIds := set.NewStringSet(ids...).Difference(search.ResultsToIDSet(result))
		return errors.Wrapf(errox.NotFound, "Following CVEs not found: %s", strings.Join(missingIds.AsSlice(), ", "))
	}
	return nil
}
