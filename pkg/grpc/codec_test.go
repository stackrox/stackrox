package grpc

import (
	"context"
	"errors"
	"fmt"
	"net"
	"testing"

	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/encoding"
	"google.golang.org/grpc/encoding/proto"
	"google.golang.org/grpc/test/bufconn"
	"google.golang.org/protobuf/types/known/durationpb"
)

func TestCodecFallback(t *testing.T) {
	c := encoding.GetCodecV2(proto.Name)

	nonVtMessage := durationpb.New(1)

	data, err := c.Marshal(nonVtMessage)
	assert.NoError(t, err)
	assert.NotEmpty(t, data)

	err = c.Unmarshal(data, nonVtMessage)
	assert.NoError(t, err)

	_, err = c.Marshal(nil)
	assert.EqualError(t, err, "codec failed: type <nil> does not support VT; fallback failed: proto: failed to marshal, message is <nil>, want proto.Message")

	err = c.Unmarshal(data, nil)
	assert.EqualError(t, err, "type <nil> does not support VT; fallback failed: failed to unmarshal, message is <nil>, want proto.Message")

	_, err = c.Marshal(fakeVtMsg{})
	assert.EqualError(t, err, "codec failed: some error; fallback failed: proto: failed to marshal, message is grpc.fakeVtMsg, want proto.Message")

	err = c.Unmarshal(data, fakeVtMsg{})
	assert.EqualError(t, err, "codec failed: some error; fallback failed: failed to unmarshal, message is grpc.fakeVtMsg, want proto.Message")

	_, err = c.Marshal(errVtMsg{nonVtMessage})
	assert.NoError(t, err)

	err = c.Unmarshal(data, errVtMsg{nonVtMessage})
	assert.NoError(t, err)
}

type fakeVtMsg struct {
}

func (fakeVtMsg) MarshalToSizedBufferVT(_ []byte) (int, error) {
	return 0, errors.New("some error")
}

func (fakeVtMsg) UnmarshalVT([]byte) error {
	return errors.New("some error")
}

func (fakeVtMsg) SizeVT() int {
	return 0
}

type errVtMsg struct {
	*durationpb.Duration
}

func (errVtMsg) MarshalToSizedBufferVT(_ []byte) (int, error) {
	return 0, errors.New("some error")
}

func (errVtMsg) UnmarshalVT([]byte) error {
	return errors.New("some error")
}

func (errVtMsg) SizeVT() int {
	return 0
}

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
