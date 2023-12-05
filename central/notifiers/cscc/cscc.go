package cscc

import (
	"context"
	"fmt"
	"net/http"

	securitycenter "cloud.google.com/go/securitycenter/apiv1"
	"cloud.google.com/go/securitycenter/apiv1/securitycenterpb"
	"github.com/googleapis/gax-go/v2"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/cloudproviders/gcp"
	clusterDatastore "github.com/stackrox/rox/central/cluster/datastore"
	notifierUtils "github.com/stackrox/rox/central/notifiers/utils"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/administration/events/codes"
	adminOption "github.com/stackrox/rox/pkg/administration/events/option"
	gcpHandler "github.com/stackrox/rox/pkg/cloudproviders/gcp/handler"
	gcpUtils "github.com/stackrox/rox/pkg/cloudproviders/gcp/utils"
	"github.com/stackrox/rox/pkg/cryptoutils/cryptocodec"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/notifiers"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sac/resources"
	"github.com/stackrox/rox/pkg/utils"
)

var (
	log = logging.LoggerForModule(adminOption.EnableAdministrationEvents())

	clusterForAlertContext = sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.AllowFixedScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_ACCESS),
			sac.ResourceScopeKeys(resources.Cluster),
		))
)

func init() {
	cryptoKey := ""
	var err error
	if env.EncNotifierCreds.BooleanSetting() {
		cryptoKey, err = notifierUtils.GetNotifierSecretEncryptionKey()
		if err != nil {
			utils.Should(errors.Wrap(err, "Error reading encryption key, notifier will be unable to send notifications"))
		}
	}

	notifiers.Add(notifiers.CSCCType, func(notifier *storage.Notifier) (notifiers.Notifier, error) {
		j, err := newCSCC(notifier, cryptocodec.Singleton(), cryptoKey)
		return j, err
	})
}

type config struct {
	ServiceAccount string `json:"serviceAccount"`
	SourceID       string `json:"sourceID"`
}

// The Cloud SCC notifier plugin integrates with Google's Cloud Security Command Center.
type cscc struct {
	// The Service Account is a Google JSON service account key.
	// The GCP Organization ID is a numeric identifier for the Google Cloud Platform
	// organization. It is required so that we can tag findings to the right org.
	client gcpHandler.Handler[*securitycenter.Client]
	config *config
	*storage.Notifier
}

func newCSCC(protoNotifier *storage.Notifier, cryptoCodec cryptocodec.CryptoCodec, cryptoKey string) (*cscc, error) {
	conf := protoNotifier.GetCscc()
	if err := Validate(conf, !env.EncNotifierCreds.BooleanSetting()); err != nil {
		return nil, err
	}

	decCreds := conf.ServiceAccount
	var err error
	if env.EncNotifierCreds.BooleanSetting() {
		if protoNotifier.GetNotifierSecret() == "" {
			return nil, errors.Errorf("encrypted notifier credentials for notifier '%s' empty", protoNotifier.GetName())
		}
		decCreds, err = cryptoCodec.Decrypt(cryptoKey, protoNotifier.GetNotifierSecret())
		if err != nil {
			return nil, errors.Errorf("Error decrypting notifier secret for notifier '%s'", protoNotifier.GetName())
		}
	}

	var handler gcpHandler.Handler[*securitycenter.Client]
	if features.CloudCredentials.Enabled() {
		handler, err = gcpUtils.CreateSecurityCenterHandlerFromConfigWithManager(context.Background(), gcp.Singleton(),
			[]byte(decCreds), conf.GetWifEnabled())
	} else {
		handler, err = gcpUtils.CreateSecurityCenterHandlerFromConfig(context.Background(), []byte(decCreds), conf.GetWifEnabled())
	}
	if err != nil {
		return nil, errors.Wrap(err, "could not create security center client handler")
	}

	return &cscc{
		Notifier: protoNotifier,
		client:   handler,
		config: &config{
			ServiceAccount: decCreds,
			SourceID:       conf.SourceId,
		},
	}, nil
}

// AlertNotify takes in an alert and generates the notification.
func (c *cscc) AlertNotify(ctx context.Context, alert *storage.Alert) error {
	findingID, finding, err := c.initFinding(ctx, alert, clusterDatastore.Singleton())
	if err != nil {
		return err
	}

	client, done := c.client.GetClient()
	defer done()

	_, err = client.CreateFinding(ctx, &securitycenterpb.CreateFindingRequest{
		Parent:    finding.GetParent(),
		FindingId: findingID,
		Finding:   finding,
	}, gax.WithTimeout(env.CSCCTimeout.DurationSetting()), gax.WithRetry(func() gax.Retryer {
		// This is mimicking the previous behavior of notifiers.CreateError.
		return gax.OnHTTPCodes(gax.Backoff{}, http.StatusServiceUnavailable)
	}))
	if err != nil {
		log.Errorw("failed to create finding",
			logging.Err(err),
			logging.ErrCode(codes.CloudPlatformGeneric),
			logging.NotifierName(c.Notifier.GetName()),
		)
	}

	return err
}

func (c *cscc) Close(_ context.Context) error {
	client, done := c.client.GetClient()
	defer done()
	return client.Close()
}

func (c *cscc) ProtoNotifier() *storage.Notifier {
	return c.Notifier
}

func (c *cscc) Test(context.Context) error {
	return errors.New("Test is not yet implemented for Cloud SCC")
}

func (c *cscc) getCluster(id string, clusterDatastore clusterDatastore.DataStore) (*storage.Cluster, error) {
	cluster, exists, err := clusterDatastore.GetCluster(clusterForAlertContext, id)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, fmt.Errorf("Could not retrieve cluster %q because it does not exist", id)
	}
	providerMetadata := cluster.GetStatus().GetProviderMetadata()
	if providerMetadata.GetGoogle().GetProject() == "" {
		return nil, fmt.Errorf("Could not find Google project for cluster %q", id)
	}
	if providerMetadata.GetGoogle().GetClusterName() == "" {
		return nil, fmt.Errorf("Could not find Google cluster name for cluster %q", id)
	}
	if providerMetadata.GetZone() == "" {
		return nil, fmt.Errorf("Could not find Google zone for cluster %q", id)
	}
	return cluster, nil
}

// initFinding takes in an alert and generates the finding.
func (c *cscc) initFinding(_ context.Context, alert *storage.Alert,
	clusterDatastore clusterDatastore.DataStore) (string, *securitycenterpb.Finding, error) {
	if alert.GetImage() != nil {
		return "", nil, errors.New("CSCC integration can only handle alerts for deployments and resources")
	}

	cluster, err := c.getCluster(alert.GetDeployment().GetClusterId(), clusterDatastore)
	if err != nil {
		return "", nil, err
	}
	providerMetadata := cluster.GetStatus().GetProviderMetadata()

	return convertAlertToFinding(alert, c.config.SourceID, c.Notifier.UiEndpoint, providerMetadata)
}
