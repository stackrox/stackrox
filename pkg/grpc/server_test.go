package grpc

import (
	"context"
	"crypto/tls"
	"fmt"
	"io"
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
	"github.com/stretchr/testify/suite"
	"google.golang.org/grpc"
)

const (
	testDefaultPort = 8080
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

	setUpPrintSocketInfoFunction(a.T(), testDefaultPort)
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

func (a *APIServerSuite) Test_TwoTestsStartingAPIs() {
	api1 := NewAPI(defaultConf())
	api2 := NewAPI(defaultConf())

	for i, api := range []API{api1, api2} {
		// Running two tests that start the API results in failure.
		a.Run(fmt.Sprintf("API test %d", i), func() {
			a.T().Cleanup(func() { api.Stop() })
			a.Assert().NoError(api.Start().Wait())
		})
	}
}

func (a *APIServerSuite) Test_CustomAPI() {
	http.DefaultTransport.(*http.Transport).TLSClientConfig = &tls.Config{InsecureSkipVerify: true}

	a.Run("fetch data from /test", func() {
		cfg, endpointReached := configWithCustomRoute()
		api := NewAPI(cfg)
		a.T().Cleanup(func() { api.Stop() })
		a.Assert().NoError(api.Start().Wait())

		a.requestWithoutErr(fmt.Sprintf("https://localhost:%d/test", testDefaultPort))
		a.waitForSignal(endpointReached)
	})

	a.Run("cannot fetch data from /test after server stopped", func() {
		cfg, endpointReached := configWithCustomRoute()
		api := NewAPI(cfg)
		a.Assert().NoError(api.Start().Wait())
		a.Require().True(api.Stop())

		_, err := http.Get(fmt.Sprintf("https://localhost:%d/test", testDefaultPort))
		a.Require().Error(err)
		a.Require().False(endpointReached.IsDone())
	})
}

func (a *APIServerSuite) Test_Stop_CalledMultipleTimes() {
	api := NewAPI(defaultConf())

	a.Assert().NoError(api.Start().Wait())

	a.Require().True(api.Stop())
	// second call should return false as stop already finished
	a.Require().False(api.Stop())
}

func (a *APIServerSuite) Test_CantCallStartMultipleTimes() {
	api := NewAPI(defaultConf())
	a.Assert().NoError(api.Start().Wait())
	a.Require().True(api.Stop())
	a.Assert().Error(api.Start().Wait())
}

func (a *APIServerSuite) requestWithoutErr(url string) {
	_, err := http.Get(url)
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

func configWithCustomRoute() (Config, *concurrency.Signal) {
	endpointReached := concurrency.NewSignal()
	cfg := defaultConf()
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

func defaultConf() Config {
	return Config{
		Endpoints: []*EndpointConfig{
			{
				ListenEndpoint: fmt.Sprintf(":%d", testDefaultPort),
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
	url := fmt.Sprintf("https://localhost:%d/v1/ping", testDefaultPort)
	jsonPayload := `{"code":3, "details":[], "error":"missing argument: invalid arguments", "message":"missing argument: invalid arguments"}`

	http.DefaultTransport.(*http.Transport).TLSClientConfig = &tls.Config{InsecureSkipVerify: true}

	api := NewAPI(defaultConf())
	grpcServiceHandler := &pingServiceTestErrorImpl{}
	api.Register(grpcServiceHandler)
	a.Assert().NoError(api.Start().Wait())
	a.T().Cleanup(func() { api.Stop() })

	resp, err := http.Get(url)
	a.Require().NoError(err)

	body, err := io.ReadAll(resp.Body)
	a.Require().NoError(err)

	bodyStr := string(body)
	a.Assert().JSONEq(jsonPayload, bodyStr)
}

func setUpPrintSocketInfoFunction(t *testing.T, ports ...uint64) {
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
					if printErr := testPrintSocketInfo(t, ports...); printErr != nil {
						t.Log(printErr)
					}
					t.Log("-----------------------------------------------")
					panic(err)
				}
			}
		}
	}
}
