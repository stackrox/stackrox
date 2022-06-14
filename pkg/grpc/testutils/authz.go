package testutils

import (
	"context"
	"fmt"
	"testing"

	grpc_auth "github.com/grpc-ecosystem/go-grpc-middleware/auth"
	pkgGRPC "github.com/stackrox/stackrox/pkg/grpc"
	"github.com/stackrox/stackrox/pkg/grpc/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
)

var (
	allowedAuthStatusCodes = []codes.Code{codes.OK, codes.Unauthenticated, codes.PermissionDenied}
)

func allMethods(service pkgGRPC.APIService) []string {
	server := grpc.NewServer()
	service.RegisterServiceServer(server)

	var result []string
	for serviceName, serviceInfo := range server.GetServiceInfo() {
		for _, method := range serviceInfo.Methods {
			result = append(result, fmt.Sprintf("/%s/%s", serviceName, method.Name))
		}
	}
	return result
}

// AssertAuthzWorks tests that all methods declared by your service can be authorized against.
func AssertAuthzWorks(t *testing.T, service pkgGRPC.APIService) {
	authFunc, ok := service.(grpc_auth.ServiceAuthFuncOverride)
	require.True(t, ok, "service must implement an AuthFuncOverride method")

	for _, method := range allMethods(service) {
		_, err := authFunc.AuthFuncOverride(context.Background(), method)
		s := errors.ErrToGrpcStatus(err)
		assert.Containsf(t, allowedAuthStatusCodes, s.Code(), "authorizing method %s: invalid auth error code %v from error %v", method, s.Code(), s)
	}
}
