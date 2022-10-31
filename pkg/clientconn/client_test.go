package clientconn

import (
	"crypto/x509"
	"os"
	"path"
	"testing"

	"github.com/stackrox/rox/pkg/mtls/verifier"
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
