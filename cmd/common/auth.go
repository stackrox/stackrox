package common

import (
	"github.com/stackrox/rox/pkg/clientconn"
	"github.com/stackrox/rox/pkg/env"
	"google.golang.org/grpc"
)

// GetGRPCConnection gets the correct GRPC connection depending on the environment
func GetGRPCConnection(endpoint string) (*grpc.ClientConn, error) {
	if token := env.TokenEnv.Setting(); token != "" {
		return clientconn.GRPCConnectionWithToken(endpoint, token)
	}
	return clientconn.UnauthenticatedGRPCConnection(endpoint)
}
