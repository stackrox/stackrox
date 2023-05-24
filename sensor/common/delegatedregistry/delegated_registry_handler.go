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

	scanTimeout = 6 * time.Minute
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
		// TODO: Change scan image so that it doesn't hold up processing other receivers, consider spawning a go routine
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

		// TODO: ensure these timeouts OK?
		// TODO: perhaps spawn a go-routine so that do not hold up sensor processing other msgs
		ci, err := utils.GenerateImageFromString(scanReq.GetImageName())
		if err != nil {
			return err
		}

		ctx, cancel := context.WithTimeout(context.Background(), scanTimeout)
		defer cancel()

		// TODO: create another method or change this method so that does not 'include' namespace
		_, err = d.localScan.EnrichLocalImageInNamespace(ctx, d.imageSvc, ci, "", scanReq.GetRequestId(), scanReq.GetForce())
		if errors.Is(err, scan.ErrEnrichNotStarted) {
			d.imageSvc.UpdateScanImageStatusInternal(ctx, &v1.UpdateScanImageStatusInternalRequest{
				RequestId: scanReq.GetRequestId(),
				Error:     err.Error(),
			})
		}
	}

	return nil
}

func (d *delegatedRegistryImpl) SetCentralGRPCClient(cc grpc.ClientConnInterface) {
	d.imageSvc = v1.NewImageServiceClient(cc)
	log.Debugf("Received central GRPC client connection")
}
