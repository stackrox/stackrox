package storage

import (
	"context"

	"cloud.google.com/go/storage"
	"github.com/stackrox/rox/pkg/httputil/proxy"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/option"
	"google.golang.org/grpc"
)

// ClientFactory creates a GCP storage client.
//
//go:generate mockgen-wrapper
type ClientFactory interface {
	NewClient(ctx context.Context, creds *google.Credentials) (*storage.Client, error)
}

var _ ClientFactory = &clientFactoryImpl{}

type clientFactoryImpl struct{}

func (s *clientFactoryImpl) NewClient(ctx context.Context, creds *google.Credentials) (*storage.Client, error) {
	return storage.NewGRPCClient(ctx,
		option.WithGRPCDialOption(grpc.WithContextDialer(proxy.AwareDialContext)),
		option.WithCredentials(creds),
	)
}
