package grpc

import (
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/pkg/errors"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/grpc/authz/allow"
	"github.com/stackrox/rox/pkg/grpc/routes"
	"github.com/stackrox/rox/pkg/mtls/verifier"
	"github.com/stackrox/rox/pkg/testutils"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"google.golang.org/grpc"
)

type APIServerSuite struct {
	suite.Suite
}

func (a *APIServerSuite) SetupTest() {
	// In order to start the gRPC server, we need to have the MTLS environment variables
	// pointing to some valid certificate/key pair. In this case we are using the ones
	// created for local-sensor, which are dummy self-signed certificates.
	a.T().Setenv("ROX_MTLS_CERT_FILE", "../../tools/local-sensor/certs/cert.pem")
	a.T().Setenv("ROX_MTLS_KEY_FILE", "../../tools/local-sensor/certs/key.pem")
	a.T().Setenv("ROX_MTLS_CA_FILE", "../../tools/local-sensor/certs/caCert.pem")
	a.T().Setenv("ROX_MTLS_CA_KEY_FILE", "../../tools/local-sensor/certs/caKey.pem")

	setUpPrintSocketInfoFunction(a.T())
}

func Test_APIServerSuite(t *testing.T) {
	suite.Run(t, new(APIServerSuite))
}

var _ suite.SetupTestSuite = &APIServerSuite{}

func (a *APIServerSuite) TestEnvValues() {
	cases := map[string]int{
		"":         defaultMaxResponseMsgSize,
		"notAnInt": defaultMaxResponseMsgSize,
		"1337":     1337,
	}

	for envValue, expected := range cases {
		a.Run(fmt.Sprintf("%s=%d", envValue, expected), func() {
			a.T().Setenv(maxResponseMsgSizeSetting.EnvVar(), envValue)
			a.Assert().Equal(expected, maxResponseMsgSize())
		})
	}
}

func waitForPortToBeFree(t *testing.T, port uint64) {
	ticker := time.NewTicker(10 * time.Millisecond)
	defer ticker.Stop()

	timer := time.NewTimer(10 * time.Second)
	defer timer.Stop()

	for {
		select {
		case <-ticker.C:
			listener, err := net.Listen("tcp4", fmt.Sprintf(":%d", port))
			if err != nil {
				t.Logf("Port %d still in use on tcp4", port)
				continue
			}
			if errClose := listener.Close(); errClose != nil {
				t.Logf("Closing tcp4 listener on port %d failed: %v", port, errClose)
			}

			listener, err = net.Listen("tcp6", fmt.Sprintf(":%d", port))
			if err != nil {
				t.Logf("Port %d still in use on tcp6", port)
				continue
			}
			if errClose := listener.Close(); errClose != nil {
				t.Logf("Closing tcp6 listener on port %d failed: %v", port, errClose)
			}

			return

		case <-timer.C:
			t.Logf("Timed out waiting for free port on: %d", port)
		}
	}
}

func (a *APIServerSuite) Test_TwoTestsStartingAPIs() {
	testPort := testutils.GetFreeTestPort()
	api1 := newAPIForTest(a.T(), defaultConf(testPort))
	api2 := newAPIForTest(a.T(), defaultConf(testPort))

	for i, api := range []API{api1, api2} {
		// Running two tests that start the API results in failure.
		a.Run(fmt.Sprintf("API test %d", i), func() {
			waitForPortToBeFree(a.T(), testPort)
			a.Assert().NoError(api.Start().Wait())
			a.Require().True(api.Stop())
		})
	}
}

func (a *APIServerSuite) Test_CustomAPI() {
	http.DefaultTransport.(*http.Transport).TLSClientConfig = &tls.Config{InsecureSkipVerify: true}

	a.Run("fetch data from /test", func() {
		testPort := testutils.GetFreeTestPort()
		cfg, endpointReached := configWithCustomRoute(testPort)
		api := newAPIForTest(a.T(), cfg)
		a.T().Cleanup(func() { api.Stop() })
		a.Assert().NoError(api.Start().Wait())

		a.requestWithoutErr(fmt.Sprintf("https://localhost:%d/test", testPort))
		a.waitForSignal(endpointReached)
	})

	a.Run("cannot fetch data from /test after server stopped", func() {
		testPort := testutils.GetFreeTestPort()
		cfg, endpointReached := configWithCustomRoute(testPort)
		api := newAPIForTest(a.T(), cfg)
		a.Assert().NoError(api.Start().Wait())
		a.Require().True(api.Stop())

		resp, err := http.Get(fmt.Sprintf("https://localhost:%d/test", testPort))
		defer testutils.SafeClientClose(resp)
		a.Require().Error(err)
		a.Require().False(endpointReached.IsDone())
	})
}

func (a *APIServerSuite) Test_Stop_CalledMultipleTimes() {
	api := newAPIForTest(a.T(), defaultConf(testutils.GetFreeTestPort()))

	a.Assert().NoError(api.Start().Wait())

	a.Require().True(api.Stop())
	// second call should return false as stop already finished
	a.Require().False(api.Stop())
}

