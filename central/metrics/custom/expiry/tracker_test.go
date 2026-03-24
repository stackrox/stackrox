package expiry

import (
	"context"
	"errors"
	"testing"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/stackrox/rox/central/credentialexpiry/service"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc"
)

type mockService struct {
	err error
}

var errTest = errors.New("test")

func (ms *mockService) AuthFuncOverride(context.Context, string) (context.Context, error) {
	panic("unimplemented")
}

func (ms *mockService) GetCertExpiry(_ context.Context, req *v1.GetCertExpiry_Request) (*v1.GetCertExpiry_Response, error) {
	if req.GetComponent() == v1.GetCertExpiry_SCANNER_V4 {
		return nil, errTest
	}
	return nil, nil
}

func (ms *mockService) RegisterServiceHandler(context.Context, *runtime.ServeMux, *grpc.ClientConn) error {
	panic("unimplemented")
}

func (ms *mockService) RegisterServiceServer(*grpc.Server) {
	panic("unimplemented")
}

var _ service.Service = (*mockService)(nil)

func Test_track(t *testing.T) {
	var s mockService

	components := make([]string, 0, len(v1.GetCertExpiry_Component_name))
	for f := range track(context.Background(), &s) {
		components = append(components, f.component)
	}
	assert.ElementsMatch(t, []string{"SCANNER", "CENTRAL_DB", "CENTRAL"}, components,
		"should have no UNKNOWN and SCANNER_V4")
}
