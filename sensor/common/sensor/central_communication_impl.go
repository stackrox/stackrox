package sensor

import (
	"context"
	"strconv"
	"time"

	metautils "github.com/grpc-ecosystem/go-grpc-middleware/v2/metadata"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/pkg/booleanpolicy/policyversion"
	"github.com/stackrox/rox/pkg/centralsensor"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/deduperkey"
	"github.com/stackrox/rox/pkg/safe"
	"github.com/stackrox/rox/pkg/set"
	"github.com/stackrox/rox/pkg/sliceutils"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/pkg/version"
	"github.com/stackrox/rox/sensor/common"
	"github.com/stackrox/rox/sensor/common/centralcaps"
	"github.com/stackrox/rox/sensor/common/centralid"
	"github.com/stackrox/rox/sensor/common/certdistribution"
	"github.com/stackrox/rox/sensor/common/clusterid"
	"github.com/stackrox/rox/sensor/common/config"
	"github.com/stackrox/rox/sensor/common/detector"
	"github.com/stackrox/rox/sensor/common/managedcentral"
	"github.com/stackrox/rox/sensor/common/sensor/helmconfig"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/encoding/gzip"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

// sensor implements the Sensor interface by sending inputs to central,
// and providing the output from central asynchronously.
type centralCommunicationImpl struct {
	receiver            CentralReceiver
	sender              CentralSender
	components          []common.SensorComponent
	clientReconcile     bool
	initialDeduperState map[deduperkey.Key]uint64
	syncTimeout         time.Duration

	stopper concurrency.Stopper

	// allFinished waits until both receiver and sender fully stopped before cleaning up the stream.
	allFinished *sync.WaitGroup

	isReconnect bool
}

var (
	errForcedConnectionRestart       = errors.New("forced connection restart")
	errCantReconcile                 = errors.New("unable to reconcile")
	errLargePayload                  = errors.Wrap(errCantReconcile, "deduper payload too large")
	errTimeoutWaitingForDeduperState = errors.Wrap(errCantReconcile, "timeout reached while waiting for the DeduperState")
	errIncorrectDeduperStateOrder    = errors.Wrap(errCantReconcile, "central sent incorrect order of chunks of the deduper state")
	errIncorrectEventOrder           = errors.Wrap(errCantReconcile, "central sent incorrect order of events")
)

func (s *centralCommunicationImpl) Start(client central.SensorServiceClient, centralReachable *concurrency.Flag, syncDone *concurrency.Signal, configHandler config.Handler, detector detector.Detector) {
	go s.sendEvents(client, centralReachable, syncDone, configHandler, detector, s.receiver.Stop, s.sender.Stop)
}

func (s *centralCommunicationImpl) Stop(err error) {
	if err != nil {
		if errors.Is(err, errForcedConnectionRestart) {
			log.Infof("Connection restart requested: %v", err)
		} else {
			log.Errorf("Stopping connection due to error: %v", err)
		}
	}
	s.stopper.Client().Stop()
}

func (s *centralCommunicationImpl) Stopped() concurrency.ReadOnlyErrorSignal {
	return s.stopper.Client().Stopped()
}

func isUnimplemented(err error) bool {
	spb, ok := status.FromError(err)
	if spb == nil || !ok {
		return false
	}
	return spb.Code() == codes.Unimplemented
}

func communicateWithAutoSensedEncoding(ctx context.Context, client central.SensorServiceClient) (central.SensorService_CommunicateClient, error) {
	opts := []grpc.CallOption{grpc.UseCompressor(gzip.Name)}

	for {
		stream, err := client.Communicate(ctx, opts...)
		if err != nil {
			if isUnimplemented(err) && len(opts) > 0 {
				opts = nil
				continue
			}
			return nil, errors.Wrap(err, "opening stream")
		}

		_, err = stream.Header()
		if err != nil {
			if isUnimplemented(err) && len(opts) > 0 {
				opts = nil
				continue
			}
			return nil, errors.Wrap(err, "receiving initial metadata")
		}

		return stream, nil
	}
}

func (s *centralCommunicationImpl) getSensorState() central.SensorHello_SensorState {
	if s.isReconnect {
		return central.SensorHello_RECONNECT
	}
	return central.SensorHello_STARTUP
}

func (s *centralCommunicationImpl) sendEvents(client central.SensorServiceClient, centralReachable *concurrency.Flag, syncDone *concurrency.Signal, configHandler config.Handler, detector detector.Detector, onStops ...func(error)) {
	var stream central.SensorService_CommunicateClient
	defer func() {
		s.stopper.Flow().ReportStopped()
		runAll(s.stopper.Client().Stopped().Err(), onStops...)
		s.allFinished.Wait()
		if stream != nil {
			if err := stream.CloseSend(); err != nil {
				log.Errorf("Failed to close stream cleanly: %v", err)
			}
		}
	}()

	// Start the stream client.
	///////////////////////////

	// Prepare the `SensorHello` message. This message informs Central about who is talking to it, and announces
	// the capabilities/features supported by this sensor.
	// While the message is only sent after the stream is established, it is also used to populate the legacy,
	// header metadata-based self-identification protocol, which needs to happen prior to making the streaming RPC
	// call. That's why we create it here and not in the `initialSync` method below.
	sensorHello := &central.SensorHello{
		SensorVersion:            version.GetMainVersion(),
		PolicyVersion:            policyversion.CurrentVersion().String(),
		DeploymentIdentification: configHandler.GetDeploymentIdentification(),
		SensorState:              s.getSensorState(),
		RequestDeduperState:      s.clientReconcile,
	}

	capsSet := set.NewSet[centralsensor.SensorCapability]()
	for _, component := range s.components {
		capsSet.AddAll(component.Capabilities()...)
	}
	capsSet.Add(centralsensor.SendDeduperStateOnReconnect)
	sensorHello.Capabilities = sliceutils.StringSlice(capsSet.AsSlice()...)

	// Inject desired Helm configuration, if any.
	if helmManagedCfg := configHandler.GetHelmManagedConfig(); helmManagedCfg != nil && helmManagedCfg.GetClusterId() == "" {
		cachedClusterID, err := helmconfig.LoadCachedClusterID()
		if err != nil {
			log.Warnf("Failed to load cached cluster ID: %s", err)
		} else if cachedClusterID != "" {
			helmManagedCfg = helmManagedCfg.CloneVT()
			helmManagedCfg.ClusterId = cachedClusterID
			log.Infof("Re-using cluster ID %s of previous run. If you see the connection to central failing, re-apply a new Helm configuration via 'helm upgrade', or delete the sensor pod.", cachedClusterID)
		}

		sensorHello.HelmManagedConfigInit = helmManagedCfg
	}

	// Prepare outgoing context
	ctx := context.Background()

	ctx = metadata.AppendToOutgoingContext(ctx, centralsensor.SensorHelloMetadataKey, "true")
	ctx, err := centralsensor.AppendSensorHelloInfoToOutgoingMetadata(ctx, sensorHello)
	if err != nil {
		s.stopper.Flow().StopWithError(err)
		return
	}

	stream, err = communicateWithAutoSensedEncoding(ctx, client)
	if err != nil {
		s.stopper.Flow().StopWithError(err)
		return
	}

	if err := s.initialSync(stream, sensorHello, configHandler, detector); err != nil {
		s.stopper.Flow().StopWithError(err)
		return
	}

	log.Info("Established connection to Central.")

	centralReachable.Set(true)
	defer centralReachable.Set(false)
	syncDone.Signal()

	// Start receiving and sending with central.
	////////////////////////////////////////////
	s.allFinished.Add(2)
	s.receiver.Start(stream, s.Stop, s.sender.Stop)
	s.sender.Start(stream, s.clientReconcile, s.initialDeduperState, s.Stop, s.receiver.Stop)
	log.Info("Communication with central started.") // Do not change this line, it is checked by TLSChallengeTest.

	// Wait for stop.
	/////////////////
	<-s.stopper.Flow().StopRequested()
	log.Info("Communication with central ended.")
}

func (s *centralCommunicationImpl) initialSync(stream central.SensorService_CommunicateClient, hello *central.SensorHello, configHandler config.Handler, detector detector.Detector) error {
	rawHdr, err := stream.Header()
	if err != nil {
		return errors.Wrap(err, "receiving headers from central")
	}

	var centralHello *central.CentralHello

	hdr := metautils.MD(rawHdr)
	if hdr.Get(centralsensor.SensorHelloMetadataKey) == "true" {
		// Yay, central supports the "sensor hello" protocol!
		err := stream.Send(&central.MsgFromSensor{Msg: &central.MsgFromSensor_Hello{Hello: hello}})
		if err != nil {
			return errors.Wrap(err, "sending SensorHello message to central")
		}

		firstMsg, err := stream.Recv()
		if err != nil {
			return errors.Wrap(err, "receiving first message from central")
		}
		centralHello = firstMsg.GetHello()
		if centralHello == nil {
			return errors.Errorf("first message received from central was not CentralHello but of type %T", firstMsg.GetMsg())
		}
	} else {
		// No sensor hello :(
		log.Warn("Central is running a legacy version that might not support all current features")
	}

	clusterID := centralHello.GetClusterId()
	clusterid.Set(clusterID)

	if centralHello.GetManagedCentral() {
		log.Info("Central is managed")
	}

	managedcentral.Set(centralHello.GetManagedCentral())
	centralid.Set(centralHello.GetCentralId())
	centralCaps := centralHello.GetCapabilities()
	centralcaps.Set(sliceutils.FromStringSlice[centralsensor.CentralCapability](centralCaps...))

	// Sensor should only communicate deduper states if central is able to do so and it has requested it.
	s.clientReconcile = s.clientReconcile &&
		centralcaps.Has(centralsensor.SendDeduperStateOnReconnect) &&
		centralHello.GetSendDeduperState()

	log.Infof("Sensor client reconciliation state=%s (centralCapability=%s, centralHello.SendDeduperState=%s)",
		strconv.FormatBool(s.clientReconcile),
		strconv.FormatBool(centralcaps.Has(centralsensor.SendDeduperStateOnReconnect)),
		strconv.FormatBool(centralHello.GetSendDeduperState()))

	if hello.HelmManagedConfigInit != nil {
		if err := helmconfig.StoreCachedClusterID(clusterID); err != nil {
			log.Warnf("Could not cache cluster ID: %v", err)
		}
	}

	if err := safe.RunE(func() error {
		return certdistribution.PersistCertificates(centralHello.GetCertBundle())
	}); err != nil {
		log.Warnf("Failed to persist certificates for distribution: %v. This might cause issues with the admission control service.", err)
	}

	// DO NOT CHANGE THE ORDER. Please refer to `Run()` at `central/sensor/service/connection/connection_impl.go`
	if err := s.initialConfigSync(stream, configHandler); err != nil {
		return err
	}

	if err := s.initialPolicySync(stream, detector); err != nil {
		return err
	}

	return s.initialDeduperSync(stream)
}

func (s *centralCommunicationImpl) initialDeduperSync(stream central.SensorService_CommunicateClient) error {
	// If client reconciliation is disabled don't expect a deduper sync message to arrive
	if !s.clientReconcile {
		log.Info("Skipping client reconciliation. Sensor will not wait for deduper state")
		return nil
	}
	log.Info("Waiting for deduper state from Central")
	current := int32(1)
	deduperState := make(map[string]uint64)
	for {
		done := make(chan struct{})
		var err error
		var msg *central.MsgToSensor
		go func() {
			msg, err = stream.Recv()
			close(done)
		}()
		select {
		case <-time.After(s.syncTimeout):
			return errTimeoutWaitingForDeduperState
		case <-done:
		}
		if err != nil {
			if e, ok := status.FromError(err); ok {
				if e.Code() == codes.ResourceExhausted {
					return errors.Wrap(errLargePayload, e.String())
				}
			}
			return errors.Wrap(err, "receiving initial deduper sync")
		}
		if msg.GetDeduperState() == nil {
			return errors.Wrapf(errIncorrectEventOrder, "expected DeduperState but received %t instead", msg.GetMsg())
		}

		// If the expected current is different from the one received is better to stop the connection and reconnect with
		// sensor's reconciliation disabled.
		if current != msg.GetDeduperState().GetCurrent() {
			return errors.Wrapf(errIncorrectDeduperStateOrder, "expected message number %d but received %d", current, msg.GetDeduperState().GetCurrent())
		}

		log.Infof("Received %d hashes (size=%d), current chunk: %d, total: %d", len(msg.GetDeduperState().GetResourceHashes()), msg.SizeVT(), msg.GetDeduperState().GetCurrent(), msg.GetDeduperState().GetTotal())
		for k, v := range msg.GetDeduperState().GetResourceHashes() {
			deduperState[k] = v
		}

		if msg.GetDeduperState().GetCurrent() == msg.GetDeduperState().GetTotal() {
			break
		}
		current++
	}
	s.initialDeduperState = deduperkey.ParseDeduperState(deduperState)
	return nil
}

func (s *centralCommunicationImpl) initialConfigSync(stream central.SensorService_CommunicateClient, handler config.Handler) error {
	msg, err := stream.Recv()
	if err != nil {
		return errors.Wrap(err, "receiving initial cluster config")
	}
	if msg.GetClusterConfig() == nil {
		return errors.Errorf("initial message received from Sensor was not a cluster config: %T", msg.Msg)
	}
	// Send the initial cluster config to the config handler
	if err := handler.ProcessMessage(msg); err != nil {
		return errors.Wrap(err, "processing initial cluster config")
	}
	return nil
}

func (s *centralCommunicationImpl) initialPolicySync(stream central.SensorService_CommunicateClient, detector detector.Detector) error {
	// Policy sync
	msg, err := stream.Recv()
	if err != nil {
		return errors.Wrap(err, "receiving initial policies")
	}
	if msg.GetPolicySync() == nil {
		return errors.Errorf("second message received from Sensor was not a policy sync: %T", msg.Msg)
	}
	if err := detector.ProcessPolicySync(context.Background(), msg.GetPolicySync()); err != nil {
		return errors.Wrap(err, "policy sync could not be successfully processed")
	}

	// Process baselines sync
	msg, err = stream.Recv()
	if err != nil {
		return errors.Wrap(err, "receiving initial baselines")
	}
	if err := detector.ProcessMessage(msg); err != nil {
		return errors.Wrap(err, "process baselines could not be successfully processed")
	}

	// Network Baseline sync
	msg, err = stream.Recv()
	if err != nil {
		return errors.Wrap(err, "receiving network baseline sync")
	}
	if msg.GetNetworkBaselineSync() == nil {
		return errors.Errorf("expected NetworkBaseline message but received %t", msg.Msg)
	}
	if err := detector.ProcessMessage(msg); err != nil {
		return errors.Wrap(err, "network baselines could not be successfully processed")
	}
	return nil
}

func runAll(err error, fs ...func(error)) {
	for _, f := range fs {
		f(err)
	}
}
