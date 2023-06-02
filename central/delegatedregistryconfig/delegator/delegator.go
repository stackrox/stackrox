package delegator

import (
	"context"
	"strings"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/delegatedregistryconfig/datastore"
	deleConnection "github.com/stackrox/rox/central/delegatedregistryconfig/util/connection"
	"github.com/stackrox/rox/central/sensor/service/connection"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/urlfmt"
	"github.com/stackrox/rox/pkg/waiter"
)

var (
	log = logging.LoggerForModule()
)

// New creates a new delegator.
func New(deleRegConfigDS datastore.DataStore, connManager connection.Manager, scanWaiterManager waiter.Manager[*storage.Image]) *delegatorImpl {
	return &delegatorImpl{
		deleRegConfigDS:   deleRegConfigDS,
		connManager:       connManager,
		scanWaiterManager: scanWaiterManager,
	}
}

type delegatorImpl struct {
	// deleRegConfigDS for pulling the current delegated registry config.
	deleRegConfigDS datastore.DataStore

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

	if err := d.validateCluster(clusterID); err != nil {
		return "", true, err
	}

	return clusterID, true, nil
}

// DelegateScanImage sends a scan request to the provided cluster.
func (d *delegatorImpl) DelegateScanImage(ctx context.Context, imgName *storage.ImageName, clusterID string, force bool) (*storage.Image, error) {
	if clusterID == "" {
		return nil, errors.New("missing cluster id")
	}

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
			},
		},
	}

	err = d.connManager.SendMessage(clusterID, msg)
	if err != nil {
		return nil, err
	}

	log.Infof("Sent scan request %q to cluster %q for %q", w.ID(), clusterID, imgName.GetFullName())

	image, err := w.Wait(ctx)
	if err != nil {
		return nil, errors.Wrapf(err, "error delegating scan to cluster %q for %q", clusterID, image.GetName().GetFullName())
	}

	log.Debugf("Scan response received for %q and image %q", w.ID(), imgName.GetFullName())

	return image, nil
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

func (d *delegatorImpl) validateCluster(clusterID string) error {
	if clusterID == "" {
		return errors.New("no ad-hoc cluster specified in delegated registry config")
	}

	conn := d.connManager.GetConnection(clusterID)
	if conn == nil {
		return errors.Errorf("no connection to %q", clusterID)
	}

	if !deleConnection.ValidForDelegation(conn) {
		return errors.Errorf("cluster %q does not support delegated scanning", clusterID)
	}

	return nil
}
