package phonehome

import (
	"context"
	"net"
	"net/http"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stackrox/rox/pkg/grpc/authn"
	idmocks "github.com/stackrox/rox/pkg/grpc/authn/mocks"
	"github.com/stackrox/rox/pkg/grpc/requestinfo"
	"github.com/stackrox/rox/pkg/telemetry/phonehome/mocks"
	"github.com/stretchr/testify/suite"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/peer"
)

type interceptorTestSuite struct {
	suite.Suite

	mockTelemeter *mocks.MockTelemeter
	ctrl          *gomock.Controller
}

var _ suite.SetupTestSuite = (*interceptorTestSuite)(nil)

func TestInterceptor(t *testing.T) {
	suite.Run(t, new(interceptorTestSuite))
}

func (s *interceptorTestSuite) SetupTest() {
	s.ctrl = gomock.NewController(s.T())
	s.mockTelemeter = mocks.NewMockTelemeter(s.ctrl)
}

type testRequest struct {
	value string
}

func (s *interceptorTestSuite) TestAddGrpcInterceptor() {
	testRP := &RequestParams{
		Path:      "/v1.Abc",
		Code:      0,
		UserAgent: "test",
		UserID:    nil,
		GrpcReq: &testRequest{
			value: "test value",
		},
	}
	cfg := &Config{
		ClientID:  "test",
		telemeter: s.mockTelemeter,
	}

	cfg.AddInterceptorFunc("TestEvent", func(rp *RequestParams, props map[string]any) bool {
		if rp.Path == testRP.Path {
			if tr, ok := rp.GrpcReq.(*testRequest); ok {
				props["Property"] = tr.value
			}
		}
		return true
	})

	s.mockTelemeter.EXPECT().Track("TestEvent", cfg.HashUserAuthID(nil), map[string]any{
		"Property": "test value",
	}).Times(1)

	cfg.track(testRP)
}

func (s *interceptorTestSuite) TestAddHttpInterceptor() {
	mockID := idmocks.NewMockIdentity(s.ctrl)
	testRP := &RequestParams{
		Path:      "/v1/abc",
		Code:      200,
		UserAgent: "test",
		UserID:    mockID,
	}
	req, err := http.NewRequest(http.MethodPost, "https://test"+testRP.Path+"?test_key=test_value", nil)
	s.NoError(err)
	testRP.HttpReq = req
	cfg := &Config{
		ClientID:  "test",
		telemeter: s.mockTelemeter,
	}

	cfg.AddInterceptorFunc("TestEvent", func(rp *RequestParams, props map[string]any) bool {
		if rp.Path == testRP.Path {
			props["Property"] = rp.HttpReq.FormValue("test_key")
		}
		return true
	})

	mockID.EXPECT().ExternalAuthProvider().Return(nil).Times(2)
	mockID.EXPECT().UID().Return("id").Times(2)
	s.mockTelemeter.EXPECT().Track("TestEvent", cfg.HashUserAuthID(mockID), map[string]any{
		"Property": "test_value",
	}).Times(1)

	cfg.track(testRP)
}

func (s *interceptorTestSuite) TestGrpcRequestInfo() {
	testRP := &RequestParams{
		UserID:    nil,
		Code:      0,
		UserAgent: "test",
		Path:      "/v1.Test",
	}

	md := metadata.New(nil)
	md.Set("User-Agent", testRP.UserAgent)
	ctx := peer.NewContext(context.Background(), &peer.Peer{Addr: &net.UnixAddr{Net: "pipe"}})

	rih := requestinfo.NewRequestInfoHandler()
	ctx, err := rih.UpdateContextForGRPC(metadata.NewIncomingContext(ctx, md))
	s.NoError(err)

	rp := getGrpcRequestDetails(ctx, err, &grpc.UnaryServerInfo{
		FullMethod: testRP.Path,
	}, "request")
	s.Equal(testRP.Path, rp.Path)
	s.Equal(testRP.Code, rp.Code)
	s.Equal(testRP.UserAgent, rp.UserAgent)
	s.Nil(rp.UserID)
	s.Equal("request", rp.GrpcReq)
}

func (s *interceptorTestSuite) TestHttpRequestInfo() {
	mockID := idmocks.NewMockIdentity(s.ctrl)
	testRP := &RequestParams{
		UserID:    mockID,
		Code:      200,
		UserAgent: "test",
		Path:      "/v1/test",
	}

	req, err := http.NewRequest(http.MethodPost, "https://test"+testRP.Path+"?test_key=test_value", nil)
	s.NoError(err)
	req.Header.Add("User-Agent", testRP.UserAgent)

	ctx := authn.ContextWithIdentity(context.Background(), testRP.UserID, nil)
	rp := getHttpRequestDetails(ctx, req, err)
	s.Equal(testRP.Path, rp.Path)
	s.Equal(testRP.Code, rp.Code)
	s.Equal(testRP.UserAgent, rp.UserAgent)
	s.Equal(mockID, rp.UserID)
}
