package auth

import (
	"context"

	"google.golang.org/grpc/credentials"
)

// Method describes the method of authentication.
type Method interface {
	Type() string
	GetCredentials(url string) (credentials.PerRPCCredentials, error)
}

type anonymousMethod struct{}

var (
	_ Method = (*anonymousMethod)(nil)
)

// Anonymous provides an anonymous auth.Method, in case no authentication is desired.
func Anonymous() Method {
	return &anonymousMethod{}
}

func (a anonymousMethod) Type() string {
	return "anonymous"
}

func (a anonymousMethod) GetCredentials(_ string) (credentials.PerRPCCredentials, error) {
	return &a, nil
}

func (a anonymousMethod) RequireTransportSecurity() bool {
	return false
}

func (a anonymousMethod) GetRequestMetadata(_ context.Context, _ ...string) (map[string]string, error) {
	return nil, nil
}
