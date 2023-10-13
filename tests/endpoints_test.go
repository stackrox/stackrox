package tests

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/cloudflare/cfssl/helpers"
	"github.com/golang/protobuf/jsonpb"
	"github.com/pkg/errors"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/set"
	"github.com/stackrox/rox/pkg/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/net/http2"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
)

type authMode int

const (
	timeout = 30 * time.Second

	portOffset = 10000

	userPKIProviderName = "test-userpki"
)

const (
	noAuth authMode = iota
	serviceAuth
	userAuth
)

var (
	dialer = net.Dialer{
		Timeout: timeout,
	}
)

func endpointForTargetPort(targetPort uint16) string {
	endpointHostname := os.Getenv("API_HOSTNAME")
	if endpointHostname == "" || endpointHostname == "localhost" {
		panic(errors.Errorf("API_HOSTNAME=%q env variable is not set correctly", endpointHostname))
	}
	return fmt.Sprintf("%s:%d", endpointHostname, targetPort)
}

type endpointsTestCase struct {
	targetPort uint16

	skipTLS          bool
	clientCert       *tls.Certificate
	validServerNames []string

	expectConnectFailure bool
	expectGRPCSuccess    bool
	expectHTTPSuccess    bool
	expectAuth           authMode
}

type endpointsTestContext struct {
	allServerNames []string
	certPool       *x509.CertPool
}

func (c *endpointsTestContext) tlsConfig(clientCert *tls.Certificate, serverName string, useSNI bool) *tls.Config {
	tlsConf := &tls.Config{
		RootCAs: c.certPool,
	}
	if clientCert != nil {
		tlsConf.Certificates = []tls.Certificate{*clientCert}
	}

	if useSNI {
		tlsConf.ServerName = serverName
	} else {
		// To validate against a server name *without* SNI, we need to write our custom verification function.
		tlsConf.InsecureSkipVerify = true
		tlsConf.VerifyPeerCertificate = func(rawCerts [][]byte, _ [][]*x509.Certificate) error {
			intermediatePool := x509.NewCertPool()
			var leaf *x509.Certificate
			for i, rawCert := range rawCerts {
				cert, err := x509.ParseCertificate(rawCert)
				if err != nil {
					return err
				}
				if i == 0 {
					leaf = cert
				} else {
					intermediatePool.AddCert(cert)
				}
			}
			if leaf == nil {
				return errors.New("no cert")
			}
			_, verifyErr := leaf.Verify(x509.VerifyOptions{
				DNSName:       serverName,
				Intermediates: intermediatePool,
				Roots:         c.certPool,
			})
			return verifyErr
		}
	}

	return tlsConf
}

func (c *endpointsTestCase) endpoint() string {
	return endpointForTargetPort(c.targetPort)
}

func (c *endpointsTestCase) Run(t *testing.T, testCtx *endpointsTestContext) {
	c.runConnectionTest(t, testCtx)

	if c.expectConnectFailure {
		return
	}

	c.runGRPCTest(t, testCtx)
	c.runHTTPTest(t, testCtx, false)
	c.runHTTPTest(t, testCtx, true)
}

func (c *endpointsTestCase) verifyDialResult(t *testing.T, conn *tls.Conn, err error) {
	if conn != nil {
		defer utils.IgnoreError(conn.Close)
	}
	if err == nil {
		err = conn.Handshake()
	}

	if !c.expectConnectFailure {
		assert.NoError(t, err, "expected no connection failure")
		return
	}

	if err == nil {
		_ = conn.SetReadDeadline(time.Now().Add(timeout))
		_, err = conn.Read(make([]byte, 1))
	}
	if assert.Error(t, err, "expected an error after TLS handshake") {
		assert.Equalf(t, err.Error(), "remote error: tls: bad certificate", "expected a bad certificate error after handshake, got: %v", err)
	}
}

