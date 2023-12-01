package grpc

import (
	"crypto/tls"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/grpc/authz/allow"
	"github.com/stackrox/rox/pkg/grpc/routes"
	"github.com/stackrox/rox/pkg/mtls/verifier"
	"github.com/stretchr/testify/suite"
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
	// TODO: Use TLS mock instead of overriding this with dummy certs

	api1 := NewAPI(defaultConf())
	api2 := NewAPI(defaultConf())

	for i, api := range []API{api1, api2} {
		// Running two tests that start the API results in failure.
		a.Run(fmt.Sprintf("API test %d", i), func() {
			a.Assert().NoError(api.Start().Wait())
			api.Stop()
		})
	}
}

func (a *APIServerSuite) Test_CustomAPI() {
	http.DefaultTransport.(*http.Transport).TLSClientConfig = &tls.Config{InsecureSkipVerify: true}

	a.Run("fetch data from /test", func() {
		cfg, endpointReached := configWithCustomRoute()
		api := NewAPI(cfg)
		a.Assert().NoError(api.Start().Wait())
		defer func() {
			api.Stop()
		}()

		a.requestWithoutErr("https://localhost:8080/test")
		a.waitForSignal(endpointReached)
	})

	a.Run("cannot fetch data from /test after server stopped", func() {
		cfg, endpointReached := configWithCustomRoute()
		api := NewAPI(cfg)
		a.Assert().NoError(api.Start().Wait())
		api.Stop()

		_, err := http.Get("https://localhost:8080/test")
		a.Require().Error(err)
		a.Require().False(endpointReached.IsDone())
	})
}

func (a *APIServerSuite) Test_Stop_CalledMultipleTimes() {
	api := NewAPI(defaultConf())

	a.Assert().NoError(api.Start().Wait())

	a.Assert().True(api.Stop())
	// second call should return false as stop already finished
	a.Assert().False(api.Stop())
}

func (a *APIServerSuite) Test_CantCallStartMultipleTimes() {
	api := NewAPI(defaultConf())
	a.Assert().NoError(api.Start().Wait())
	a.Assert().True(api.Stop())
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
				ListenEndpoint: ":8080",
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
