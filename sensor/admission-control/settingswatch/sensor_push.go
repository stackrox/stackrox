package settingswatch

import (
	"time"

	"github.com/cenkalti/backoff/v3"
	"github.com/pkg/errors"
	"github.com/stackrox/stackrox/generated/internalapi/sensor"
	"github.com/stackrox/stackrox/pkg/concurrency"
	"github.com/stackrox/stackrox/sensor/admission-control/common"
	"github.com/stackrox/stackrox/sensor/admission-control/manager"
	"google.golang.org/grpc"
)

// WatchSensorMessagePush watches for sensor pushes, and forwards them.
func WatchSensorMessagePush(mgr manager.Manager, cc *grpc.ClientConn) {
	w := &sensorPushWatch{
		ctx:                   mgr.Stopped(),
		mgmtServiceClient:     sensor.NewAdmissionControlManagementServiceClient(cc),
		settingsOutC:          mgr.SettingsUpdateC(),
		updateResourceReqOutC: mgr.ResourceUpdatesC(),
		sensorConnStatus:      mgr.SensorConnStatusFlag(),
		initialResourceSync:   mgr.InitialResourceSyncSig(),
	}
	go w.run()
}

type sensorPushWatch struct {
	ctx                   concurrency.ErrorWaitable
	settingsOutC          chan<- *sensor.AdmissionControlSettings
	updateResourceReqOutC chan<- *sensor.AdmCtrlUpdateResourceRequest

	sensorConnStatus    *concurrency.Flag
	initialResourceSync *concurrency.Signal

	mgmtServiceClient sensor.AdmissionControlManagementServiceClient
}

func (w *sensorPushWatch) run() {
	w.sensorConnStatus.Set(false)
	w.initialResourceSync.Reset()

	eb := common.NewBackOffForSensorConn()
	tC := time.After(0)

	ctx := concurrency.AsContext(w.ctx)

	for {
		select {
		case <-tC:
			communicateStart := time.Now()
			stream, err := w.mgmtServiceClient.Communicate(ctx)
			if err == nil {
				err = w.runWithStream(stream)
			}

			if time.Since(communicateStart) > common.BackoffResetThreshold {
				eb.Reset()
			}

			nextBackOff := eb.NextBackOff()
			if nextBackOff == backoff.Stop {
				log.Errorf("exceeded the maximum elapsed time %v to reconnect to Sensor", eb.MaxElapsedTime)
				return
			}
			log.Warnf("Communication to sensor failed: %v. Retrying in %v", err, nextBackOff)
			tC = time.After(nextBackOff)

		case <-w.ctx.Done():
			return
		}
	}
}

func (w *sensorPushWatch) runWithStream(stream sensor.AdmissionControlManagementService_CommunicateClient) error {
	w.sensorConnStatus.Set(true)
	defer w.sensorConnStatus.Set(false)

	for {
		msg, err := stream.Recv()
		if err != nil {
			return errors.Wrap(err, "receiving message from sensor")
		}

		if err := w.dispatchMsg(msg); err != nil {
			return err
		}
	}
}

func (w *sensorPushWatch) dispatchMsg(msg *sensor.MsgToAdmissionControl) error {
	switch m := msg.Msg.(type) {
	case *sensor.MsgToAdmissionControl_SettingsPush:
		select {
		case <-w.ctx.Done():
			return w.ctx.Err()
		case w.settingsOutC <- m.SettingsPush:
			log.Infof("Received and propagated updated admission controller settings via sensor push, timestamp: %v", m.SettingsPush.GetTimestamp())
		}
	case *sensor.MsgToAdmissionControl_UpdateResourceRequest:
		select {
		case <-w.ctx.Done():
			return w.ctx.Err()
		case w.updateResourceReqOutC <- m.UpdateResourceRequest:
		}
	default:
		log.Warnf("Received message of unknown type %T from sensor, not sure what to do with it ...", m)
	}
	return nil
}
