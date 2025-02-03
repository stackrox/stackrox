package grpc

import (
	"context"
	"fmt"
	"net"
	"testing"

	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/test/bufconn"
	"google.golang.org/protobuf/types/known/durationpb"
)

func TestCodec(t *testing.T) {
	svc := getClientForServer(t)

	// create a small message that will be below (<=)
	// buffer pooling threshold
	request := v1.SuppressCVERequest{
		Cves: []string{"ABC", "XYZ"},
		Duration: &durationpb.Duration{
			Seconds: 100,
			Nanos:   0,
		},
	}

	_, err := svc.SuppressCVEs(context.Background(), &request)
	assert.Error(t, err)

	// create a big message that will be above (>)
	// buffer pooling threshold
	for i := 0; i < 1<<10; i++ {
		request.Cves = append(request.Cves, fmt.Sprintf("CVE-%d", i))
	}
	_, err = svc.SuppressCVEs(context.Background(), &request)
	assert.Error(t, err)
}

func BenchmarkProtoUnmarshal(b *testing.B) {
	svc := getClientForServer(b)

	request := v1.SuppressCVERequest{
		Cves: []string{"ABC", "XYZ"},
		Duration: &durationpb.Duration{
			Seconds: 100,
			Nanos:   0,
		},
	}

	_, err := svc.SuppressCVEs(context.Background(), &request)
	require.EqualError(b, err, "rpc error: code = Canceled desc = ABC, XYZ, 1m40s")

	b.Run("small", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, _ = svc.SuppressCVEs(context.Background(), &request)
		}
	})

	b.Run("big", func(b *testing.B) {
		for i := 0; i < 1<<10; i++ {
			request.Cves = append(request.Cves, fmt.Sprintf("CVE-%d", i))
		}
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, _ = svc.SuppressCVEs(context.Background(), &request)
		}
	})
}

func getClientForServer(t testing.TB) v1.NodeCVEServiceClient {
	grpcServiceHandler := &supressCveServiceTestErrorImpl{}

	buffer := 1024 * 1024
	listener := bufconn.Listen(buffer)

	s := grpc.NewServer()
	v1.RegisterNodeCVEServiceServer(s, grpcServiceHandler)
	go func() {
		utils.IgnoreError(func() error { return s.Serve(listener) })
	}()

	conn, _ := grpc.DialContext(context.Background(), "", grpc.WithContextDialer(func(ctx context.Context, s string) (net.Conn, error) {
		return listener.Dial()
	}), grpc.WithTransportCredentials(insecure.NewCredentials()))

	t.Cleanup(func() {
		utils.IgnoreError(listener.Close)
		s.Stop()
	})

	svc := v1.NewNodeCVEServiceClient(conn)
	return svc
}
