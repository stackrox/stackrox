package acscsemail

import (
	"bytes"
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"net/http"
	"os"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/notifiers/acscsemail/message"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/httputil/proxy"
	"github.com/stackrox/rox/pkg/satoken"
	"github.com/stackrox/rox/pkg/utils"
)

// serviceOperatorCAPath points to the secret of the service account, which within an OpenShift environment
// also has the service-ca.crt, which includes the CA to verify certificates issued by the service-ca operator.
// The service-ca operator is used to issue the certificate used by the emailsender service in ACSCS
const serviceOperatorCAPath = "/var/run/secrets/kubernetes.io/serviceaccount/service-ca.crt"

const sendMsgPath = "/api/v1/acscsemail"

type clientImpl struct {
	loadToken  func() (string, error)
	url        string
	httpClient *http.Client
}

var _ Client = &clientImpl{}

var client *clientImpl

// ClientSingleton returns an instance of the default ACSCS email service client implementation.
// It uses HTTP to communicate to the ACSCS email service instead of SMTP, because the ACSCS email service
// acts as a direct proxy to underlying cloud email service APIs e.g. AWS SES. Using HTTP here has several benefits:
// 1. Matches the core knowledge of ACSCS team
// 2. Authentication with K8s service account tokens is easier
// 3. Reduced latency and complexity compared to SMTP, since SMTP involves a lot of back and forth messaging
// between client and server.
func ClientSingleton() Client {
	if client != nil {
		return client
	}

	url := fmt.Sprintf("%s:%s", env.ACSCSEmailURL.Setting(), sendMsgPath)

	client = &clientImpl{
		loadToken:  satoken.LoadTokenFromFile,
		url:        url,
		httpClient: &http.Client{Transport: transportWithServiceCA()},
	}

	return client
}

func transportWithServiceCA() http.RoundTripper {
	return transportWithAdditionalCA(serviceOperatorCAPath)
}

func transportWithAdditionalCA(caFile string) *http.Transport {
	rootCAs, err := x509.SystemCertPool()
	if err != nil {
		rootCAs = x509.NewCertPool()
	}

	// Trust local cluster services.
	if serviceCA, err := os.ReadFile(caFile); err == nil {
		rootCAs.AppendCertsFromPEM(serviceCA)
	}

	return &http.Transport{
		TLSClientConfig: &tls.Config{
			RootCAs: rootCAs,
		},
		Proxy: proxy.FromConfig(),
	}
}

func (c *clientImpl) SendMessage(ctx context.Context, msg message.AcscsEmail) error {
	token, err := c.loadToken()
	if err != nil {
		return errors.Wrap(err, "failed to load authorization token")
	}

	msgBytes, err := json.Marshal(&msg)
	if err != nil {
		return errors.Wrap(err, "failed to marshal message")
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.url, bytes.NewReader(msgBytes))
	if err != nil {
		return errors.Wrap(err, "failed to build HTTP requests")
	}

	req.Header.Set("Content-Type", "application/json; charset=UTF-8")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))

	res, err := c.httpClient.Do(req)
	if err != nil {
		return errors.Wrap(err, "failed to send HTTP request")
	}
	defer utils.IgnoreError(res.Body.Close)

	if res.StatusCode > 300 {
		return fmt.Errorf("request to %s failed with HTTP status: %d", c.url, res.StatusCode)
	}

	return nil
}
