package delegator

import (
	"context"
	"strings"
	"time"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/delegatedregistryconfig/datastore"
	deleConnection "github.com/stackrox/rox/central/delegatedregistryconfig/util/connection"
	centralMetrics "github.com/stackrox/rox/central/metrics"
	"github.com/stackrox/rox/central/role/sachelper"
	"github.com/stackrox/rox/central/sensor/service/connection"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/images/utils"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/urlfmt"
	"github.com/stackrox/rox/pkg/waiter"
)

var (
	log = logging.LoggerForModule()

	inferNamespacePermissions = []string{"Image"}
)

// New creates a new delegator.
func New(deleRegConfigDS datastore.DataStore, connManager connection.Manager, scanWaiterManager waiter.Manager[*storage.Image], namespaceSACHelper sachelper.ClusterNamespaceSacHelper) *delegatorImpl {
	return &delegatorImpl{
		deleRegConfigDS:    deleRegConfigDS,
		connManager:        connManager,
		scanWaiterManager:  scanWaiterManager,
		namespaceSACHelper: namespaceSACHelper,
	}
}

type delegatorImpl struct {
	// deleRegConfigDS for pulling the current delegated registry config.
	deleRegConfigDS datastore.DataStore

	// namespaceSACHelper for confirming namespace exists and user has access.
	namespaceSACHelper sachelper.ClusterNamespaceSacHelper

	// connManager for sending scan requests to secured clusters and ensuring
	// clusters are valid for delegation.
	connManager connection.Manager

	// scanWaiterManager creates waiters that wait for async scan responses.
	scanWaiterManager waiter.Manager[*storage.Image]
}

// GetDelegateClusterID returns the cluster id that should enrich this image (if any) and
// true if enrichment should be delegated to a secured cluster, false otherwise.
func (d *delegatorImpl) GetDelegateClusterID(ctx context.Context, imgName *storage.ImageName) (string, bool, error) {
	config, exists, err := d.deleRegConfigDS.GetConfig(ctx)
	if err != nil || !exists {
		return "", false, err
	}

	shouldDelegate, clusterID := d.shouldDelegate(imgName, config)
	if !shouldDelegate {
		return "", false, nil
	}

	if clusterID == "" {
		return "", true, errox.InvalidArgs.New("no ad-hoc cluster ID specified in the delegated scanning config")
	}

	if err := d.ValidateCluster(clusterID); err != nil {
		return "", true, err
	}

	return clusterID, true, nil
}

// DelegateScanImage sends a scan request to the provided cluster.
func (d *delegatorImpl) DelegateScanImage(ctx context.Context, imgName *storage.ImageName, clusterID string, force bool) (*storage.Image, error) {
	if clusterID == "" {
		return nil, errors.New("missing cluster id")
	}

	namespace := d.inferNamespace(ctx, imgName, clusterID)

	w, err := d.scanWaiterManager.NewWaiter()
	if err != nil {
		return nil, err
	}
	defer w.Close()

	msg := &central.MsgToSensor{
		Msg: &central.MsgToSensor_ScanImage{
			ScanImage: &central.ScanImage{
				RequestId: w.ID(),
				ImageName: imgName.GetFullName(),
				Force:     force,
				Namespace: namespace,
			},
		},
	}

	err = d.connManager.SendMessage(clusterID, msg)
	if err != nil {
		return nil, err
	}

	log.Infof("Sent scan request %q to cluster %q for %q with inferred namespace %q", w.ID(), clusterID, imgName.GetFullName(), namespace)

	image, err := w.Wait(ctx)
	if err != nil {
		return nil, errors.Wrapf(err, "error delegating scan to cluster %q for %q", clusterID, image.GetName().GetFullName())
	}

	log.Debugf("Scan response received for %q and image %q", w.ID(), imgName.GetFullName())

	return image, nil
}

// inferNamespace attempts to guess a namespace based on an image path, which is the convention used by images from
// the OCP integrated registry. A namespace is returned only if it exists and the user has access. This inference
// would be more accurate if done in the secured cluster however user access cannot be checked there. The namespace
// is used by Sensor to pull additional secrets for authenticating to the image registry.
func (d *delegatorImpl) inferNamespace(ctx context.Context, imgName *storage.ImageName, clusterID string) string {
	// Extract namespace from image path following OCP integrated registry convention.
	namespace := utils.ExtractOpenShiftProject(imgName)

	defer centralMetrics.SetFunctionSegmentDuration(time.Now(), "ScanDelegatorInferNamespace")

	namespaces, err := d.namespaceSACHelper.GetNamespacesForClusterAndPermissions(ctx, clusterID, inferNamespacePermissions)
	if err != nil {
		log.Warnf("Skipping namespace inference for %q (%s) and cluster %q due to error: %v", imgName.GetFullName(), namespace, clusterID, err)
		return ""
	}

	for _, ns := range namespaces {
		if ns.GetName() == namespace {
			return namespace
		}
	}

	return ""
}

func (d *delegatorImpl) shouldDelegate(imgName *storage.ImageName, config *storage.DelegatedRegistryConfig) (bool, string) {
	if config.GetEnabledFor() == storage.DelegatedRegistryConfig_NONE {
		return false, ""
	}

	clusterID := config.GetDefaultClusterId()
	imageFullName := urlfmt.TrimHTTPPrefixes(imgName.GetFullName())

	for _, reg := range config.GetRegistries() {
		regPath := urlfmt.TrimHTTPPrefixes(reg.GetPath())

		if strings.HasPrefix(imageFullName, regPath) {
			if reg.GetClusterId() != "" {
				return true, reg.GetClusterId()
			}

			return true, clusterID
		}
	}

	if config.GetEnabledFor() == storage.DelegatedRegistryConfig_ALL {
		return true, clusterID
	}

	return false, ""
}

// ValidateCluster returns nil if a cluster is a valid target for delegation, otherwise returns an error.
func (d *delegatorImpl) ValidateCluster(clusterID string) error {
	conn := d.connManager.GetConnection(clusterID)
	if conn == nil {
		return errors.Errorf("no connection to %q", clusterID)
	}

	if !deleConnection.ValidForDelegation(conn) {
		return errors.Errorf("cluster %q does not support delegated scanning", clusterID)
	}

	return nil
}