func (c *endpointsTestCase) runConnectionTest(t *testing.T, testCtx *endpointsTestContext) {
	if c.skipTLS {
		if c.expectConnectFailure {
			require.Fail(t, "malformed test spec: cannot expect connection failures in non-TLS mode")
		}

		conn, err := dialer.Dial("tcp", c.endpoint())
		require.NoError(t, err, "expected connection attempt to succeed in plaintext mode")
		_ = conn.Close()
		return
	}

	// Test connecting without SNI
	require.NotEmpty(t, c.validServerNames, "need at least one valid server name")
	defaultServerName := c.validServerNames[0]

	tlsConf := testCtx.tlsConfig(c.clientCert, defaultServerName, false)
	conn, err := tls.DialWithDialer(&dialer, "tcp", c.endpoint(), tlsConf)
	c.verifyDialResult(t, conn, err)

	// Test connecting with all valid server names
	for _, serverName := range c.validServerNames {
		tlsConf := testCtx.tlsConfig(c.clientCert, serverName, true)
		conn, err := tls.DialWithDialer(&dialer, "tcp", c.endpoint(), tlsConf)
		c.verifyDialResult(t, conn, err)
	}

	// Test connecting with all invalid server names
	invalidServerNames := set.NewStringSet(testCtx.allServerNames...)
	invalidServerNames.RemoveAll(c.validServerNames...)
	for serverName := range invalidServerNames {
		tlsConf := testCtx.tlsConfig(c.clientCert, serverName, true)
		conn, err := tls.DialWithDialer(&dialer, "tcp", c.endpoint(), tlsConf)
		if conn != nil {
			_ = conn.Close()
		}
		_, ok := err.(x509.HostnameError)
		assert.True(t, ok, "expected error to be of type x509.HostnameError, was: %T (%v)", err, err)
	}
}

func (c *endpointsTestCase) verifyAuthStatus(t *testing.T, testCtx *endpointsTestContext, authStatus *v1.AuthStatus) {
	switch id := authStatus.GetId().(type) {
	case *v1.AuthStatus_ServiceId:
		assert.Equal(t, serviceAuth, c.expectAuth, "got service ID from auth status, expected this to be a service client")
	case *v1.AuthStatus_UserId:
		if assert.Equal(t, userAuth, c.expectAuth, "got user ID from auth status, expected this to be a non-service client") {
			assert.Equal(t, userPKIProviderName, authStatus.GetAuthProvider().GetName())
		}
	default:
		assert.Failf(t, "invalid ID type in auth status", "got type: %T", id)
	}
}

func (c *endpointsTestCase) runGRPCTest(t *testing.T, testCtx *endpointsTestContext) {
	var creds credentials.TransportCredentials
	if c.skipTLS {
		creds = insecure.NewCredentials()
	} else {
		creds = credentials.NewTLS(testCtx.tlsConfig(c.clientCert, c.validServerNames[0], true))
	}
	conn, err := grpc.Dial(c.endpoint(), grpc.WithTransportCredentials(creds))
	if !assert.NoError(t, err, "expected gRPC dial to succeed") {
		return
	}
	if conn != nil {
		defer utils.IgnoreError(conn.Close)
	}

	mdClient := v1.NewMetadataServiceClient(conn)
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	_, err = mdClient.GetMetadata(ctx, &v1.Empty{})
	if !c.expectGRPCSuccess {
		assert.Error(t, err, "expected GetMetadata request to fail")
		return
	}
	assert.NoError(t, err, "expected GetMetadata request to succeed")

	authClient := v1.NewAuthServiceClient(conn)
	ctx, cancel = context.WithTimeout(context.Background(), timeout)
	defer cancel()

	authStatus, err := authClient.GetAuthStatus(ctx, &v1.Empty{})
	if c.expectAuth == noAuth {
		assert.Error(t, err, "expected get auth status request to fail")
		statusPB, _ := status.FromError(err)
		if assert.NotNil(t, statusPB, "expected gRPC error to have an associated status") {
			assert.Equal(t, codes.Unauthenticated, statusPB.Code(), "expected error code to be `Unauthenticated`")
		}
		return
	}

	c.verifyAuthStatus(t, testCtx, authStatus)
}

