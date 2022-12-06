package phonehome

import (
	"context"
	"net"
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stackrox/rox/pkg/grpc/requestinfo"
	"github.com/stackrox/rox/pkg/set"
	"github.com/stackrox/rox/pkg/telemetry/phonehome/mocks"
	"github.com/stretchr/testify/suite"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/peer"
)

type interceptorTestSuite struct {
	suite.Suite

	mockTelemeter *mocks.MockTelemeter
}

var _ suite.SetupTestSuite = (*interceptorTestSuite)(nil)

func TestInterceptor(t *testing.T) {
	suite.Run(t, new(interceptorTestSuite))
}

func (s *interceptorTestSuite) SetupTest() {
	s.mockTelemeter = mocks.NewMockTelemeter(gomock.NewController(s.T()))
}

func (s *interceptorTestSuite) expect(path string, code int) {
	s.mockTelemeter.EXPECT().Track("API Call", "local:11102e5e-ca16-4f2b-8d2e-e9e04e8dc531:unauthenticated", map[string]any{
		"Path":       path,
		"Code":       code,
		"User-Agent": "test",
	})
}

func (s *interceptorTestSuite) testSpecific(path, allowed string) {
	rih := requestinfo.NewRequestInfoHandler()
	u, err := url.Parse("https://central" + path)
	s.NoError(err)

	md := rih.AnnotateMD(context.TODO(), &http.Request{URL: u})
	md.Set("User-Agent", "test")
	ctx := peer.NewContext(context.Background(), &peer.Peer{Addr: &net.UnixAddr{Net: "pipe"}})
	ctx, err = rih.UpdateContextForGRPC(metadata.NewIncomingContext(ctx, md))
	s.NoError(err)

	track(ctx, s.mockTelemeter, nil, nil, set.NewFrozenSet(strings.Split(allowed, ",")...))
}

func (s *interceptorTestSuite) TestInterceptorHttp() {
	s.testSpecific("/v1/one", "/v1/two")
	s.testSpecific("/v1/one", "/v1/two,/v1/three")

	s.expect("/v1/abc", 200)
	s.testSpecific("/v1/abc", "*")
	s.testSpecific("/v1/ping", "*")

	s.expect("/v1/pong", 200)
	s.testSpecific("/v1/pong", "/v1/pong")

	s.expect("/v1/four", 200)
	s.testSpecific("/v1/four", "/v1/two,/v1/three,/v1/four")
}

func (s *interceptorTestSuite) TestInterceptorGrpc() {
	s.expect("/v1.Abc", 0)

	md := metadata.New(nil)
	md.Set("User-Agent", "test")
	ctx := peer.NewContext(context.Background(), &peer.Peer{Addr: &net.UnixAddr{Net: "pipe"}})

	rih := requestinfo.NewRequestInfoHandler()
	ctx, err := rih.UpdateContextForGRPC(metadata.NewIncomingContext(ctx, md))
	s.NoError(err)

	track(ctx, s.mockTelemeter, nil, &grpc.UnaryServerInfo{
		FullMethod: "/v1.Abc",
	}, set.NewFrozenSet("*"))
}
