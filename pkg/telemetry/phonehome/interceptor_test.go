package phonehome

import (
	"bytes"
	"context"
	"fmt"
	"net"
	"net/http"
	"testing"

	"github.com/stackrox/rox/pkg/grpc/authn"
	idmocks "github.com/stackrox/rox/pkg/grpc/authn/mocks"
	"github.com/stackrox/rox/pkg/grpc/requestinfo"
	"github.com/stackrox/rox/pkg/telemetry/phonehome/telemeter"
	"github.com/stackrox/rox/pkg/telemetry/phonehome/telemeter/mocks"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
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

type optionsMatcher struct {
	opts *telemeter.CallOptions
}

func (m *optionsMatcher) Matches(x any) bool {
	if o, ok := x.([]telemeter.Option); ok {
		return gomock.Eq(m.opts).Matches(telemeter.ApplyOptions(o))
	}
	return false
}

func (m *optionsMatcher) String() string {
	return fmt.Sprint(m.opts)
}

func matchOptions(opts ...telemeter.Option) gomock.Matcher {
	return &optionsMatcher{telemeter.ApplyOptions(opts)}
}

func (s *interceptorTestSuite) TestAddGrpcInterceptor() {
	testRP := &RequestParams{
		Path:      "/v1.Abc",
		Code:      0,
		UserAgent: "test",
		UserID:    nil,
		GRPCReq: &testRequest{
			value: "test value",
		},
	}
	cfg := &Config{
		ClientID:  "test",
		GroupType: "TEST",
		telemeter: s.mockTelemeter,
	}

	cfg.AddInterceptorFunc("TestEvent", func(rp *RequestParams, props map[string]any) bool {
		if rp.Path == testRP.Path {
			if tr, ok := rp.GRPCReq.(*testRequest); ok {
				props["Property"] = tr.value
			}
		}
		return true
	})

	s.mockTelemeter.EXPECT().Track("TestEvent", map[string]any{
		"Property": "test value",
	}, matchOptions(telemeter.WithUserID(cfg.HashUserAuthID(nil)), telemeter.WithGroups("TEST", ""))).Times(1)

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
	testRP.HTTPReq = req
	cfg := &Config{
		ClientID:  "test",
		GroupType: "TEST",
		telemeter: s.mockTelemeter,
	}

	cfg.AddInterceptorFunc("TestEvent", func(rp *RequestParams, props map[string]any) bool {
		if rp.Path == testRP.Path {
			props["Property"] = rp.HTTPReq.FormValue("test_key")
		}
		return true
	})

	mockID.EXPECT().ExternalAuthProvider().Return(nil).Times(2)
	mockID.EXPECT().UID().Return("id").Times(2)
	s.mockTelemeter.EXPECT().Track("TestEvent", map[string]any{
		"Property": "test_value",
	}, matchOptions(telemeter.WithUserID(cfg.HashUserAuthID(mockID)), telemeter.WithGroups("TEST", ""))).Times(1)

	cfg.track(testRP)
}

func (s *interceptorTestSuite) TestGrpcRequestInfo() {
	testRP := &RequestParams{
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

	rp := getGRPCRequestDetails(ctx, err, testRP.Path, "request")
	s.Equal(testRP.Path, rp.Path)
	s.Equal(testRP.Code, rp.Code)
	s.Equal(testRP.UserAgent, rp.UserAgent)
	s.Nil(rp.UserID)
	s.Equal("request", rp.GRPCReq)
}

func (s *interceptorTestSuite) TestGrpcWithHTTPRequestInfo() {
	req, _ := http.NewRequest("PATCH", "/wrapped/http", nil)
	rih := requestinfo.NewRequestInfoHandler()
	ctx := peer.NewContext(context.Background(), &peer.Peer{Addr: &net.UnixAddr{Net: "pipe"}})
	md := rih.AnnotateMD(ctx, req)
	md.Set("User-Agent", "test")

	ctx, err := rih.UpdateContextForGRPC(metadata.NewIncomingContext(ctx, md))
	s.NoError(err)

	rp := getGRPCRequestDetails(ctx, err, "ignored grpc method", "request")
	s.Equal(http.StatusOK, rp.Code)
	s.Equal("test", rp.UserAgent)
	s.Nil(rp.UserID)
	s.Equal("request", rp.GRPCReq)
	s.Equal("/wrapped/http", rp.Path)
	s.Equal(http.MethodPatch, rp.Method)
}

type testBody struct {
	N int `json:"n"`
}

type testBodyI interface {
	getTestBody(context.Context, *testBody) (*any, error)
}

func (s *interceptorTestSuite) TestHttpWithBody() {
	body := "{ \"n\": 42 }"
	req, _ := http.NewRequest(http.MethodPost, "/http/body", bytes.NewReader([]byte(body)))
	rp := getHTTPRequestDetails(context.Background(), req, 0)

	rb := GetGRPCRequestBody(testBodyI.getTestBody, rp)
	s.Nil(rb, "body is not captured for HTTP requests")
}

func (s *interceptorTestSuite) TestGrpcWithBody() {
	rp := getGRPCRequestDetails(context.Background(), nil, "/grpc/body", &testBody{N: 42})

	rb := GetGRPCRequestBody(testBodyI.getTestBody, rp)
	if s.NotNil(rb) {
		s.Equal(42, rb.N)
	}

	rp = getGRPCRequestDetails(context.Background(), nil, "/grpc/body", nil)

	rb = GetGRPCRequestBody(testBodyI.getTestBody, rp)
	s.Nil(rb)
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
	rp := getHTTPRequestDetails(ctx, req, 200)
	s.Equal(testRP.Path, rp.Path)
	s.Equal(testRP.Code, rp.Code)
	s.Equal(testRP.UserAgent, rp.UserAgent)
	s.Equal(mockID, rp.UserID)
}
