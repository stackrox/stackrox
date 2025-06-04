package clientconn

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path"
	"strings"
	"testing"

	v1API "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/httputil"
	"github.com/stackrox/rox/pkg/mtls"
	"github.com/stackrox/rox/pkg/mtls/verifier"
	"github.com/stackrox/rox/pkg/testutils"
	"github.com/stretchr/testify/suite"
)

const centralEndpoint = "central.stackrox:443"

func TestClient(t *testing.T) {
	suite.Run(t, new(ClientTestSuite))
}

type ClientTestSuite struct {
	suite.Suite
}

func (t *ClientTestSuite) SetupTest() {
	wd, _ := os.Getwd()
	testdata := path.Join(wd, "testdata")

	t.T().Setenv("ROX_MTLS_CA_FILE", path.Join(testdata, "ca.pem"))
}

func (t *ClientTestSuite) TestAddRootCA() {
	const certCount = 2
	cert := &x509.Certificate{Raw: []byte(`cert data`), SubjectKeyId: []byte(`SubjectKeyId1`), RawSubject: []byte(`RawSubject1`)}
	cert2 := &x509.Certificate{Raw: []byte(`cert data2`), SubjectKeyId: []byte(`SubjectKeyId2`), RawSubject: []byte(`RawSubject2`)}

	opts, err := OptionsForEndpoint(centralEndpoint, AddRootCAs(cert, cert2))
	t.Require().NoError(err)

	// read system root CAs
	sysCertPool, err := verifier.SystemCertPool()
	t.Require().NoError(err)

	addedCertsCount := len(opts.TLS.RootCAs.Subjects()) - len(sysCertPool.Subjects())
	t.Equalf(addedCertsCount, certCount, "Expected %d certificates being added", certCount)
}

func (t *ClientTestSuite) TestRootCA_WithNilCA_ShouldPanic() {
	t.Panics(func() {
		_, _ = OptionsForEndpoint(centralEndpoint, AddRootCAs(nil))
	})
}

func (t *ClientTestSuite) TestAuthenticatedHTTPTransport_WebSocket() {
	noopServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	baseTransport := httputil.DefaultTransport()

	testcases := []struct {
		name   string
		scheme string
		valid  bool
	}{
		{
			name:   "valid wss",
			scheme: "wss",
			valid:  true,
		},
		{
			name:   "invalid wss",
			scheme: "wss",
			valid:  false,
		},
		{
			name:   "valid ws",
			scheme: "ws",
			valid:  true,
		},
		{
			name:   "invalid ws",
			scheme: "ws",
			valid:  false,
		},
	}
	for _, testcase := range testcases {
		t.Run(testcase.name, func() {
			// Ensure the request's URL drops the WebSocket.
			baseTransport.Proxy = func(r *http.Request) (*url.URL, error) {
				if !testcase.valid {
					t.FailNow("Should not make it this far")
				}

				// http because TLS is disabled.
				t.Equal("http://central.stackrox.svc:443/hello/howdy?file=rhelv2%2Frepository-to-cpe.json&uuid=f81dbc6b-5899-433b-bc86-9127219a9d89", r.URL.String())

				// Forward traffic to the NO-OP Server
				return url.Parse(noopServer.URL)
			}

			host := testcase.scheme + "://central.stackrox.svc:443"
			// This is sorted by key.
			rawQuery := url.Values{
				"uuid": []string{"f81dbc6b-5899-433b-bc86-9127219a9d89"},
				"file": []string{"rhelv2/repository-to-cpe.json"},
			}.Encode()
			endpoint := (&url.URL{Path: "/hello/howdy", RawQuery: rawQuery}).String()
			if !testcase.valid {
				endpoint = (&url.URL{
					Scheme:   "https",
					Host:     host,
					Path:     "hello/howdy",
					RawQuery: rawQuery,
				}).String()
			}

			req, err := http.NewRequest(http.MethodGet, endpoint, nil)
			if testcase.valid {
				t.NoError(err)
			} else {
				errEndpoint := `"https://` + testcase.scheme + `:%2F%2Fcentral.stackrox.svc:443/hello/howdy?file=rhelv2%2Frepository-to-cpe.json&uuid=f81dbc6b-5899-433b-bc86-9127219a9d89"`
				errString := `parse ` + errEndpoint + `: invalid URL escape "%2F"`
				t.EqualError(err, errString)
				return
			}

			transport, err := AuthenticatedHTTPTransport(host, mtls.CentralSubject, baseTransport, UseInsecureNoTLS(true))
			t.Require().NoError(err)
			client := &http.Client{
				Transport: transport,
				Timeout:   0,
			}

			resp, err := client.Do(req)
			t.NoError(err)
			t.Equal(http.StatusOK, resp.StatusCode)
		})
	}
}

