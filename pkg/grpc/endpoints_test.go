package grpc

import (
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"net"
	"net/http"
	"testing"

	"github.com/stackrox/rox/pkg/mtls/verifier"
	"github.com/stackrox/rox/pkg/netutil"
	"github.com/stackrox/rox/pkg/testutils"
	"github.com/stackrox/rox/pkg/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"golang.org/x/net/http2"
	"google.golang.org/grpc"
)

type misdirectedRequestSuite struct {
	tlsConfigurer verifier.TLSConfigurer
	grpcSrv       *grpc.Server
	httpHandler   http.Handler
	suite.Suite
}

type baseCase struct {
	URLHost              string
	ServerName           string
	Expect421IfSupported bool
}

func (c *baseCase) Run(t *testing.T, endpoint net.Addr, serverBaseCfg EndpointConfig, clientUseHTTP2 bool) {
	protocol := "https"
	if serverBaseCfg.TLS == nil {
		protocol = "http"
	}
	url := fmt.Sprintf("%s://%s/", protocol, c.URLHost)

	resp := makeRequestWithSNI(t, endpoint, url, c.ServerName, clientUseHTTP2)
	defer utils.IgnoreError(resp.Body.Close)

	expect421 := c.Expect421IfSupported &&
		serverBaseCfg.DenyMisdirectedRequests &&
		serverBaseCfg.TLS != nil &&
		clientUseHTTP2 &&
		!serverBaseCfg.NoHTTP2

	if expect421 {
		assert.Equal(t, http.StatusMisdirectedRequest, resp.StatusCode, "expected a 421 status code")
		body, err := io.ReadAll(resp.Body)
		require.NoError(t, err, "failed to read HTTP response body")
		bodyStr := string(body)
		urlHostWithoutPort, _, _, err := netutil.ParseEndpoint(c.URLHost)
		require.NoError(t, err, "failed to parse URL host", c.URLHost)
		assert.Contains(t, bodyStr, urlHostWithoutPort, "error response should contain hostname from the request")
		assert.Contains(t, bodyStr, c.ServerName, "error response should contain ServerName")
	} else {
		assert.Equal(t, http.StatusOK, resp.StatusCode, "expected a 200 status code")
	}
}

func (s *misdirectedRequestSuite) SetupSuite() {
	// Dummy, no-op instances
	s.httpHandler = http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {})
	s.grpcSrv = grpc.NewServer()

	cert := testutils.IssueSelfSignedCert(s.T(), "*.example.com", "*.example.com")
	tlsConfig := &tls.Config{
		Certificates: []tls.Certificate{cert},
	}
	s.tlsConfigurer = verifier.TLSConfigurerFunc(func() (*tls.Config, error) {
		return tlsConfig, nil
	})
}

// testWithEndpoint spins up a server using the given EndpointConfig as a basis, listening on an ephemeral port.
// It then executes the `do` function, and makes sure the servers are shut down cleanly.
func (s *misdirectedRequestSuite) testWithEndpoint(name string, baseCfg EndpointConfig, do func(t *testing.T, addr net.Addr)) {
	s.Require().True(baseCfg.ServeHTTP, "HTTP serving must always be enabled")
	s.Require().Empty(baseCfg.ListenEndpoint, "listen endpoint must not be set in base config")

	cfg := baseCfg
	cfg.ListenEndpoint = "127.0.0.1:0"

	addr, servers, err := cfg.instantiate(s.httpHandler, s.grpcSrv)
	s.Require().NoError(err, "instantiating server for endpoint config")

	serverErrs := make(chan error, len(servers))
	activeSrvAndLiss := make([]serverAndListener, 0, len(servers))

	defer func() {
		for _, srvAndLis := range activeSrvAndLiss {
			_ = srvAndLis.listener.Close()
		}
		for range activeSrvAndLiss {
			<-serverErrs
		}
	}()
	for _, srvAndLis := range servers {
		srvAndLis := srvAndLis
		go func() {
			serverErrs <- srvAndLis.srv.Serve(srvAndLis.listener)
		}()
		activeSrvAndLiss = append(activeSrvAndLiss, srvAndLis)
	}

	s.T().Run(name, func(t *testing.T) {
		do(t, addr)
	})

	// Check that no server terminated prematurely
	select {
	case err := <-serverErrs:
		s.NoError(err, "premature termination of server")
	default:
	}
}

