package settingswatch

import (
	"time"

	"github.com/cenkalti/backoff/v3"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/internalapi/sensor"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/sensor/admission-control/common"
	"github.com/stackrox/rox/sensor/admission-control/manager"
	"google.golang.org/grpc"
)

// WatchSensorSettingsPush watches for sensor pushes of admission control settings, and forwards them.
func WatchSensorSettingsPush(mgr manager.Manager, cc *grpc.ClientConn) {
	w := &sensorPushWatch{
		ctx:               mgr.Stopped(),
		mgmtServiceClient: sensor.NewAdmissionControlManagementServiceClient(cc),
		outC:              mgr.SettingsUpdateC(),
		sensorConnStatus:  mgr.SensorConnStatusFlag(),
	}
	go w.run()
}

type sensorPushWatch struct {
	ctx              concurrency.ErrorWaitable
	outC             chan<- *sensor.AdmissionControlSettings
	sensorConnStatus *concurrency.Flag

	mgmtServiceClient sensor.AdmissionControlManagementServiceClient
}

func (w *sensorPushWatch) run() {
	w.sensorConnStatus.Set(false)

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
		case w.outC <- m.SettingsPush:
			log.Infof("Received and propagated updated admission controller settings via sensor push, timestamp: %v", m.SettingsPush.GetTimestamp())
		}
	default:
		log.Warnf("Received message of unknown type %T from sensor, not sure what to do with it ...", m)
	}
	return nil
}
