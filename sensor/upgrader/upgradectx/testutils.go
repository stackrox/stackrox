package upgradectx

import (
	"context"
	"net/http"
	"testing"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/clientconn"
	"github.com/stackrox/rox/pkg/mtls"
	"github.com/stackrox/rox/sensor/upgrader/config"
)

// CreateForTest creates a test upgrader context from the given config.
func CreateForTest(
	ctx context.Context,
	_ testing.TB,
	config *config.UpgraderConfig,
) (*UpgradeContext, error) {
	transport, err := clientconn.AuthenticatedHTTPTransport(
		config.CentralEndpoint,
		mtls.CentralSubject,
		nil,
		clientconn.UseServiceCertToken(false),
		clientconn.UseInsecureNoTLS(true),
	)
	if err != nil {
		return nil, errors.Wrap(err, "failed to initialize HTTP transport to Central")
	}
	c := &UpgradeContext{
		ctx:    ctx,
		config: *config,
		centralHTTPClient: &http.Client{
			Transport: transport,
		},
	}
	return c, nil
}
