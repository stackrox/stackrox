package grpc

import (
	"crypto/tls"
	"fmt"
	"net/http"
	"os"
	"path"
	"runtime"
	"syscall"
	"testing"
	"time"

	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/grpc/authn"
	"github.com/stackrox/rox/pkg/grpc/authz/allow"
	"github.com/stackrox/rox/pkg/grpc/routes"
	"github.com/stackrox/rox/pkg/mtls/verifier"
	"github.com/stackrox/rox/pkg/testutils/envisolator"
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

type testHandler struct {
	received concurrency.Signal
}

func (h *testHandler) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	h.received.Signal()
	_, _ = w.Write([]byte("ASCII response"))
}

func NewHandler(sig concurrency.Signal) *testHandler {
	return &testHandler{
		received: sig,
	}
}

func fromRoot(p string) string {
	_, file, _, _ := runtime.Caller(1)
	d := path.Dir(file)
	return path.Clean(fmt.Sprintf("%s/../../%s", d, p))
}

func TestGrpcServer(t *testing.T) {
	suite.Run(t, new(serverSuite))
}

type serverSuite struct {
	envIsolator    *envisolator.EnvIsolator
	receivedSignal concurrency.Signal
	api            API
	endpoint       string

	suite.Suite
}

var _ suite.SetupTestSuite = &serverSuite{}
var _ suite.TearDownTestSuite = &serverSuite{}

func (s *serverSuite) SetupTest() {
	s.envIsolator = envisolator.NewEnvIsolator(s.T())
	s.envIsolator.Setenv("ROX_MTLS_CERT_FILE", fromRoot("tools/local-sensor/certs/cert.pem"))
	s.envIsolator.Setenv("ROX_MTLS_KEY_FILE", fromRoot("tools/local-sensor/certs/key.pem"))
	s.envIsolator.Setenv("ROX_MTLS_CA_FILE", fromRoot("tools/local-sensor/certs/caCert.pem"))
	s.envIsolator.Setenv("ROX_MTLS_CA_KEY_FILE", fromRoot("tools/local-sensor/certs/caKey.pem"))

	s.receivedSignal = concurrency.NewSignal()
	fakeHandler := NewHandler(s.receivedSignal)

	conf := Config{
		CustomRoutes: []routes.CustomRoute{
			{
				Route:         "/test",
				Authorizer:    allow.Anonymous(),
				ServerHandler: fakeHandler,
				Compression:   false,
			},
		},
		IdentityExtractors: []authn.IdentityExtractor{},
		Endpoints: []*EndpointConfig{
			{
				ListenEndpoint: ":9999",
				TLS:            verifier.NonCA{},
				ServeGRPC:      true,
				ServeHTTP:      true,
			},
		},
	}
	s.api = NewAPI(conf)

	http.DefaultTransport.(*http.Transport).TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
}

func (s *serverSuite) TearDownTest() {
	s.envIsolator.RestoreAll()
	stopped := s.api.Stop()
	stopped.Wait()
}

func (s *serverSuite) TestHTTPsServerReturnsResponse() {
	started := s.api.Start()
	started.Wait()

	_, err := http.Get("https://localhost:9999/test")
	s.NoError(err)
	select {
	case <-s.receivedSignal.Done():
		break
	case <-time.After(2 * time.Second):
		s.Fail("timed-out (2s): should have received HTTPs request")
	}
}

func (s *serverSuite) TestHTTPsStopsCompletelyAfterCallingStop() {
	started := s.api.Start()
	started.Wait()

	_, err := http.Get("https://localhost:9999/test")
	s.NoError(err)

	stopped := s.api.Stop()
	stopped.Wait()

	_, err = http.Get("https://localhost:9999/test")
	s.ErrorIs(err, syscall.ECONNREFUSED)
}