func (a *APIServerSuite) Test_CantCallStartMultipleTimes() {
	api := newAPIForTest(a.T(), defaultConf(testutils.GetFreeTestPort()))
	a.Assert().NoError(api.Start().Wait())
	a.Require().True(api.Stop())
	a.Assert().Error(api.Start().Wait())
}

func (a *APIServerSuite) requestWithoutErr(url string) {
	resp, err := http.Get(url)
	defer testutils.SafeClientClose(resp)
	a.Require().NoError(err)
}

func (a *APIServerSuite) waitForSignal(s *concurrency.Signal) {
	select {
	case <-s.Done():
		break
	case <-time.After(2 * time.Second):
		a.FailNow("Should have received request on endpoint")
	}
}

func configWithCustomRoute(port uint64) (Config, *concurrency.Signal) {
	endpointReached := concurrency.NewSignal()
	cfg := defaultConf(port)
	handler := &testHandler{received: &endpointReached}
	cfg.CustomRoutes = []routes.CustomRoute{
		{
			Route:         "/test",
			Authorizer:    allow.Anonymous(),
			ServerHandler: handler,
		},
	}
	return cfg, &endpointReached
}

func defaultConf(port uint64) Config {
	return Config{
		Endpoints: []*EndpointConfig{
			{
				ListenEndpoint: fmt.Sprintf(":%d", port),
				TLS:            verifier.NonCA{},
				ServeGRPC:      true,
				ServeHTTP:      true,
			},
		},
	}
}

type testHandler struct {
	received *concurrency.Signal
}

func (h *testHandler) ServeHTTP(w http.ResponseWriter, _ *http.Request) {
	h.received.Signal()
	_, _ = w.Write([]byte("Hello!"))
}

// Testing server error response from gRPC Gateway.
type pingServiceTestErrorImpl struct {
	v1.UnimplementedPingServiceServer
}

func (s *pingServiceTestErrorImpl) RegisterServiceServer(grpcServer *grpc.Server) {
	v1.RegisterPingServiceServer(grpcServer, s)
}

func (s *pingServiceTestErrorImpl) RegisterServiceHandler(ctx context.Context, mux *runtime.ServeMux, conn *grpc.ClientConn) error {
	return v1.RegisterPingServiceHandler(ctx, mux, conn)
}

func (s *pingServiceTestErrorImpl) AuthFuncOverride(ctx context.Context, fullMethodName string) (context.Context, error) {
	return ctx, allow.Anonymous().Authorized(ctx, fullMethodName)
}

func (s *pingServiceTestErrorImpl) Ping(context.Context, *v1.Empty) (*v1.PongMessage, error) {
	return nil, errors.Wrap(errox.InvalidArgs, "missing argument")
}

func (a *APIServerSuite) Test_GRPC_Server_Error_Response() {
	testPort := testutils.GetFreeTestPort()
	url := fmt.Sprintf("https://localhost:%d/v1/ping", testPort)
	jsonPayload := `{"code":3, "details":[], "error":"missing argument: invalid arguments", "message":"missing argument: invalid arguments"}`

	http.DefaultTransport.(*http.Transport).TLSClientConfig = &tls.Config{InsecureSkipVerify: true}

	api := newAPIForTest(a.T(), defaultConf(testPort))
	grpcServiceHandler := &pingServiceTestErrorImpl{}
	api.Register(grpcServiceHandler)
	a.Assert().NoError(api.Start().Wait())
	a.T().Cleanup(func() { api.Stop() })

	resp, err := http.Get(url)
	defer testutils.SafeClientClose(resp)
	a.Require().NoError(err)

	body, err := io.ReadAll(resp.Body)
	a.Require().NoError(err)

	bodyStr := string(body)
	a.Assert().JSONEq(jsonPayload, bodyStr)
}

func newAPIForTest(t *testing.T, config Config) API {
	api := NewAPI(config)
	impl, ok := api.(*apiImpl)
	require.True(t, ok)
	impl.debugLog = newDebugLogger(t)
	return api
}

func setUpPrintSocketInfoFunction(t *testing.T) {
	printSocketInfo = func(_ *testing.T) {
		if r := recover(); r != nil {
			if err, ok := r.(string); ok {
				if strings.Contains(err, syscall.EADDRINUSE.Error()) {
					t.Log("-----------------------------------------------")
					t.Log(" STACK TRACE INFO")
					t.Log("-----------------------------------------------")
					if printErr := testPrintStackTraceInfo(t); printErr != nil {
						t.Log(printErr)
					}
					t.Log("-----------------------------------------------")
					t.Log(" SOCKET INFO")
					t.Log("-----------------------------------------------")
					if printErr := testPrintSocketInfo(t, testutils.GetUsedPortsList()...); printErr != nil {
						t.Log(printErr)
					}
					t.Log("-----------------------------------------------")
					panic(err)
				}
			}
		}
	}
}
