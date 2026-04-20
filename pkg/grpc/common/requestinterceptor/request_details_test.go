package requestinterceptor

import (
	"bytes"
	"context"
	"net"
	"net/http"
	"testing"

	"github.com/stackrox/rox/pkg/grpc/authn"
	idmocks "github.com/stackrox/rox/pkg/grpc/authn/mocks"
	"github.com/stackrox/rox/pkg/grpc/requestinfo"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/peer"
)

func withUserAgent(ua string) http.Header {
	return http.Header{userAgentHeaderKey: {ua}}
}

type requestDetailsTestSuite struct {
	suite.Suite
	ctrl *gomock.Controller
}

func TestRequestDetails(t *testing.T) {
	suite.Run(t, new(requestDetailsTestSuite))
}

func (s *requestDetailsTestSuite) SetupTest() {
	s.ctrl = gomock.NewController(s.T())
}

func (s *requestDetailsTestSuite) TestGrpcRequestInfo() {
	testRP := &RequestParams{
		Code:    0,
		Path:    "/v1.Test",
		Headers: withUserAgent("test"),
	}

	md := metadata.New(nil)
	md.Set(userAgentHeaderKey, testRP.Headers.Values(userAgentHeaderKey)...)
	ctx := peer.NewContext(context.Background(), &peer.Peer{Addr: &net.UnixAddr{Net: "pipe"}})

	rih := requestinfo.NewRequestInfoHandler()
	ctx, err := rih.UpdateContextForGRPC(metadata.NewIncomingContext(ctx, md))
	s.NoError(err)

	rp := getGRPCRequestDetails(ctx, err, testRP.Path, "request")
	s.Equal(testRP.Path, rp.Path)
	s.Equal(testRP.Code, rp.Code)
	s.Nil(rp.UserID)
	s.Equal("request", rp.GRPCReq)
	s.Equal(testRP.Headers.Values(userAgentHeaderKey), rp.Headers.Values(userAgentHeaderKey))

	// Verify that gRPC metadata lowercase keys are canonicalized so that
	// glob patterns with canonical case (e.g., "User-Agent") match.
	ua := rp.Headers.Values("User-Agent")
	s.NoError(err)
	s.Equal([]string{"test"}, ua)
}

func (s *requestDetailsTestSuite) TestGrpcWithHTTPRequestInfo() {
	req, _ := http.NewRequest(http.MethodPatch, "/wrapped/http", nil)
	req.Header.Add(userAgentHeaderKey, "user")
	rih := requestinfo.NewRequestInfoHandler()
	ctx := peer.NewContext(context.Background(), &peer.Peer{Addr: &net.UnixAddr{Net: "pipe"}})
	md := rih.AnnotateMD(ctx, req)
	// Simulate the gRPC transport User-Agent (set via grpc.WithUserAgent).
	md.Set(userAgentHeaderKey, "gateway")

	ctx, err := rih.UpdateContextForGRPC(metadata.NewIncomingContext(ctx, md))
	s.NoError(err)

	rp := getGRPCRequestDetails(ctx, err, "ignored grpc method", "request")
	s.Equal(http.StatusOK, rp.Code)
	// Original HTTP User-Agent + gRPC transport agent merged under one key.
	s.Equal([]string{"user", "gateway"}, rp.Headers.Values(userAgentHeaderKey))
	s.Nil(rp.UserID)
	s.Equal("request", rp.GRPCReq)
	s.Equal("/wrapped/http", rp.Path)
	s.Equal(http.MethodPatch, rp.Method)
}

func (s *requestDetailsTestSuite) TestGrpcWithHTTPRequestInfo_UserAgentVariants() {
	cases := map[string]struct {
		httpUserAgent []string // User-Agent values on the HTTP request.
		mdUserAgent   []string // User-Agent values in gRPC metadata (transport agent).
		expected      []string // Expected merged User-Agent values in the result.
	}{
		"HTTP and gRPC transport User-Agent": {
			httpUserAgent: []string{"curl/8.0"},
			mdUserAgent:   []string{"Rox Central/4.11 grpc-go/1.80.0"},
			expected:      []string{"curl/8.0", "Rox Central/4.11 grpc-go/1.80.0"},
		},
		"only HTTP User-Agent": {
			httpUserAgent: []string{"curl/8.0"},
			expected:      []string{"curl/8.0"},
		},
		"only gRPC transport User-Agent": {
			mdUserAgent: []string{"grpc-go/1.80.0"},
			expected:    []string{"grpc-go/1.80.0"},
		},
		"no User-Agent anywhere": {
			expected: nil,
		},
	}
	for name, tc := range cases {
		s.Run(name, func() {
			req, _ := http.NewRequest(http.MethodGet, "/test", nil)
			req.Header.Del(userAgentHeaderKey)
			for _, ua := range tc.httpUserAgent {
				req.Header.Add(userAgentHeaderKey, ua)
			}
			rih := requestinfo.NewRequestInfoHandler()
			ctx := peer.NewContext(context.Background(), &peer.Peer{Addr: &net.UnixAddr{Net: "pipe"}})
			md := rih.AnnotateMD(ctx, req)
			if tc.mdUserAgent != nil {
				md.Set(userAgentHeaderKey, tc.mdUserAgent...)
			}

			ctx, err := rih.UpdateContextForGRPC(metadata.NewIncomingContext(ctx, md))
			s.NoError(err)

			rp := getGRPCRequestDetails(ctx, err, "ignored", "request")
			s.Equal(tc.expected, rp.Headers.Values(userAgentHeaderKey))
		})
	}
}

type testBody struct {
	N int `json:"n"`
}

type testBodyI interface {
	getTestBody(context.Context, *testBody) (*any, error)
}

func (s *requestDetailsTestSuite) TestHttpWithBody() {
	body := "{ \"n\": 42 }"
	req, _ := http.NewRequest(http.MethodPost, "/http/body", bytes.NewReader([]byte(body)))
	rp := getHTTPRequestDetails(context.Background(), req, 0)

	rb := GetGRPCRequestBody(testBodyI.getTestBody, rp)
	s.Nil(rb, "body is not captured for HTTP requests")
}

func (s *requestDetailsTestSuite) TestGrpcWithBody() {
	rp := getGRPCRequestDetails(context.Background(), nil, "/grpc/body", &testBody{N: 42})

	rb := GetGRPCRequestBody(testBodyI.getTestBody, rp)
	if s.NotNil(rb) {
		s.Equal(42, rb.N)
	}

	rp = getGRPCRequestDetails(context.Background(), nil, "/grpc/body", nil)

	rb = GetGRPCRequestBody(testBodyI.getTestBody, rp)
	s.Nil(rb)
}

func (s *requestDetailsTestSuite) TestHttpRequestInfo() {
	mockID := idmocks.NewMockIdentity(s.ctrl)
	testRP := &RequestParams{
		UserID:  mockID,
		Code:    200,
		Headers: withUserAgent("test"),
		Path:    "/v1/test",
	}

	req, err := http.NewRequest(http.MethodPost, "https://test"+testRP.Path+"?test_key=test_value", nil)
	s.NoError(err)
	req.Header.Add(userAgentHeaderKey, testRP.Headers.Get(userAgentHeaderKey))

	ctx := authn.ContextWithIdentity(context.Background(), testRP.UserID, nil)
	rp := getHTTPRequestDetails(ctx, req, 200)
	s.Equal(testRP.Path, rp.Path)
	s.Equal(testRP.Code, rp.Code)
	s.Equal(mockID, rp.UserID)
	s.Equal(testRP.Headers.Values(userAgentHeaderKey), rp.Headers.Values(userAgentHeaderKey))
}