func (s *misdirectedRequestSuite) generateBaseServerConfigs() map[string]EndpointConfig {
	baseCfgs := make(map[string]EndpointConfig)
	for _, useGRPC := range []bool{false, true} {
		for _, useHTTP2 := range []bool{false, true} {
			for _, useTLS := range []bool{false, true} {
				for _, denyMisdirectedRequests := range []bool{false, true} {
					baseCfg := EndpointConfig{
						ServeHTTP:               true,
						ServeGRPC:               useGRPC,
						NoHTTP2:                 !useHTTP2,
						DenyMisdirectedRequests: denyMisdirectedRequests,
					}
					if useTLS {
						baseCfg.TLS = s.tlsConfigurer
					}
					baseCfgs[fmt.Sprintf("grpc=%t,http2=%t,tls=%t,denyMisdirected=%t", useGRPC, useHTTP2, useTLS, denyMisdirectedRequests)] = baseCfg
				}
			}
		}
	}
	return baseCfgs
}

func (s *misdirectedRequestSuite) TestAll() {
	baseCases := map[string]baseCase{
		"correct ServerName": {
			URLHost:              "foo.example.com",
			ServerName:           "foo.example.com",
			Expect421IfSupported: false,
		},
		"correct ServerName with port in Host": {
			URLHost:              "foo.example.com:1234",
			ServerName:           "foo.example.com",
			Expect421IfSupported: false,
		},
		"no ServerName": {
			URLHost:              "foo.example.com",
			ServerName:           "",
			Expect421IfSupported: false,
		},
		"no ServerName with port in Host": {
			URLHost:              "foo.example.com:1234",
			ServerName:           "",
			Expect421IfSupported: false,
		},
		"incorrect ServerName": {
			URLHost:              "foo.example.com",
			ServerName:           "bar.example.com",
			Expect421IfSupported: true,
		},
		"incorrect ServerName with port in Host": {
			URLHost:              "foo.example.com:1234",
			ServerName:           "bar.example.com",
			Expect421IfSupported: true,
		},
		"IP, no ServerName": {
			URLHost:              "1.2.3.4",
			ServerName:           "",
			Expect421IfSupported: false,
		},
		"IP:port, no ServerName": {
			URLHost:              "1.2.3.4:1234",
			ServerName:           "",
			Expect421IfSupported: false,
		},
		"IP, correct ServerName": {
			URLHost:              "1.2.3.4",
			ServerName:           "foo.example.com",
			Expect421IfSupported: false,
		},
		"IP:port, correct ServerName": {
			URLHost:              "1.2.3.4:1234",
			ServerName:           "foo.example.com",
			Expect421IfSupported: false,
		},
	}

	for name, serverBaseCfg := range s.generateBaseServerConfigs() {
		s.testWithEndpoint(fmt.Sprintf("server[%s]", name), serverBaseCfg, func(t *testing.T, addr net.Addr) {
			for _, clientUseHTTP2 := range []bool{false, true} {
				t.Run(fmt.Sprintf("client[http2=%t]", clientUseHTTP2), func(t *testing.T) {
					for name, c := range baseCases {
						if serverBaseCfg.TLS == nil && c.ServerName != "" {
							continue // SNI-dependent tests are not applicable for non-TLS setting
						}

						t.Run(name, func(t *testing.T) {
							c.Run(t, addr, serverBaseCfg, clientUseHTTP2)
						})
					}
				})
			}
		})
	}
}

func makeRequestWithSNI(t *testing.T, endpoint net.Addr, targetURL, serverName string, useHTTP2 bool) *http.Response {
	tlsConfig := &tls.Config{
		InsecureSkipVerify: true,
		ServerName:         serverName,
	}
	transport := &http.Transport{
		DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
			return (&net.Dialer{}).DialContext(ctx, endpoint.Network(), endpoint.String())
		},
		DialTLSContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
			return (&tls.Dialer{
				Config: tlsConfig,
			}).DialContext(ctx, endpoint.Network(), endpoint.String())
		},
		TLSClientConfig:   tlsConfig,
		ForceAttemptHTTP2: useHTTP2, // necessary because we have a custom DialTLSContext
	}

	if useHTTP2 {
		require.NoError(t, http2.ConfigureTransport(transport))
	}

	client := &http.Client{
		Transport: transport,
	}

	req, err := http.NewRequest(http.MethodGet, targetURL, nil)
	require.NoError(t, err, "creating HTTP request")

	if useHTTP2 {
		req.Proto = "HTTP/2.0"
		req.ProtoMajor, req.ProtoMinor = 2, 0
	}

	resp, err := client.Do(req)
	require.NoError(t, err, "unexpected HTTP transport error")

	return resp
}

func TestMisdirectedRequests(t *testing.T) {
	suite.Run(t, new(misdirectedRequestSuite))
}
