package tokenbased

import (
	"context"
	"fmt"

	"google.golang.org/grpc/credentials"
)

type perRPCCreds struct {
	metadata map[string]string
}

func (c perRPCCreds) GetRequestMetadata(_ context.Context, _ ...string) (map[string]string, error) {
	return c.metadata, nil
}

func (c perRPCCreds) RequireTransportSecurity() bool {
	return true
}

// PerRPCCredentials returns per-RPC credentials using the given username and password for basic auth.
func PerRPCCredentials(token string) credentials.PerRPCCredentials {
	authHeaderValue := fmt.Sprintf("Bearer %s", token)
	metadata := map[string]string{
		"authorization": authHeaderValue,
	}
	return perRPCCreds{
		metadata: metadata,
	}
}
