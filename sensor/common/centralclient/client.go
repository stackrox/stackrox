package centralclient

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/golang/protobuf/proto"
	"github.com/pkg/errors"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/centralsensor"
	"github.com/stackrox/rox/pkg/clientconn"
	"github.com/stackrox/rox/pkg/cryptoutils"
	"github.com/stackrox/rox/pkg/httputil"
	"github.com/stackrox/rox/pkg/httputil/proxy"
	"github.com/stackrox/rox/pkg/jsonutil"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/mtls"
	"github.com/stackrox/rox/pkg/mtls/verifier"
	"github.com/stackrox/rox/pkg/utils"
	"github.com/stackrox/rox/pkg/x509utils"
)

var (
	log = logging.LoggerForModule()
)

const (
	requestTimeout          = 10 * time.Second
	tlsChallengeRoute       = "/v1/tls-challenge"
	pingRoute               = "/v1/ping"
	challengeTokenParamName = "challengeToken"
)

// Client is a client which provides functions to call rest routes in central
type Client struct {
	endpoint       *url.URL
	httpClient     *http.Client
	nonceGenerator cryptoutils.NonceGenerator
}

// NewClient creates a new client
func NewClient(endpoint string) (*Client, error) {
	if endpoint == "" {
		return nil, errors.New("creating Client with empty endpoint is not allowed")
	}

	parts := strings.SplitN(endpoint, "://", 2)
	switch parts[0] {
	case "wss":
		endpoint = fmt.Sprintf("https://%s", parts[1])
	case "https":
		break
	default:
		if len(parts) == 1 {
			endpoint = fmt.Sprintf("https://%s", endpoint)
			break
		}
		return nil, errors.Errorf("creating client unsupported scheme %s", parts[0])
	}

	endpointURL, err := url.Parse(endpoint)
	if err != nil {
		return nil, errors.Wrap(err, "parsing endpoint url")
	}

	// Load the client certificate. Note that while all endpoints accessed by the client do not require
	// authentication, it is possible that a user has required client certificate authentication for the
	// endpoint Sensor is connecting to. Since a client certificate can be used without harm even if the
	// remote is not trusted, make it available here to be on the safe side.
	//
	// Moreover, authentication requirements can be tightened in future and thus having an older version
	// of Sensor authenticating itself will enable backward compatibility with newer Centrals. This has
	// indeed happened in the past when `/v1/metadata` became authenticated.
	clientCert, err := mtls.LeafCertificateFromFile()
	if err != nil {
		return nil, errors.Wrap(err, "obtaining client certificate")
	}
	tlsConf := &tls.Config{
		InsecureSkipVerify: true,
		Certificates: []tls.Certificate{
			clientCert,
		},
	}
	httpClient := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: tlsConf,
			Proxy:           proxy.FromConfig(),
		},
		Timeout: requestTimeout,
	}

	return &Client{
		httpClient:     httpClient,
		endpoint:       endpointURL,
		nonceGenerator: cryptoutils.NewNonceGenerator(centralsensor.ChallengeTokenLength, nil),
	}, nil
}

// GetPing pings Central.
func (c *Client) GetPing(ctx context.Context) (*v1.PongMessage, error) {
	resp, _, err := c.doHTTPRequest(ctx, http.MethodGet, pingRoute, nil, nil)
	if err != nil {
		return nil, errors.Wrap(err, "pinging Central")
	}
	defer utils.IgnoreError(resp.Body.Close)

	var pong v1.PongMessage
	if err := jsonutil.JSONReaderToProto(resp.Body, &pong); err != nil {
		return nil, errors.Wrapf(err, "parsing Central %s response with status code %d", pingRoute, resp.StatusCode)
	}

	return &pong, nil
}

// GetTLSTrustedCerts returns all certificates which are trusted by Central and its leaf certificates.
// Sensor validates the identity of Central by verifying the given signature against Central's public key presented by its leaf cert.
func (c *Client) GetTLSTrustedCerts(ctx context.Context) ([]*x509.Certificate, error) {
	token, err := c.generateChallengeToken()
	if err != nil {
		return nil, errors.Wrap(err, "creating challenge token")
	}

	resp, hostCertChain, err := c.doTLSChallengeRequest(ctx, &v1.TLSChallengeRequest{ChallengeToken: token})
	if err != nil {
		return nil, err
	}

	trustInfo, err := c.parseTLSChallengeResponse(resp)
	if err != nil {
		return nil, errors.Wrap(err, "verifying tls challenge")
	}

	if trustInfo.SensorChallenge != token {
		return nil, errors.Errorf("validating Central response failed: Sensor token %q did not match received token %q", token, trustInfo.SensorChallenge)
	}

	var certs []*x509.Certificate
	for _, ca := range trustInfo.GetAdditionalCas() {
		cert, err := x509.ParseCertificate(ca)
		if err != nil {
			return nil, errors.Wrap(err, "parsing additional CA")
		}
		certs = append(certs, cert)
	}

	leafCert := hostCertChain[0]
	if !issuedByStackRoxCA(leafCert) {
		certPool, err := x509.SystemCertPool()
		if err != nil {
			return nil, errors.Wrap(err, "failed to get trusted certificate pool")
		}
		for _, cert := range certs {
			certPool.AddCert(cert)
		}

		err = hostCertChain[0].VerifyHostname(c.endpoint.Hostname())
		if err != nil {
			return nil, errors.Wrapf(err, "host leaf certificate can't be verified against hostname %s", c.endpoint.Hostname())
		}

		err = x509utils.VerifyCertificateChain(hostCertChain, x509.VerifyOptions{
			Roots: certPool,
		})

		if err != nil {
			return certs, newAdditionalCANeededErr(leafCert.DNSNames, c.endpoint.Hostname(), err.Error())
		}
	}

	return certs, nil
}

