//go:build sql_integration

package tests

import (
	"context"
	"net"

	"github.com/stackrox/rox/pkg/sac"
	"google.golang.org/grpc/test/bufconn"
)

func allAccessCtx() context.Context {
	return sac.WithAllAccess(context.Background())
}

func bufconnDialer(lis *bufconn.Listener) func(context.Context, string) (net.Conn, error) {
	return func(ctx context.Context, _ string) (net.Conn, error) {
		return lis.DialContext(ctx)
	}
}
