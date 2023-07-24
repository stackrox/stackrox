package delegatedregistry

import (
	"context"
	"errors"
	"time"

	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/pkg/centralsensor"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/images/utils"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/sensor/common"
	"github.com/stackrox/rox/sensor/common/message"
	"github.com/stackrox/rox/sensor/common/registry"
	"github.com/stackrox/rox/sensor/common/scan"
	"google.golang.org/grpc"
)

var (
	log = logging.LoggerForModule()

	scanTimeout         = env.ScanTimeout.DurationSetting()
	statusUpdateTimeout = 10 * time.Second

	enabled = env.LocalImageScanningEnabled.BooleanSetting() && !env.DelegatedScanningDisabled.BooleanSetting()
)

// Handler is responsible for processing delegated registry related requests
// from Central.
type Handler interface {
	common.SensorComponent
}

type delegatedRegistryImpl struct {
	registryStore *registry.Store
	stopSig       concurrency.Signal
	localScan     *scan.LocalScan
	imageSvc      v1.ImageServiceClient
}

// NewHandler returns a new instance of Handler.
func NewHandler(registryStore *registry.Store, localScan *scan.LocalScan) Handler {
	return &delegatedRegistryImpl{
		registryStore: registryStore,
		stopSig:       concurrency.NewSignal(),
		localScan:     localScan,
	}
}

func (d *delegatedRegistryImpl) Capabilities() []centralsensor.SensorCapability {
	if !enabled {
		return nil
	}

	return []centralsensor.SensorCapability{centralsensor.DelegatedRegistryCap}
}

func (d *delegatedRegistryImpl) Notify(_ common.SensorComponentEvent) {}

func (d *delegatedRegistryImpl) ProcessMessage(msg *central.MsgToSensor) error {
	if !enabled {
		return nil
	}

	switch {
	case msg.GetDelegatedRegistryConfig() != nil:
		return d.processUpdatedDelegatedRegistryConfig(msg.GetDelegatedRegistryConfig())
	case msg.GetScanImage() != nil:
		return d.processScanImage(msg.GetScanImage())
	case msg.GetImageIntegrations() != nil:
		return d.processImageIntegrations(msg.GetImageIntegrations())
	}

	return nil
}

func (d *delegatedRegistryImpl) ResponsesC() <-chan *message.ExpiringMessage {
	return nil
}

func (d *delegatedRegistryImpl) Start() error {
	return nil
}

func (d *delegatedRegistryImpl) Stop(_ error) {
	d.stopSig.Signal()
}

func (d *delegatedRegistryImpl) processUpdatedDelegatedRegistryConfig(config *central.DelegatedRegistryConfig) error {
	select {
	case <-d.stopSig.Done():
		return errors.New("could not process updated delegated registry config, stop requested")
	default:
		d.registryStore.SetDelegatedRegistryConfig(config)
		log.Infof("Upserted delegated registry config: %q", config)
	}
	return nil
}

func (d *delegatedRegistryImpl) processScanImage(scanReq *central.ScanImage) error {
	select {
	case <-d.stopSig.Done():
		return errors.New("could not process scan image request, stop requested")
	default:
		log.Infof("Received scan request: %q", scanReq)

		// Spawn a goroutine so that this handler doesn't block other messages from being processed
		// while waiting for scan to complete.
		go d.executeScan(scanReq)
	}

	return nil
}

func (d *delegatedRegistryImpl) executeScan(scanReq *central.ScanImage) {
	ci, err := utils.GenerateImageFromString(scanReq.GetImageName())
	if err != nil {
		d.sendScanStatusUpdate(scanReq, err)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), scanTimeout)
	defer cancel()

	// Execute the scan, ignore returned image because will be sent to Central during enrichment.
	_, err = d.localScan.EnrichLocalImageInNamespace(ctx, d.imageSvc, ci, "", scanReq.GetRequestId(), scanReq.GetForce())
	if err != nil {
		log.Errorf("Scan failed for req %q image %q: %v", scanReq.GetRequestId(), ci.GetName().GetFullName(), err)

		if errors.Is(err, scan.ErrEnrichNotStarted) {
			// This error indicates enrichment never started and therefore a message will
			// not be sent to Central for this request.

			// Send an update now to be a good citizen so that the waiting goroutine in
			// Central is not waiting for the full timeout and the failure reason is
			// communicated.
			d.sendScanStatusUpdate(scanReq, err)
		}
	}
}

func (d *delegatedRegistryImpl) sendScanStatusUpdate(scanReq *central.ScanImage, enrichErr error) {
	ctx, cancel := context.WithTimeout(context.Background(), statusUpdateTimeout)
	defer cancel()

	req := &v1.UpdateLocalScanStatusInternalRequest{
		RequestId: scanReq.GetRequestId(),
		Error:     enrichErr.Error(),
	}

	_, err := d.imageSvc.UpdateLocalScanStatusInternal(ctx, req)
	if err != nil {
		log.Warnf("Error sending local scan status update for req %q image %q: %v", scanReq.GetRequestId(), scanReq.GetImageName(), err)
	}
}

func (d *delegatedRegistryImpl) SetCentralGRPCClient(cc grpc.ClientConnInterface) {
	d.imageSvc = v1.NewImageServiceClient(cc)
	log.Debugf("Received central GRPC client connection")
}

func (d *delegatedRegistryImpl) processImageIntegrations(iiReq *central.ImageIntegrations) error {
	select {
	case <-d.stopSig.Done():
		return errors.New("could not process updated image integrations, stop requested")
	default:
		log.Infof("Received %d updated and %d deleted image integrations", len(iiReq.GetUpdatedIntegrations()), len(iiReq.GetDeletedIntegrationIds()))

		d.registryStore.UpsertCentralRegistryIntegrations(iiReq.GetUpdatedIntegrations())
		d.registryStore.DeleteCentralRegistryIntegrations(iiReq.GetDeletedIntegrationIds())
	}
	return nil
}