func issuedByStackRoxCA(proxyCert *x509.Certificate) bool {
	return proxyCert.Issuer.CommonName == mtls.ServiceCACommonName
}

func (c *Client) parseTLSChallengeResponse(challenge *v1.TLSChallengeResponse) (*v1.TrustInfo, error) {
	var trustInfo v1.TrustInfo
	err := proto.Unmarshal(challenge.GetTrustInfoSerialized(), &trustInfo)
	if err != nil {
		return nil, errors.Wrap(err, "parsing TrustInfo proto")
	}

	if len(trustInfo.GetCertChain()) == 0 {
		return nil, errors.New("reading Central's leaf certificate from response")
	}

	rootCAs, err := verifier.TrustedCertPool()
	if err != nil {
		return nil, errors.Wrap(err, "reading CA cert")
	}

	x509CertChain, err := x509utils.ParseCertificateChain(trustInfo.GetCertChain())
	if err != nil {
		return nil, errors.Wrap(err, "parsing Central cert chain")
	}

	if len(x509CertChain) == 0 {
		return nil, errors.New("parsing Central chain was empty, expected certificate chain")
	}

	err = verifyCentralCertificateChain(x509CertChain, rootCAs)
	if err != nil {
		return nil, errors.Wrap(err, "validating certificate chain")
	}

	err = verifySignatureAgainstCertificate(x509CertChain[0], challenge.TrustInfoSerialized, challenge.Signature)
	if err != nil {
		return nil, errors.Wrap(err, "validating payload signature")
	}
	return &trustInfo, nil
}

// doTLSChallengeRequest send the HTTP request to Central and receives the trust info.
func (c *Client) doTLSChallengeRequest(ctx context.Context, req *v1.TLSChallengeRequest) (*v1.TLSChallengeResponse, []*x509.Certificate, error) {
	params := url.Values{challengeTokenParamName: []string{req.GetChallengeToken()}}

	resp, peerCertificates, err := c.doHTTPRequest(ctx, http.MethodGet, tlsChallengeRoute, params, nil)
	if err != nil {
		return nil, peerCertificates, errors.Wrap(err, "receiving Central's trust info")
	}
	defer utils.IgnoreError(resp.Body.Close)

	tlsChallengeResp := &v1.TLSChallengeResponse{}
	err = jsonutil.JSONReaderToProto(resp.Body, tlsChallengeResp)
	if err != nil {
		return nil, peerCertificates, errors.Wrap(err, "parsing Central response")
	}
	return tlsChallengeResp, peerCertificates, nil
}

func (c *Client) doHTTPRequest(ctx context.Context, method, route string, params url.Values, body io.Reader) (*http.Response, []*x509.Certificate, error) {
	u := *c.endpoint
	u.Path = route
	u.RawQuery = params.Encode()

	req, err := http.NewRequestWithContext(ctx, method, u.String(), body)
	if err != nil {
		return nil, nil, errors.Wrapf(err, "creating request for %s", u.String())
	}

	req.Header.Set("User-Agent", clientconn.GetUserAgent())

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, nil, errors.Wrapf(err, "calling %s", u.String())
	}

	peerCertificates := resp.TLS.PeerCertificates
	if len(peerCertificates) == 0 {
		return nil, nil, errors.New("no peer certificates found in HTTP request")
	}

	if !httputil.Is2xxStatusCode(resp.StatusCode) {
		body, err := io.ReadAll(resp.Body)

		if err != nil {
			return nil, peerCertificates, errors.Wrapf(err, "reading response body with HTTP status code '%s'", resp.Status)
		}
		return nil, peerCertificates, errors.Errorf("HTTP request %s with code '%s', body: %s", u.String(), resp.Status, body)
	}
	return resp, peerCertificates, nil
}

func (c *Client) generateChallengeToken() (string, error) {
	challenge, err := c.nonceGenerator.Nonce()
	if err != nil {
		return "", err
	}

	return challenge, nil
}