func (t *ClientTestSuite) TestGRPCConnection() {
	for name, tc := range map[string]struct {
		nextProtos        []string
		skipWhenEnv       map[string]string
		requiredEnv       map[string]string
		expectsError      bool
		expectedErrorText string
	}{
		"h2 as next proto allows GRPCConnection to connect": {
			nextProtos:   []string{"h2"},
			expectsError: false,
		},
		"no (ALPN) next proto and no GRPC env var set (behaviour: enforce prevents connection)": {
			nextProtos: []string{},
			skipWhenEnv: map[string]string{
				"GRPC_ENFORCE_ALPN_ENABLED": "false",
			},
			expectsError:      true,
			expectedErrorText: "missing selected ALPN property",
		},
		"no (ALPN) next proto and GRPC env set to enforce prevents connection": {
			nextProtos: []string{},
			requiredEnv: map[string]string{
				"GRPC_ENFORCE_ALPN_ENABLED": "true",
			},
			expectsError:      true,
			expectedErrorText: "missing selected ALPN property",
		},
		"no (ALPN) next proto and GRPC env set to not enforce allows connection": {
			nextProtos: []string{},
			requiredEnv: map[string]string{
				"GRPC_ENFORCE_ALPN_ENABLED": "false",
			},
		},
	} {
		t.Run(name, func() {
			envAllows, skipReason := checkEnvSkipConditions(t.T(), name, tc.skipWhenEnv, tc.requiredEnv)
			if !envAllows {
				t.T().Skip(skipReason)
			}
			cert := testutils.IssueSelfSignedCert(t.T(), "localhost", "localhost")
			handler := &dummyHandler{}
			tlsSvr := httptest.NewUnstartedServer(http.HandlerFunc(handler.ServeHTTP))
			tlsSvr.TLS = &tls.Config{
				Certificates: []tls.Certificate{cert},
				NextProtos:   tc.nextProtos,
			}
			tlsSvr.EnableHTTP2 = true
			tlsSvr.StartTLS()
			defer closeTLSServer(tlsSvr)
			endpoint := tlsSvr.URL
			dialCtx := context.Background()
			server := mtls.CentralSubject
			connectOptions := Options{TLS: TLSConfigOptions{
				InsecureSkipVerify: true,
				GRPCOnly:           true,
			}}
			endpoint = strings.TrimPrefix(endpoint, "https://")
			cnx, err := GRPCConnection(dialCtx, server, endpoint, connectOptions)
			t.NoError(err)

			client := v1API.NewPingServiceClient(cnx)
			rsp, err := client.Ping(context.Background(), &v1API.Empty{})
			if tc.expectsError {
				t.ErrorContains(err, tc.expectedErrorText)
			} else {
				log.Info(err)
				log.Info(rsp)
			}
		})
	}
}

func checkEnvSkipConditions(
	_ *testing.T,
	caseName string,
	rejectingEnv map[string]string,
	requiredEnv map[string]string,
) (bool, string) {
	envAllows := true
	var skipReason strings.Builder
	skipReason.WriteString("Test \"TestGRPCConnection/")
	skipReason.WriteString(caseName)
	skipReason.WriteString("\" skipped because of env values")
	for varName, value := range requiredEnv {
		actualValue := os.Getenv(varName)
		if actualValue != value {
			skipReason.WriteString(fmt.Sprintf(" %q ", varName))
			skipReason.WriteString(fmt.Sprintf("(got %q but expected %q)", actualValue, value))
			envAllows = false
		}
	}
	for varName, value := range rejectingEnv {
		actualValue := os.Getenv(varName)
		if actualValue == value {
			skipReason.WriteString(fmt.Sprintf(" %q ", varName))
			skipReason.WriteString(fmt.Sprintf("(got value %q requiring skip)", actualValue))
			envAllows = false
		}
	}
	return envAllows, skipReason.String()
}

type dummyHandler struct {
	called bool
}

func (h *dummyHandler) ServeHTTP(w http.ResponseWriter, _ *http.Request) {
	h.called = true
	w.WriteHeader(http.StatusOK)
}

func closeTLSServer(server *httptest.Server) {
	server.CloseClientConnections()
	server.Close()
}
