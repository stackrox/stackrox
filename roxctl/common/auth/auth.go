package auth

import (
	"context"
	"fmt"

	"google.golang.org/grpc/credentials"
)

// Method bla
type Method interface {
	Name() string
	GetCreds(baseURL string) (credentials.PerRPCCredentials, error)
}

type anonymous struct{}

func (anonymous) Name() string {
	return "anonymous"
}

func (anonymous) GetCreds(_ string) (credentials.PerRPCCredentials, error) {
	return anonymous{}, nil
}

func (anonymous) RequireTransportSecurity() bool {
	return false
}

func (anonymous) GetRequestMetadata(ctx context.Context, uri ...string) (map[string]string, error) {
	return nil, nil
}

// Anonymous returns
func Anonymous() Method {
	return anonymous{}
}

type errorMethod struct {
	err error
}

func (m errorMethod) Name() string {
	return fmt.Sprintf("ERROR: %s", m.err)
}

func (m errorMethod) GetCreds(_ string) (credentials.PerRPCCredentials, error) {
	return nil, m.err
}

// Error returns
func Error(err error) Method {
	return errorMethod{
		err: err,
	}
}
