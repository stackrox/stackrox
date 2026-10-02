package phonehome

import (
	"fmt"
	"net/http"
	"testing"

	idmocks "github.com/stackrox/rox/pkg/grpc/authn/mocks"
	"github.com/stackrox/rox/pkg/telemetry/phonehome/telemeter"
	"github.com/stackrox/rox/pkg/telemetry/phonehome/telemeter/mocks"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
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
		Path:   "/v1.Abc",
		Code:   0,
		UserID: nil,
		GRPCReq: &testRequest{
			value: "test value",
		},
	}
	c := newClientFromConfig(&config{
		clientID:   "test",
		groups:     []telemeter.Option{telemeter.WithGroup("test", "TEST")},
		storageKey: "test-key",
	})
	c.telemeter = s.mockTelemeter
	c.gatherer = &nilGatherer{}

	c.AddInterceptorFuncs("TestEvent", func(rp *RequestParams, props map[string]any) bool {
		if rp.Path == testRP.Path {
			if tr, ok := rp.GRPCReq.(*testRequest); ok {
				props["Property"] = tr.value
			}
		}
		return true
	})

	s.mockTelemeter.EXPECT().Track("TestEvent", map[string]any{
		"Property": "test value",
	}, matchOptions(
		telemeter.WithUserID(c.config.HashUserAuthID(nil)),
		telemeter.WithGroup("test", "TEST"))).Times(1)

	c.GrantConsent()
	defer c.WithdrawConsent()
	c.track(testRP)
}

func (s *interceptorTestSuite) TestAddHttpInterceptor() {
	mockID := idmocks.NewMockIdentity(s.ctrl)
	testRP := &RequestParams{
		Path:   "/v1/abc",
		Code:   200,
		UserID: mockID,
	}
	req, err := http.NewRequest(http.MethodPost, "https://test"+testRP.Path+"?test_key=test_value", nil)
	s.NoError(err)
	testRP.HTTPReq = req
	c := newClientFromConfig(&config{
		clientID:   "test",
		groups:     []telemeter.Option{telemeter.WithGroup("test", "TEST")},
		storageKey: "test-key",
	})
	c.telemeter = s.mockTelemeter
	c.gatherer = &nilGatherer{}
	c.WithdrawConsent()

	c.AddInterceptorFuncs("TestEvent", func(rp *RequestParams, props map[string]any) bool {
		if rp.Path == testRP.Path {
			props["Property"] = rp.HTTPReq.FormValue("test_key")
		}
		return true
	})

	mockID.EXPECT().ExternalAuthProvider().Return(nil).Times(2)
	mockID.EXPECT().UID().Return("id").Times(2)
	s.mockTelemeter.EXPECT().Track("TestEvent", map[string]any{
		"Property": "test_value",
	}, matchOptions(
		telemeter.WithUserID(c.config.HashUserAuthID(mockID)),
		telemeter.WithGroup("test", "TEST"))).Times(1)

	c.GrantConsent()
	defer c.WithdrawConsent()
	c.track(testRP)
}
