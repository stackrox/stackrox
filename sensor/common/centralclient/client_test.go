package centralclient

import (
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path"
	"testing"

	"github.com/stackrox/rox/pkg/centralsensor"
	"github.com/stackrox/rox/pkg/testutils"
	"github.com/stretchr/testify/suite"
)

const (
	endpoint         = "localhost:8000"
	trustInfoExample = "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"
	signatureExample = "IV6XZmBRj3T6mqAaAzaYe41W//E7EIONwhu0GOyWqlRLRiPoi4T68NhhCL5GUuPFxPmCv0Ix+WrwouYtu9LOLWtk57Fxk3eQvgzx9AYPsdrBxA/xdI3f4xIwy0pfFr66SJF9C0I6/vJyhSnWs4uom9NlasIPG2aWNd0w+lotz8WR21QjEwK+SIo+GH8VykRRBITVyNtGAh+e8N4pkNzZGfC+vhb/xb/qZbwmFIHjRzRGEL6otmreoe2rqyuwuZyNFmUwjLEJLiQ+MkxMftqK6lVVSfRL4TtW4QIfwaQwfIKec2z1BXsPPvIcLJWMG4PGHuAgMdKcGMOa6aEkf7YAHg=="
	// invalidSignature signature signed by a different private key
	invalidSignature = "GkMQkkfqyTF6laWBKi09Kqjxq5+GDV1Sk78KoiavfZCS8OuFMv66Z1N2cHW5w68RzOmgpLAoMli9/8JKTq9qpT2Txr1YYIjabKOMLh3Mzit2p2esGZ8ekuAP+0bw7L1icbWOX8IDgt8/5uaKaxzQ6YhyoGegcqNDr0EFyRpU9rlmk4CJ+ouNTjBDax1JxOyG/ThFsJa8IeLztXsTA86OEuQlKKfXpgcB0Kshw/pPUB1UbMEkMPVudcwtZFMua+UxcIhq5Pe1sAtBxVcMEp2linexxenFcBVHQXZfMgpKvfZ8iBbpd/Ye8FCq1U/8xFwVAgdj+uMSewBUHTq1EOTNrQ=="
)

func TestServiceImpl(t *testing.T) {
	suite.Run(t, new(ClientTestSuite))
}

type ClientTestSuite struct {
	suite.Suite
	envIsolator *testutils.EnvIsolator
}

func (t *ClientTestSuite) SetupSuite() {
	t.envIsolator = testutils.NewEnvIsolator(t.T())
}

func (t *ClientTestSuite) SetupTest() {
	wd, _ := os.Getwd()
	testdata := path.Join(wd, "testdata")

	t.envIsolator.Setenv("ROX_MTLS_CA_FILE", path.Join(testdata, "ca.pem"))
}

func (t *ClientTestSuite) TearDownTest() {
	t.envIsolator.RestoreAll()
}

func (t *ClientTestSuite) TestGetTLSTrustedCerts_GetCertificate() {
	ts := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sensorChallengeToken := r.URL.Query().Get("challengeToken")
		sensorChallengeTokenBytes, err := base64.URLEncoding.DecodeString(sensorChallengeToken)
		t.Require().NoError(err)
		t.Assert().Len(sensorChallengeTokenBytes, centralsensor.ChallengeTokenLength)

		resp := fmt.Sprintf(`{"trustInfoSerialized":"%s", "signature":"%s"}`, trustInfoExample, signatureExample)
		_, _ = w.Write([]byte(resp))
	}))
	defer ts.Close()

	c, err := NewClient(ts.URL)
	t.Require().NoError(err)

	_, err = c.GetTLSTrustedCerts()
	t.Require().NoError(err)
}

func (t *ClientTestSuite) TestGetTLSTrustedCerts_WithSignatureSignedByAnotherPrivateKey_ShouldFail() {
	ts := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := fmt.Sprintf(`{"trustInfoSerialized":"%s", "signature":"%s"}`, trustInfoExample, invalidSignature)
		_, _ = w.Write([]byte(resp))
	}))
	defer ts.Close()

	c, err := NewClient(ts.URL)
	t.Require().NoError(err)

	_, err = c.GetTLSTrustedCerts()
	t.Require().Error(err)
	t.Equal("verifying tls challenge: verifying central trust info signature: failed to verify rsa signature: crypto/rsa: verification error", err.Error())
}

func (t *ClientTestSuite) TestGetTLSTrustedCerts_WithInvalidTrustInfo() {
	ts := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := fmt.Sprintf(`{"trustInfoSerialized":"%s", "signature":"%s"}`, base64.StdEncoding.EncodeToString([]byte("Invalid trust info")), signatureExample)
		_, _ = w.Write([]byte(resp))
	}))
	defer ts.Close()

	c, err := NewClient(ts.URL)
	t.Require().NoError(err)

	_, err = c.GetTLSTrustedCerts()
	t.Require().Error(err)
	t.True(errors.Is(err, io.ErrUnexpectedEOF))
}

func (t *ClientTestSuite) TestGetTLSTrustedCerts_WithInvalidSignature() {
	ts := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := fmt.Sprintf(`{"trustInfoSerialized":"%s", "signature":"%s"}`, trustInfoExample, base64.StdEncoding.EncodeToString([]byte("Invalid signature")))
		_, _ = w.Write([]byte(resp))
	}))
	defer ts.Close()

	c, err := NewClient(ts.URL)
	t.Require().NoError(err)

	_, err = c.GetTLSTrustedCerts()
	t.Require().Error(err)
	t.Contains(err.Error(), "verifying tls challenge: verifying central trust info signature: failed to verify rsa signature: crypto/rsa: verification error")
}

func (t *ClientTestSuite) Test_NewClientReplacesProtocols() {
	// By default HTTPS will be prepended
	c, err := NewClient(endpoint)
	t.Require().NoError(err)
	t.Equal(fmt.Sprintf("https://%s", endpoint), c.endpoint)

	// HTTPS is accepted
	c, err = NewClient(fmt.Sprintf("https://%s", endpoint))
	t.Require().NoError(err)
	t.Equal(fmt.Sprintf("https://%s", endpoint), c.endpoint)

	// WebSockets are converted to HTTPS
	c, err = NewClient(fmt.Sprintf("wss://%s", endpoint))
	t.Require().NoError(err)
	t.Equal(fmt.Sprintf("https://%s", endpoint), c.endpoint)

	// HTTP is not accepted
	_, err = NewClient(fmt.Sprintf("http://%s", endpoint))
	t.Require().Error(err)
	t.Equal("creating client unsupported scheme http", err.Error())
}
