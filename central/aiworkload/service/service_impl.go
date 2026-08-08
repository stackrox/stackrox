package service

import (
	"context"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	aiWorkloadDataStore "github.com/stackrox/rox/central/aiworkload/datastore"
	"github.com/stackrox/rox/central/convert/storagetov2"
	v2 "github.com/stackrox/rox/generated/api/v2"
	"github.com/stackrox/rox/pkg/auth/permissions"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/grpc/authz"
	"github.com/stackrox/rox/pkg/grpc/authz/perrpc"
	"github.com/stackrox/rox/pkg/grpc/authz/user"
	"github.com/stackrox/rox/pkg/sac/resources"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/search/paginated"
	"google.golang.org/grpc"
)

const defaultPageSize = 100

var authorizer = perrpc.FromMap(map[authz.Authorizer][]string{
	user.With(permissions.View(resources.AIWorkload)): {
		"/v2.AIWorkloadService/GetAIWorkload",
		"/v2.AIWorkloadService/ListAIWorkloads",
	},
})

type serviceImpl struct {
	v2.UnimplementedAIWorkloadServiceServer
	datastore aiWorkloadDataStore.DataStore
}

func New(datastore aiWorkloadDataStore.DataStore) Service {
	return &serviceImpl{datastore: datastore}
}

func (s *serviceImpl) RegisterServiceServer(server *grpc.Server) {
	v2.RegisterAIWorkloadServiceServer(server, s)
}

func (s *serviceImpl) RegisterServiceHandler(ctx context.Context, mux *runtime.ServeMux, conn *grpc.ClientConn) error {
	return v2.RegisterAIWorkloadServiceHandler(ctx, mux, conn)
}

func (s *serviceImpl) AuthFuncOverride(ctx context.Context, fullMethodName string) (context.Context, error) {
	return ctx, authorizer.Authorized(ctx, fullMethodName)
}

func (s *serviceImpl) GetAIWorkload(ctx context.Context, req *v2.GetAIWorkloadRequest) (*v2.AIWorkload, error) {
	workload, found, err := s.datastore.GetAIWorkload(ctx, req.GetId())
	if err != nil {
		return nil, err
	}
	if !found {
		return nil, errox.NotFound.Newf("AI workload %q not found", req.GetId())
	}
	return storagetov2.AIWorkload(workload), nil
}

func (s *serviceImpl) ListAIWorkloads(ctx context.Context, req *v2.ListAIWorkloadsRequest) (*v2.ListAIWorkloadsResponse, error) {
	searchQuery, err := search.ParseQuery(req.GetQuery().GetQuery(), search.MatchAllIfEmpty())
	if err != nil {
		return nil, err
	}

	countQuery := searchQuery.CloneVT()
	count, err := s.datastore.CountAIWorkloads(ctx, countQuery)
	if err != nil {
		return nil, err
	}

	paginated.FillPaginationV2(searchQuery, req.GetQuery().GetPagination(), defaultPageSize)
	workloads, err := s.datastore.SearchRawAIWorkloads(ctx, searchQuery)
	if err != nil {
		return nil, err
	}

	result := make([]*v2.AIWorkload, 0, len(workloads))
	for _, w := range workloads {
		result = append(result, storagetov2.AIWorkload(w))
	}
	return &v2.ListAIWorkloadsResponse{
		AiWorkloads: result,
		TotalCount:  int32(count),
	}, nil
}
