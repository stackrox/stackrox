package acscsemail

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"io"
	"math/big"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path"
	"testing"
	"time"

	"github.com/stackrox/rox/central/notifiers/acscsemail/message"
	"github.com/stackrox/rox/pkg/certgen"
	"github.com/stackrox/rox/pkg/mtls"
	"github.com/stackrox/rox/pkg/retry"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSendMessage(t *testing.T) {

	fakeTokenFunc := func() (string, error) {
		return "test-token", nil
	}

	tokenErr := errors.New("token error")
	fakeTokenErrFunc := func() (string, error) {
		return "", tokenErr
	}

	defaultMsg := message.AcscsEmail{
		To:         []string{"test@test.acscs-email.test"},
		RawMessage: []byte("test message content"),
	}

	defaultContext := context.Background()

	tests := map[string]struct {
		tokenFunc       func() (string, error)
		inputMessage    message.AcscsEmail
		expectedError   error
		ctx             context.Context
		response        *http.Response
		expectedHeader  http.Header
		expectedBodyStr string
	}{
		"error on loadToken": {
			tokenFunc:     fakeTokenErrFunc,
			expectedError: tokenErr,
			inputMessage:  defaultMsg,
			ctx:           defaultContext,
		},
		"error on invalid context": {
			tokenFunc:     fakeTokenFunc,
			expectedError: errors.New("failed to build HTTP"),
			inputMessage:  defaultMsg,
			// ctx nil causes an error on http.NewRequest
			ctx: nil,
		},
		"error on bad status code": {
			tokenFunc:     fakeTokenFunc,
			inputMessage:  defaultMsg,
			expectedError: errors.New("failed with HTTP status: 400"),
			ctx:           defaultContext,
			response: &http.Response{
				StatusCode: 400,
			},
		},
		"successful request": {
			tokenFunc:    fakeTokenFunc,
			inputMessage: defaultMsg,
			ctx:          defaultContext,
			response: &http.Response{
				StatusCode: 200,
			},
			expectedHeader: map[string][]string{
				"Content-Type":  {"application/json; charset=UTF-8"},
				"Authorization": {"Bearer test-token"},
			},
			// RawMessage is the b64 encoded value of "test message content" defined in the inputMessage
			expectedBodyStr: `{"to":["test@test.acscs-email.test"],"rawMessage":"dGVzdCBtZXNzYWdlIGNvbnRlbnQ="}`,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {

			httpClient, actualRequest := testClient(tc.response, tc.expectedError)
			client := clientImpl{
				loadToken:  tc.tokenFunc,
				url:        "http://localhost:8080",
				httpClient: httpClient,
			}

			err := client.SendMessage(tc.ctx, tc.inputMessage)
			if tc.expectedError != nil {
				require.Error(t, err)
				require.Contains(t, err.Error(), tc.expectedError.Error())
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tc.expectedHeader, actualRequest.Header)
			actualBody, err := io.ReadAll(actualRequest.Body)
			require.NoError(t, err, "error parsing request body")
			assert.Equal(t, tc.expectedBodyStr, string(actualBody))
		})
	}
}

func testClient(res *http.Response, returnErr error) (*http.Client, *http.Request) {
	var receivedRequest http.Request

	client := &http.Client{
		Transport: testRoundTripper(func(req *http.Request) (*http.Response, error) {
			receivedRequest = *req
			return res, returnErr
		}),
	}

	return client, &receivedRequest
}

type testRoundTripper func(req *http.Request) (*http.Response, error)

func (t testRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	return t(req)
}

func TestTransportWithAdditonalCA(t *testing.T) {
	ca, err := certgen.GenerateCA()
	require.NoError(t, err, "failed to generate test CA")

	caDir := t.TempDir()
	filePath := path.Join(caDir, "cert.pem")
	err = os.WriteFile(filePath, ca.CertPEM(), 0644)
	require.NoError(t, err, "failed to write test CA to file")

	testServerCalled := false
	tlsServ := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		testServerCalled = true
		w.WriteHeader(200)
	}))

	tlsServ.TLS = &tls.Config{
		Certificates: []tls.Certificate{generateTestServerCert(t, ca)},
	}

	tlsServ.StartTLS()
	defer tlsServ.Close()

	transport := transportWithAdditionalCA(filePath)
	httpClient := http.Client{Transport: transport}
	err = retry.WithRetry(
		// there's a chance the first call fails on tests depending on
		// server startup timing
		func() error {
			_, err := httpClient.Get(tlsServ.URL)
			return err
		},
		retry.Tries(3),
		retry.BetweenAttempts(func(_ int) {
			time.Sleep(1 * time.Second)
		}),
	)

	require.NoError(t, err, "expected HTTP call to test server to succeed")
	require.True(t, testServerCalled, "expected test server to be called succesfully")
}

func generateTestServerCert(t *testing.T, ca mtls.CA) tls.Certificate {
	template := &x509.Certificate{
		SerialNumber:          big.NewInt(1),
		NotBefore:             time.Now(),
		NotAfter:              time.Now().Add(time.Hour),
		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageAny},
		BasicConstraintsValid: true,
		IsCA:                  false,
		DNSNames:              []string{"localhost"},
		IPAddresses:           []net.IP{net.ParseIP("127.0.0.1")},
	}

	certKey, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err, "failed to generate test server TLS key")

	certDER, err := x509.CreateCertificate(rand.Reader, template, ca.Certificate(), certKey.Public(), ca.PrivateKey())
	require.NoError(t, err, "failed to generate test server TLS cert")

	certPem := &bytes.Buffer{}
	err = pem.Encode(certPem, &pem.Block{Type: "CERTIFICATE", Bytes: certDER})
	require.NoError(t, err, "failed to encode test server TLS cert to pem")

	keyDER, err := x509.MarshalPKCS8PrivateKey(certKey)
	require.NoError(t, err, "failed to marshal test server TLS key")
	keyPem := &bytes.Buffer{}
	err = pem.Encode(keyPem, &pem.Block{Type: "PRIVATE KEY", Bytes: keyDER})
	require.NoError(t, err, "failed to encode test server TLS key to pem")

	cert, err := tls.X509KeyPair(certPem.Bytes(), keyPem.Bytes())
	require.NoError(t, err, "failed to create test server TLS key pair")
	return cert
}
