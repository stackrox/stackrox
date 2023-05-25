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
	"github.com/stackrox/rox/sensor/common/registry"
	"github.com/stackrox/rox/sensor/common/scan"
	"google.golang.org/grpc"
)

var (
	log = logging.LoggerForModule()

	scanTimeout         = 6 * time.Minute
	statusUpdateTimeout = 10 * time.Second
)

// Handler is responsible for processing delegated
// registry config updates from central.
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
	if !env.LocalImageScanningEnabled.BooleanSetting() {
		// do not advertise the capability if local scanning is disabled
		return nil
	}

	return []centralsensor.SensorCapability{centralsensor.DelegatedRegistryCap}
}

func (d *delegatedRegistryImpl) Notify(_ common.SensorComponentEvent) {}

func (d *delegatedRegistryImpl) ProcessMessage(msg *central.MsgToSensor) error {
	if !env.LocalImageScanningEnabled.BooleanSetting() {
		// ignore all messages if local scanning is disabled
		return nil
	}

	switch {
	case msg.GetDelegatedRegistryConfig() != nil:
		return d.processUpdatedDelegatedRegistryConfig(msg.GetDelegatedRegistryConfig())
	case msg.GetScanImage() != nil:
		return d.processScanImage(msg.GetScanImage())
	}

	return nil
}

func (d *delegatedRegistryImpl) ResponsesC() <-chan *central.MsgFromSensor {
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
		log.Debugf("Stored updated delegated registry config: %q", config)
	}
	return nil
}

func (d *delegatedRegistryImpl) processScanImage(scanReq *central.ScanImage) error {
	select {
	case <-d.stopSig.Done():
		return errors.New("could not process scan image request, stop requested")
	default:
		log.Debugf("Received scan request: %q", scanReq)

		// Spawn a goroutine so that this handler doesn't block other messages from being processed
		// while waiting for scan to complete
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

	// Execute the scan
	_, err = d.localScan.EnrichLocalImageInNamespace(ctx, d.imageSvc, ci, "", scanReq.GetRequestId(), scanReq.GetForce())
	if errors.Is(err, scan.ErrEnrichNotStarted) {
		// This error indicates enrichment never started and therefore a message will
		// not be sent to central for this request id, so send one now to be a good
		// citizen to the waiting goroutine in central
		d.sendScanStatusUpdate(scanReq, err)
	}
}

func (d *delegatedRegistryImpl) sendScanStatusUpdate(scanReq *central.ScanImage, enrichErr error) {
	ctx, cancel := context.WithTimeout(context.Background(), statusUpdateTimeout)
	defer cancel()
	_, err := d.imageSvc.UpdateLocalScanStatusInternal(ctx, &v1.UpdateLocalScanStatusInternalRequest{
		RequestId: scanReq.GetRequestId(),
		Error:     enrichErr.Error(),
	})
	if err != nil {
		log.Warnf("Error updating local scan status: %v", err)
	}
}

func (d *delegatedRegistryImpl) SetCentralGRPCClient(cc grpc.ClientConnInterface) {
	d.imageSvc = v1.NewImageServiceClient(cc)
	log.Debugf("Received central GRPC client connection")
}