func (c *endpointsTestCase) runHTTPTest(t *testing.T, testCtx *endpointsTestContext, useHTTP2 bool) {
	assert.True(t, c.skipTLS == (len(c.validServerNames) == 0), "invalid test case: either skipTLS is set or validServerNames are provided")

	var scheme string
	var transport http.RoundTripper
	var targetHost string
	if c.skipTLS {
		scheme = "http"
		targetHost = c.endpoint()
		if useHTTP2 {
			transport = &http2.Transport{
				AllowHTTP: true,
				DialTLS: func(network string, _ string, _ *tls.Config) (net.Conn, error) {
					return dialer.Dial(network, c.endpoint())
				},
			}
		}
	} else {
		scheme = "https"
		targetHost = c.validServerNames[0]
		tlsConfig := testCtx.tlsConfig(c.clientCert, targetHost, true)
		if useHTTP2 {
			transport = &http2.Transport{
				DialTLS: func(network string, _ string, tlsConf *tls.Config) (net.Conn, error) {
					return tls.Dial(network, c.endpoint(), tlsConf)
				},
				TLSClientConfig: tlsConfig,
			}
		} else {
			transport = &http.Transport{
				DialTLSContext: func(ctx context.Context, network, _ string) (net.Conn, error) {
					return (&tls.Dialer{Config: tlsConfig}).DialContext(ctx, network, c.endpoint())
				},
			}
		}
	}

	client := &http.Client{
		Transport: transport,
		Timeout:   timeout,
	}

	resp, err := client.Get(fmt.Sprintf("%s://%s/v1/metadata", scheme, targetHost))
	if resp != nil {
		defer utils.IgnoreError(resp.Body.Close)
	}
	if !c.expectHTTPSuccess {
		// If we're in this branch, that means we're speaking to a gRPC-only server, which cannot handle normal HTTP
		// requests.
		if resp == nil {
			assert.Error(t, err, "expected HTTP request to fail at the transport level")
		} else {
			assert.Equal(t, http.StatusUnsupportedMediaType, resp.StatusCode, "expected HTTP request to fail")
		}
		return
	}
	if !assert.NoError(t, err, "expected HTTP request to succeed at the transport level") {
		return
	}

	assert.Equal(t, http.StatusOK, resp.StatusCode, "expected 200 status code for metadata request")
	var md v1.Metadata
	assert.NoError(t, jsonpb.Unmarshal(resp.Body, &md), "expected response for metadata request to be unmarshalable into metadata PB")

	resp, err = client.Get(fmt.Sprintf("%s://%s/v1/auth/status", scheme, targetHost))
	if !assert.NoError(t, err, "expected HTTP request to succeed at the transport level") {
		return
	}
	defer utils.IgnoreError(resp.Body.Close)

	if c.expectAuth == noAuth {
		assert.Equal(t, http.StatusUnauthorized, resp.StatusCode, "expected 401 status code for auth status request")
		return
	}

	var authStatus v1.AuthStatus
	if !assert.NoError(t, jsonpb.Unmarshal(resp.Body, &authStatus), "expected response for auth status request to be unmarshalable into auth status PB") {
		return
	}
	c.verifyAuthStatus(t, testCtx, &authStatus)
}

