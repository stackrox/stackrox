package grpc

import (
	"crypto/tls"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"os"
	"testing"

	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/grpc/authz/allow"
	"github.com/stackrox/rox/pkg/grpc/routes"
	"github.com/stackrox/rox/pkg/mtls/verifier"
	"github.com/stackrox/rox/pkg/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

func TestMaxResponseMsgSize_Unset(t *testing.T) {
	require.NoError(t, os.Unsetenv(maxResponseMsgSizeSetting.EnvVar()))

	assert.Equal(t, defaultMaxResponseMsgSize, maxResponseMsgSize())
}

func TestMaxResponseMsgSize_Empty(t *testing.T) {
	require.NoError(t, os.Setenv(maxResponseMsgSizeSetting.EnvVar(), ""))

	assert.Equal(t, defaultMaxResponseMsgSize, maxResponseMsgSize())
}

func TestMaxResponseMsgSize_Invalid(t *testing.T) {
	require.NoError(t, os.Setenv(maxResponseMsgSizeSetting.EnvVar(), "notAnInt"))

	assert.Equal(t, defaultMaxResponseMsgSize, maxResponseMsgSize())
}

func TestMaxResponseMsgSize_Valid(t *testing.T) {
	require.NoError(t, os.Setenv(maxResponseMsgSizeSetting.EnvVar(), "1337"))

	assert.Equal(t, 1337, maxResponseMsgSize())
}

type ServerLifecycleTest struct {
	suite.Suite
	conf                Config
	testEndpointReached concurrency.Signal
	httpClient          *http.Client
}

func Test_ServerLifecycleSuite(t *testing.T) {
	suite.Run(t, new(ServerLifecycleTest))
}

var (
	handlerCount int = 1
)

func (s *ServerLifecycleTest) SetupTest() {
	// TODO: Use TLS mock instead of overriding this with dummy certs
	utils.CrashOnError(os.Setenv("ROX_MTLS_CERT_FILE", "../../tools/local-sensor/certs/cert.pem"))
	utils.CrashOnError(os.Setenv("ROX_MTLS_KEY_FILE", "../../tools/local-sensor/certs/key.pem"))
	utils.CrashOnError(os.Setenv("ROX_MTLS_CA_FILE", "../../tools/local-sensor/certs/caCert.pem"))
	utils.CrashOnError(os.Setenv("ROX_MTLS_CA_KEY_FILE", "../../tools/local-sensor/certs/caKey.pem"))

	s.testEndpointReached = concurrency.NewSignal()
	handler := &testHandler{received: s.testEndpointReached, name: fmt.Sprintf("handler-%d", handlerCount)}
	handlerCount++

	s.conf = Config{
		CustomRoutes: []routes.CustomRoute{
			{
				Route:         "/test",
				Authorizer:    allow.Anonymous(),
				ServerHandler: handler,
			},
		},

		Endpoints: []*EndpointConfig{
			{
				ListenEndpoint: ":8080",
				TLS:            verifier.NonCA{},
				ServeGRPC:      true,
				ServeHTTP:      true,
			},
		},
	}
	http.DefaultTransport.(*http.Transport).TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	s.httpClient = &http.Client{
		Transport: http.DefaultTransport,
	}
}

var _ suite.SetupTestSuite = &ServerLifecycleTest{}

//func (s *ServerLifecycleTest) Test_NewAPI() {
//	api := NewAPI(s.conf)
//
//	started := api.Start()
//	started.Wait()
//	api.Stop().Wait()
//}

func (s *ServerLifecycleTest) Test_NewAPI_TestEndpointAvailable() {
	api := NewAPI(s.conf)
	api.Start().Wait()

	s.Require().False(s.testEndpointReached.IsDone())
	r, _ := s.httpClient.Get("https://localhost:8080/test")
	content, _ := io.ReadAll(r.Body)
	s.T().Logf("## RESPONSE FROM API: %s", content)
	//s.Assert().NoError(err)
	//s.Assert().True(s.testEndpointReached.IsDone())
	api.Stop().Wait()
}

//func (s *ServerLifecycleTest) Test_NewAPI_StartAndStop() {
//	api1 := NewAPI(s.conf)
//	api1.Start().Wait()
//
//	api1.Stop().Wait()
//
//	api2 := NewAPI(s.conf)
//	api2.Start().Wait()
//	api2.Stop().Wait()
//}

func (s *ServerLifecycleTest) Test_NewAPI_TestEndpointNotAvailableAfterStop() {
	api := NewAPI(s.conf)
	api.Start().Wait()
	api.Stop().Wait()

	s.Require().False(s.testEndpointReached.IsDone())
	r, err := s.httpClient.Get("https://localhost:8080/test")
	content, _ := io.ReadAll(r.Body)
	s.T().Logf("## RESPONSE FROM API: %s", content)
	//s.Assert().Equal(404, r.StatusCode)
	//defer r.Body.Close()
	//content, err := io.ReadAll(r.Body)
	s.Assert().Error(err)
	s.Assert().False(s.testEndpointReached.IsDone())
}

type testHandler struct {
	name     string
	received concurrency.Signal
}

func (h *testHandler) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	h.received.Signal()
	n := rand.Intn(10000)
	_, _ = w.Write([]byte(fmt.Sprintf("%s-%d", h.name, n)))
}
