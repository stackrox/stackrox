package delegator

import (
	"context"
	"fmt"
	"strings"

	"github.com/stackrox/rox/central/delegatedregistryconfig/datastore"
	"github.com/stackrox/rox/central/sensor/service/connection"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/centralsensor"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/urlfmt"
	"github.com/stackrox/rox/pkg/waiter"
)

var (
	log = logging.LoggerForModule()
)

func New(deleRegConfigDS datastore.DataStore, connManager connection.Manager, scanWaiterManager waiter.Manager[*storage.Image]) *delegatorImpl {
	return &delegatorImpl{
		deleRegConfigDS:   deleRegConfigDS,
		connManager:       connManager,
		scanWaiterManager: scanWaiterManager,
	}
}

type delegatorImpl struct {
	deleRegConfigDS   datastore.DataStore
	connManager       connection.Manager
	scanWaiterManager waiter.Manager[*storage.Image]
}

func (d *delegatorImpl) DelegateEnrichImage(ctx context.Context, image *storage.Image) (bool, error) {
	config, err := d.getConfig(ctx)
	if err != nil {
		return false, err
	}
	log.Debugf("Got delegated registry config %q", config)

	should, clusterID := d.shouldDelegate(image, config)
	if !should {
		return false, nil
	}
	log.Debugf("Should delegate %v to clsuter %q", should, clusterID)

	// if got here enrichment should be delegated to a secured cluster
	err = d.validateCluster(clusterID)
	if err != nil {
		return true, err
	}

	// cluster valid, create the waiter and send the scan request
	w, err := d.scanWaiterManager.NewWaiter()
	if err != nil {
		return true, err
	}
	log.Debugf("Scan waiter created with id %q", w.ID())

	msg := &central.MsgToSensor{
		Msg: &central.MsgToSensor_ScanImage{
			ScanImage: &central.ScanImage{
				RequestId: w.ID(),
				ImageName: image.GetName().GetFullName(),
			},
		},
	}

	err = d.connManager.SendMessage(clusterID, msg)
	if err != nil {
		w.Close()
		return true, err
	}

	log.Debugf("Successful sent scan request to cluster %q", clusterID)

	img, err := w.Wait(ctx)
	if err != nil {
		return true, err
	}

	log.Debugf("Scan response received for image %q", img)
	// assign the values from the returned image to this image
	*image = *img

	return true, nil
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
	clusterID := config.GetDefaultClusterId()

	if config.GetEnabledFor() == storage.DelegatedRegistryConfig_ALL {
		should = true
	}

	imageFullName := urlfmt.TrimHTTPPrefixes(image.GetName().GetFullName())
	for _, reg := range config.GetRegistries() {
		regPath := urlfmt.TrimHTTPPrefixes(reg.GetRegistryPath())
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
		return fmt.Errorf("no ad-hoc cluster specified in delegation config")
	}

	conn := d.connManager.GetConnection(clusterID)

	if conn == nil {
		return fmt.Errorf("no connection to %q", clusterID)
	}

	if !conn.HasCapability(centralsensor.DelegatedRegistryCap) {
		return fmt.Errorf("cluster %q does not support delegated scanning", clusterID)
	}

	return nil
}