func TestEndpoints(t *testing.T) {
	userCert, err := tls.LoadX509KeyPair(os.Getenv("CLIENT_CERT_PATH"), os.Getenv("CLIENT_KEY_PATH"))
	require.NoError(t, err, "failed to load user certificate")

	serviceCert, err := tls.LoadX509KeyPair(os.Getenv("SERVICE_CERT_FILE"), os.Getenv("SERVICE_KEY_FILE"))
	require.NoError(t, err, "failed to load service certificate")

	trustPool := x509.NewCertPool()
	serviceCAPEMBytes, err := os.ReadFile(os.Getenv("SERVICE_CA_FILE"))
	require.NoError(t, err, "failed to load service CA file")
	serviceCACert, err := helpers.ParseCertificatePEM(serviceCAPEMBytes)
	require.NoError(t, err, "failed to parse service CA cert")
	trustPool.AddCert(serviceCACert)

	defaultCAPEMBytes, err := os.ReadFile(os.Getenv("DEFAULT_CA_FILE"))
	require.NoError(t, err, "failed to load default CA file")
	defaultCACert, err := helpers.ParseCertificatePEM(defaultCAPEMBytes)
	require.NoError(t, err, "failed to parse default CA cert")
	trustPool.AddCert(defaultCACert)

	defaultCertDNSName := os.Getenv("ROX_TEST_CENTRAL_CN")
	require.NotEmpty(t, defaultCertDNSName, "missing default certificate DNS name")

	testCtx := &endpointsTestContext{
		allServerNames: []string{defaultCertDNSName, "central.stackrox"},
		certPool:       trustPool,
	}

	cases := map[string]endpointsTestCase{
		"default port with no client cert": {
			targetPort:        8443,
			validServerNames:  testCtx.allServerNames,
			expectHTTPSuccess: true,
			expectGRPCSuccess: true,
		},
		"default port with service client cert": {
			targetPort:        8443,
			validServerNames:  testCtx.allServerNames,
			clientCert:        &serviceCert,
			expectAuth:        serviceAuth,
			expectGRPCSuccess: true,
			expectHTTPSuccess: true,
		},
		"default port with user client cert": {
			targetPort:        8443,
			validServerNames:  testCtx.allServerNames,
			clientCert:        &userCert,
			expectAuth:        userAuth,
			expectGRPCSuccess: true,
			expectHTTPSuccess: true,
		},
		"multiplexed plaintext port": {
			targetPort:        8080,
			skipTLS:           true,
			expectGRPCSuccess: true,
			expectHTTPSuccess: true,
		},
		"http-only plaintext port": {
			targetPort:        8082,
			skipTLS:           true,
			expectGRPCSuccess: false,
			expectHTTPSuccess: true,
		},
		"grpc-only plaintext port": {
			targetPort:        8081,
			skipTLS:           true,
			expectGRPCSuccess: true,
			expectHTTPSuccess: false,
		},
		"service-only TLS port with no client cert": {
			targetPort:           8444,
			validServerNames:     []string{"central.stackrox"},
			expectConnectFailure: true,
		},
		"service-only TLS port with service client cert": {
			targetPort:        8444,
			validServerNames:  []string{"central.stackrox"},
			clientCert:        &serviceCert,
			expectAuth:        serviceAuth,
			expectGRPCSuccess: true,
			expectHTTPSuccess: true,
		},
		"service-only TLS port with user client cert": {
			targetPort:           8444,
			validServerNames:     []string{"central.stackrox"},
			clientCert:           &userCert,
			expectConnectFailure: true,
		},
		"user-only TLS port with no client cert": {
			targetPort:        8445,
			validServerNames:  []string{defaultCertDNSName},
			expectGRPCSuccess: true,
			expectHTTPSuccess: true,
		},
		"user-only TLS port with service client cert": {
			targetPort:        8445,
			validServerNames:  []string{defaultCertDNSName},
			clientCert:        &serviceCert,
			expectGRPCSuccess: true,
			expectHTTPSuccess: true,
		},
		"user-only TLS port with user client cert": {
			targetPort:        8445,
			validServerNames:  []string{defaultCertDNSName},
			clientCert:        &userCert,
			expectAuth:        userAuth,
			expectGRPCSuccess: true,
			expectHTTPSuccess: true,
		},
		"http-only TLS port with no client cert": {
			targetPort:        8446,
			validServerNames:  testCtx.allServerNames,
			expectGRPCSuccess: false,
			expectHTTPSuccess: true,
		},
		"http-only TLS port with service client cert": {
			targetPort:        8446,
			validServerNames:  testCtx.allServerNames,
			clientCert:        &serviceCert,
			expectAuth:        serviceAuth,
			expectGRPCSuccess: false,
			expectHTTPSuccess: true,
		},
		"http-only TLS port with user client cert": {
			targetPort:        8446,
			validServerNames:  testCtx.allServerNames,
			clientCert:        &userCert,
			expectAuth:        userAuth,
			expectGRPCSuccess: false,
			expectHTTPSuccess: true,
		},
		"grpc-only client-auth required TLS port with no client cert": {
			targetPort:           8447,
			validServerNames:     testCtx.allServerNames,
			expectConnectFailure: true,
		},
		"grpc-only client-auth required TLS port with service client cert": {
			targetPort:        8447,
			validServerNames:  testCtx.allServerNames,
			clientCert:        &serviceCert,
			expectAuth:        serviceAuth,
			expectGRPCSuccess: true,
			expectHTTPSuccess: false,
		},
		"grpc-only client-auth required TLS port with user client cert": {
			targetPort:        8447,
			validServerNames:  testCtx.allServerNames,
			clientCert:        &userCert,
			expectAuth:        userAuth,
			expectGRPCSuccess: true,
			expectHTTPSuccess: false,
		},
		"multiplexed client CA-less TLS port with no client cert": {
			targetPort:        8448,
			validServerNames:  testCtx.allServerNames,
			expectGRPCSuccess: true,
			expectHTTPSuccess: true,
		},
		"multiplexed client CA-less TLS port with service client cert": {
			targetPort:        8448,
			validServerNames:  testCtx.allServerNames,
			clientCert:        &serviceCert,
			expectGRPCSuccess: true,
			expectHTTPSuccess: true,
		},
		"multiplexed client CA-less TLS port with user client cert": {
			targetPort:        8448,
			validServerNames:  testCtx.allServerNames,
			clientCert:        &userCert,
			expectGRPCSuccess: true,
			expectHTTPSuccess: true,
		},
	}

	for desc, tc := range cases {
		t.Run(desc, func(t *testing.T) {
			tc.Run(t, testCtx)
		})
	}
}
