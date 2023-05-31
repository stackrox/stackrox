package delegator

import (
	"context"
	"errors"
	"fmt"
	"strings"

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
func (d *delegatorImpl) GetDelegateClusterID(ctx context.Context, image *storage.Image) (string, bool, error) {
	config, err := d.getConfig(ctx)
	if err != nil {
		return "", false, err
	}

	shouldDelegate, clusterID := d.shouldDelegate(image, config)
	if !shouldDelegate {
		return "", false, nil
	}

	err = d.validateCluster(clusterID)
	return clusterID, true, err
}

// DelegateEnrichImage sends an enrichment request to the provided cluster.
func (d *delegatorImpl) DelegateEnrichImage(ctx context.Context, image *storage.Image, clusterID string, force bool) error {
	if clusterID == "" {
		return errors.New("missing cluster id")
	}

	w, err := d.scanWaiterManager.NewWaiter()
	if err != nil {
		return err
	}

	msg := &central.MsgToSensor{
		Msg: &central.MsgToSensor_ScanImage{
			ScanImage: &central.ScanImage{
				RequestId: w.ID(),
				ImageName: image.GetName().GetFullName(),
				Force:     force,
			},
		},
	}

	err = d.connManager.SendMessage(clusterID, msg)
	if err != nil {
		w.Close()
		return err
	}

	log.Infof("Sent scan request %q to cluster %q for %q", w.ID(), clusterID, image.GetName().GetFullName())

	img, err := w.Wait(ctx)
	if err != nil {
		return fmt.Errorf("error delegating scan to cluster %q for %q: %w", clusterID, image.GetName().GetFullName(), err)
	}

	log.Debugf("Scan response received for %q and image %q", w.ID(), img.GetName().GetFullName())

	// Copy the fields from img into image, callers expecting image to be modified in place.
	*image = *img

	return nil
}

func (d *delegatorImpl) getConfig(ctx context.Context) (*storage.DelegatedRegistryConfig, error) {
	config, _, err := d.deleRegConfigDS.GetConfig(ctx)
	return config, err
}

func (d *delegatorImpl) shouldDelegate(image *storage.Image, config *storage.DelegatedRegistryConfig) (bool, string) {
	if config == nil || config.GetEnabledFor() == storage.DelegatedRegistryConfig_NONE {
		return false, ""
	}

	var should bool
	if config.GetEnabledFor() == storage.DelegatedRegistryConfig_ALL {
		should = true
	}

	clusterID := config.GetDefaultClusterId()
	imageFullName := urlfmt.TrimHTTPPrefixes(image.GetName().GetFullName())
	for _, reg := range config.GetRegistries() {
		regPath := urlfmt.TrimHTTPPrefixes(reg.GetPath())
		if strings.HasPrefix(imageFullName, regPath) {
			should = true
			if reg.GetClusterId() != "" {
				clusterID = reg.GetClusterId()
			}
		}
	}

	return should, clusterID
}

func (d *delegatorImpl) validateCluster(clusterID string) error {
	if clusterID == "" {
		return errors.New("no ad-hoc cluster specified in delegated registry config")
	}

	conn := d.connManager.GetConnection(clusterID)
	if conn == nil {
		return fmt.Errorf("no connection to %q", clusterID)
	}

	if !deleConnection.ValidForDelegation(conn) {
		return fmt.Errorf("cluster %q does not support delegated scanning", clusterID)
	}

	return nil
}
