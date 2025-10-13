package basic

import (
	"context"
	"encoding/base64"
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
func PerRPCCredentials(username, password string) credentials.PerRPCCredentials {
	tokenRaw := fmt.Sprintf("%s:%s", username, password)
	authHeaderValue := fmt.Sprintf("Basic %s", base64.StdEncoding.EncodeToString([]byte(tokenRaw)))

	metadata := map[string]string{
		"authorization": authHeaderValue,
	}
	return perRPCCreds{
		metadata: metadata,
	}
}
