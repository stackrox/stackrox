package grpc

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/grpc/authz/allow"
	"github.com/stackrox/rox/pkg/grpc/ratelimit"
	"github.com/stackrox/rox/pkg/grpc/routes"
	"github.com/stackrox/rox/pkg/mtls/verifier"
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

func (a *APIServerSuite) Test_Server_RateLimit_HTTP_Integration() {
	http.DefaultTransport.(*http.Transport).TLSClientConfig = &tls.Config{InsecureSkipVerify: true}

	a.Run("no limiter", func() {
		cfg, endpointReached := configWithCustomRoute()

		api := NewAPI(cfg)
		a.Assert().NoError(api.Start().Wait())
		defer func() {
			api.Stop()
		}()

		for i := 0; i < 50; i++ {
			resp, err := http.Get("https://localhost:8080/test")

			a.Require().NoError(err)
			a.Require().Equal(http.StatusOK, resp.StatusCode)
		}
		a.waitForSignal(endpointReached)
	})

	a.Run("default unlimited", func() {
		cfg, endpointReached := configWithCustomRoute()
		cfg.RateLimiter = ratelimit.NewRateLimiter()

		api := NewAPI(cfg)
		a.Assert().NoError(api.Start().Wait())
		defer func() {
			api.Stop()
		}()

		for i := 0; i < 50; i++ {
			resp, err := http.Get("https://localhost:8080/test")

			a.Require().NoError(err)
			a.Require().Equal(http.StatusOK, resp.StatusCode)
		}
		a.waitForSignal(endpointReached)
	})

	a.Run("hit rate limit", func() {
		a.T().Setenv(env.CentralApiRateLimitPerSecond.EnvVar(), "10")

		cfg, endpointReached := configWithCustomRoute()
		cfg.RateLimiter = ratelimit.NewRateLimiter()

		api := NewAPI(cfg)
		a.Assert().NoError(api.Start().Wait())
		defer func() {
			api.Stop()
		}()

		hitLimit := false
		for i := 0; i < 30; i++ {
			resp, err := http.Get("https://localhost:8080/test")
			a.Require().NoError(err)

			if i < 10 {
				a.Require().Equal(http.StatusOK, resp.StatusCode)
				continue
			}

			if resp.StatusCode != 200 {
				a.Require().Equal(http.StatusTooManyRequests, resp.StatusCode)
				hitLimit = true
				break
			}
		}
		a.Assert().True(hitLimit)

		// Wait for rate limit to refill.
		time.Sleep(2 * time.Second)

		resp, err := http.Get("https://localhost:8080/test")
		a.Assert().NoError(err)
		a.Assert().Equal(http.StatusOK, resp.StatusCode)

		a.waitForSignal(endpointReached)
	})
}

// The pingServiceTestImpl is employed for the purpose of testing GRPC API invocations.
// It is an implementation of the Ping service.
type pingServiceTestImpl struct {
	v1.UnimplementedPingServiceServer

	RequestCount int
}

func (s *pingServiceTestImpl) RegisterServiceServer(grpcServer *grpc.Server) {
	v1.RegisterPingServiceServer(grpcServer, s)
}

func (s *pingServiceTestImpl) RegisterServiceHandler(ctx context.Context, mux *runtime.ServeMux, conn *grpc.ClientConn) error {
	return v1.RegisterPingServiceHandler(ctx, mux, conn)
}

func (s *pingServiceTestImpl) AuthFuncOverride(ctx context.Context, fullMethodName string) (context.Context, error) {
	return ctx, allow.Anonymous().Authorized(ctx, fullMethodName)
}

func (s *pingServiceTestImpl) Ping(context.Context, *v1.Empty) (*v1.PongMessage, error) {
	s.RequestCount += 1

	result := &v1.PongMessage{
		Status: "test",
	}

	return result, nil
}

func (a *APIServerSuite) Test_Server_RateLimit_GRPC_Integration() {
	http.DefaultTransport.(*http.Transport).TLSClientConfig = &tls.Config{InsecureSkipVerify: true}

	a.Run("no limiter", func() {
		cfg := defaultConf()

		api := NewAPI(cfg)
		pingService := &pingServiceTestImpl{}
		api.Register(pingService)
		a.Assert().NoError(api.Start().Wait())
		defer func() { api.Stop() }()

		numOfRequests := 50
		for i := 0; i < numOfRequests; i++ {
			resp, err := http.Get("https://localhost:8080/v1/ping")
			a.Require().NoError(err)
			a.Require().Equal(http.StatusOK, resp.StatusCode)
		}
		a.Assert().Equal(numOfRequests, pingService.RequestCount)
	})

	a.Run("default unlimited", func() {
		cfg := defaultConf()
		cfg.RateLimiter = ratelimit.NewRateLimiter()

		api := NewAPI(cfg)
		pingService := &pingServiceTestImpl{}
		api.Register(pingService)
		a.Assert().NoError(api.Start().Wait())
		defer func() { api.Stop() }()

		numOfRequests := 50
		for i := 0; i < numOfRequests; i++ {
			resp, err := http.Get("https://localhost:8080/v1/ping")
			a.Require().NoError(err)
			a.Require().Equal(http.StatusOK, resp.StatusCode)
		}
		a.Assert().Equal(numOfRequests, pingService.RequestCount)
	})

	a.Run("hit rate limit", func() {
		a.T().Setenv(env.CentralApiRateLimitPerSecond.EnvVar(), "10")

		cfg := defaultConf()
		cfg.RateLimiter = ratelimit.NewRateLimiter()

		api := NewAPI(cfg)
		pingService := &pingServiceTestImpl{}
		api.Register(pingService)
		a.Assert().NoError(api.Start().Wait())
		defer func() { api.Stop() }()

		hitLimit := false
		requestCount := 0
		for requestCount < 30 {
			requestCount += 1
			resp, err := http.Get("https://localhost:8080/v1/ping")
			a.Require().NoError(err)

			if requestCount <= 10 {
				a.Require().Equal(http.StatusOK, resp.StatusCode)
				continue
			}

			if resp.StatusCode != 200 {
				a.Require().Equal(http.StatusTooManyRequests, resp.StatusCode)
				hitLimit = true

				// Request was rejected because of rate limit.
				requestCount -= 1
				break
			}
		}
		a.Assert().True(hitLimit)
		a.Assert().Equal(requestCount, pingService.RequestCount)

		// Wait for rate limit to refill.
		time.Sleep(2 * time.Second)

		requestCount += 1
		resp, err := http.Get("https://localhost:8080/v1/ping")
		a.Assert().NoError(err)
		a.Assert().Equal(http.StatusOK, resp.StatusCode)

		a.Assert().Equal(requestCount, pingService.RequestCount)
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
	name     string
	received *concurrency.Signal
}

func (h *testHandler) ServeHTTP(w http.ResponseWriter, _ *http.Request) {
	h.received.Signal()
	_, _ = w.Write([]byte("Hello!"))
}
