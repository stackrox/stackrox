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

type anonymous struct{}

var (
	_ Method                        = (*anonymous)(nil)
	_ credentials.PerRPCCredentials = (*anonymous)(nil)
)

// AnonymousAuth provides an auth.Method for anonymous access.
func AnonymousAuth() Method {
	return &anonymous{}
}

func (a anonymous) Type() string {
	return "anonymous"
}

func (a anonymous) GetCredentials(_ string) (credentials.PerRPCCredentials, error) {
	return &a, nil
}

func (a anonymous) RequireTransportSecurity() bool {
	return false
}

func (a anonymous) GetRequestMetadata(_ context.Context, _ ...string) (map[string]string, error) {
	return nil, nil
}
