package grpc

import (
	"context"
	"crypto/tls"
	"net/http"
	"time"

	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/grpc/authz/allow"
	"github.com/stackrox/rox/pkg/grpc/ratelimit"
	"github.com/stackrox/rox/pkg/grpc/routes"
	"google.golang.org/grpc"
)

type testHTTPHandler struct {
	requestCount int
}

func (h *testHTTPHandler) ServeHTTP(w http.ResponseWriter, _ *http.Request) {
	h.requestCount++

	_, _ = w.Write([]byte("Hello!"))
}

// The pingServiceTestImpl is employed for the purpose of testing GRPC API invocations.
// It is an implementation of the Ping service.
type pingServiceTestImpl struct {
	v1.UnimplementedPingServiceServer

	requestCount int
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
	s.requestCount++

	result := &v1.PongMessage{
		Status: "test",
	}

	return result, nil
}

func (a *APIServerSuite) Test_Server_RateLimit_HTTP_Integration() {
	http.DefaultTransport.(*http.Transport).TLSClientConfig = &tls.Config{InsecureSkipVerify: true}

	tests := []struct {
		name       string
		hasLimiter bool
		maxPerSec  int
		useHTTP    bool
		useGRPC    bool
	}{
		{"no limiter http only", false, 0, true, false},
		{"default unlimited http only", true, 0, true, false},
		{"hit rate limit http only", true, 2, true, false},
		{"no limiter grpc only", false, 0, false, true},
		{"default unlimited grpc only", true, 0, false, true},
		{"hit rate limit grpc only", true, 2, false, true},
		{"no limiter http and grpc", false, 0, true, true},
		{"default unlimited http and grpc", true, 0, true, true},
		{"hit rate limit http and grpc", true, 2, true, true},
	}

	for _, tt := range tests {
		a.Run(tt.name, func() {
			cfg := defaultConf()
			if tt.hasLimiter {
				cfg.RateLimiter = ratelimit.NewRateLimiter(tt.maxPerSec)
			}

			httpHandler := &testHTTPHandler{}
			cfg.CustomRoutes = []routes.CustomRoute{
				{
					Route:         "/test",
					Authorizer:    allow.Anonymous(),
					ServerHandler: httpHandler,
				},
			}

			api := NewAPI(cfg)
			grpcService := &pingServiceTestImpl{}
			api.Register(grpcService)
			a.Assert().NoError(api.Start().Wait())
			defer func() { api.Stop() }()

			var urls []string
			if tt.useHTTP {
				urls = append(urls, "https://localhost:8080/test")
			}
			if tt.useGRPC {
				urls = append(urls, "https://localhost:8080/v1/ping")
			}

			hitLimit := false
			requestCount := 0
			numOfRequests := 50
			for requestCount < numOfRequests {
				for _, url := range urls {
					requestCount++
					resp, err := http.Get(url)
					a.Require().NoError(err)

					if !tt.hasLimiter || tt.maxPerSec == 0 || requestCount <= tt.maxPerSec {
						a.Require().Equal(http.StatusOK, resp.StatusCode)
						continue
					}

					if resp.StatusCode != 200 {
						a.Require().Equal(http.StatusTooManyRequests, resp.StatusCode)
						hitLimit = true

						// Request was rejected because of rate limit.
						requestCount--
						break
					}
				}

				if hitLimit {
					break
				}
			}
			a.Assert().Equal(requestCount, httpHandler.requestCount+grpcService.requestCount)

			if tt.useHTTP {
				a.Assert().Greater(httpHandler.requestCount, 0)
			}
			if tt.useGRPC {
				a.Assert().Greater(grpcService.requestCount, 0)
			}

			if tt.hasLimiter && tt.maxPerSec > 0 {
				a.Assert().True(hitLimit)

				// Wait for rate limit to refill.
				time.Sleep(time.Second)
			}

			for _, url := range urls {
				requestCount++
				resp, err := http.Get(url)
				a.Assert().NoError(err)
				a.Assert().Equal(http.StatusOK, resp.StatusCode)
			}

			a.Assert().Equal(requestCount, httpHandler.requestCount+grpcService.requestCount)
		})
	}
}
