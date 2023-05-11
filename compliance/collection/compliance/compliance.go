package compliance

import (
	"context"
	"math/rand"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/cenkalti/backoff/v3"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/compliance/collection/auditlog"
	"github.com/stackrox/rox/compliance/collection/intervals"
	"github.com/stackrox/rox/generated/internalapi/sensor"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/clientconn"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/k8sutil"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/mtls"
	"github.com/stackrox/rox/pkg/protoutils"
	"github.com/stackrox/rox/pkg/utils"
	"github.com/stackrox/rox/pkg/version"
	"google.golang.org/grpc/metadata"
)

type Compliance struct {
	log                *logging.Logger
	nodeNameProvider   NodeNameProvider
	nodeScanner        NodeScanner
	sensorReplyHandler SensorReplyHandler
}

func NewComplianceApp(np NodeNameProvider,
	scanner NodeScanner,
	srh SensorReplyHandler) *Compliance {
	return &Compliance{
		nodeNameProvider:   np,
		nodeScanner:        scanner,
		sensorReplyHandler: srh,
	}
}

func (l *Compliance) Start() {
	log.Infof("Running StackRox Version: %s", version.GetMainVersion())
	clientconn.SetUserAgent(clientconn.Compliance)

	// Set the random seed based on the current time.
	rand.Seed(time.Now().UnixNano())

	// Set up Compliance <-> Sensor connection
	conn, err := clientconn.AuthenticatedGRPCConnection(env.AdvertisedEndpoint.Setting(), mtls.SensorSubject)
	if err != nil {
		log.Fatal(err)
	}
	log.Info("Initialized gRPC stream connection to Sensor")
	defer func() {
		if err := conn.Close(); err != nil {
			log.Errorf("Failed to close connection: %v", err)
		}
	}()

	cli := sensor.NewComplianceServiceClient(conn)

	ctx, cancel := context.WithCancel(context.Background())
	ctx = metadata.AppendToOutgoingContext(ctx, "rox-compliance-nodename", l.nodeNameProvider.GetNodeName())

	stoppedSig := concurrency.NewSignal()

	toSensorC := make(chan *sensor.MsgFromCompliance)
	defer close(toSensorC)
	// the anonymous go func will read from toSensorC and send it using the client
	go func() {
		l.manageStream(ctx, cli, &stoppedSig, toSensorC)
	}()

	if env.RHCOSNodeScanning.BooleanSetting() && l.nodeScanner.IsActive() {
		i := intervals.NewNodeScanIntervalFromEnv()
		nodeInventoriesC := l.nodeScanner.ManageNodeScanLoop(ctx, i)

		// sending nodeInventories into output toSensorC
		for n := range nodeInventoriesC {
			toSensorC <- n
		}
	}

	signalsC := make(chan os.Signal, 1)
	signal.Notify(signalsC, syscall.SIGINT, syscall.SIGTERM)
	// Wait for a signal to terminate
	sig := <-signalsC
	log.Infof("Caught %s signal. Shutting down", sig)

	cancel()
	stoppedSig.Wait()
	log.Info("Successfully closed Sensor communication")
}

func (l *Compliance) manageStream(ctx context.Context, cli sensor.ComplianceServiceClient, sig *concurrency.Signal, toSensorC <-chan *sensor.MsgFromCompliance) {
	for {
		select {
		case <-ctx.Done():
			sig.Signal()
			return
		default:
			// initializeStream must only be called once across all Compliance components,
			// as multiple calls would overwrite associations on the Sensor side.
			client, config, err := l.initializeStream(ctx, cli)
			if err != nil {
				if ctx.Err() != nil {
					// continue and the <-ctx.Done() path should be taken next iteration
					continue
				}
				log.Fatalf("error initializing stream to sensor: %v", err)
			}
			// A second Context is introduced for cancelling the goroutine if runRecv returns.
			// runRecv only returns on errors, upon which the client will get reinitialized,
			// orphaning manageSendToSensor in the process.
			ctx2, cancelFn := context.WithCancel(ctx)
			if toSensorC != nil {
				go l.manageSendToSensor(ctx2, client, toSensorC)
			}
			if err := l.runRecv(ctx, client, config); err != nil {
				log.Errorf("error running recv: %v", err)
			}
			cancelFn() // runRecv is blocking, so the context is safely cancelled before the next  call to initializeStream
		}
	}
}

func (l *Compliance) runRecv(ctx context.Context, client sensor.ComplianceService_CommunicateClient, config *sensor.MsgToCompliance_ScrapeConfig) error {
	var auditReader auditlog.Reader
	defer func() {
		if auditReader != nil {
			// Stopping is idempotent so no need to check if it's already been called
			auditReader.StopReader()
		}
	}()

	for {
		msg, err := client.Recv()
		if err != nil {
			return errors.Wrap(err, "error receiving msg from sensor")
		}
		switch t := msg.Msg.(type) {
		case *sensor.MsgToCompliance_Trigger:
			if err := runChecks(client, config, t.Trigger, l.nodeNameProvider); err != nil {
				return errors.Wrap(err, "error running checks")
			}
		case *sensor.MsgToCompliance_AuditLogCollectionRequest_:
			switch r := t.AuditLogCollectionRequest.GetReq().(type) {
			case *sensor.MsgToCompliance_AuditLogCollectionRequest_StartReq:
				if auditReader != nil {
					log.Info("Audit log reader is being restarted")
					auditReader.StopReader() // stop the old one
				}
				auditReader = l.startAuditLogCollection(ctx, client, r.StartReq)
			case *sensor.MsgToCompliance_AuditLogCollectionRequest_StopReq:
				if auditReader != nil {
					log.Infof("Stopping audit log reader on node %s.", l.nodeNameProvider.GetNodeName())
					auditReader.StopReader()
					auditReader = nil
				} else {
					log.Warn("Attempting to stop an un-started audit log reader - this is a no-op")
				}
			}
		case *sensor.MsgToCompliance_Ack:
			switch t.Ack.GetAction() {
			case sensor.MsgToCompliance_NodeInventoryACK_ACK:
				// TODO(ROX-16687): Implement behavior when receiving Ack here
				// TODO(ROX-16549): Add metric to see the ratio of Ack/Nack(?)
				l.sensorReplyHandler.HandleACK(ctx, client)
			case sensor.MsgToCompliance_NodeInventoryACK_NACK:
				l.sensorReplyHandler.HandleNACK(ctx, client)
			}
		default:
			utils.Should(errors.Errorf("Unhandled msg type: %T", t))
		}
	}
}

func (l *Compliance) startAuditLogCollection(ctx context.Context, client sensor.ComplianceService_CommunicateClient, request *sensor.MsgToCompliance_AuditLogCollectionRequest_StartRequest) auditlog.Reader {
	if request.GetCollectStartState() == nil {
		log.Infof("Starting audit log reader on node %s in cluster %s with no saved state", l.nodeNameProvider.GetNodeName(), request.GetClusterId())
	} else {
		log.Infof("Starting audit log reader on node %s in cluster %s using previously saved state: %s)",
			l.nodeNameProvider.GetNodeName(), request.GetClusterId(), protoutils.NewWrapper(request.GetCollectStartState()))
	}

	auditReader := auditlog.NewReader(client, l.nodeNameProvider.GetNodeName(), request.GetClusterId(), request.GetCollectStartState())
	start, err := auditReader.StartReader(ctx)
	if err != nil {
		log.Errorf("Failed to start audit log reader %v", err)
		// TODO: Report health
	} else if !start {
		// It shouldn't get here unless Sensor mistakenly sends a start event to a non-master node
		log.Error("Audit log reader did not start because audit logs do not exist on this node")
	}
	return auditReader
}

func (l *Compliance) manageSendToSensor(ctx context.Context, cli sensor.ComplianceService_CommunicateClient, toSensorC <-chan *sensor.MsgFromCompliance) {
	for {
		select {
		case <-ctx.Done():
			return
		case sc := <-toSensorC:
			if err := cli.Send(sc); err != nil {
				log.Errorf("failed sending node scan to sensor: %v", err)
			}
		}
	}
}

func (l *Compliance) initializeStream(ctx context.Context, cli sensor.ComplianceServiceClient) (sensor.ComplianceService_CommunicateClient, *sensor.MsgToCompliance_ScrapeConfig, error) {
	eb := backoff.NewExponentialBackOff()
	eb.MaxInterval = 30 * time.Second
	eb.MaxElapsedTime = 3 * time.Minute

	var client sensor.ComplianceService_CommunicateClient
	var config *sensor.MsgToCompliance_ScrapeConfig

	operation := func() error {
		var err error
		client, config, err = l.initialClientAndConfig(ctx, cli)
		if err != nil && ctx.Err() != nil {
			return backoff.Permanent(err)
		}
		return err
	}
	err := backoff.RetryNotify(operation, eb, func(err error, t time.Duration) {
		log.Infof("Sleeping for %0.2f seconds between attempts to connect to Sensor, err: %v", t.Seconds(), err)
	})
	if err != nil {
		return nil, nil, errors.Wrap(err, "Failed to initialize sensor connection")
	}
	log.Infof("Successfully connected to Sensor at %s", env.AdvertisedEndpoint.Setting())

	return client, config, nil
}

func (l *Compliance) initialClientAndConfig(ctx context.Context, cli sensor.ComplianceServiceClient) (sensor.ComplianceService_CommunicateClient, *sensor.MsgToCompliance_ScrapeConfig, error) {
	client, err := cli.Communicate(ctx)
	if err != nil {
		return nil, nil, errors.Wrap(err, "error communicating with sensor")
	}

	initialMsg, err := client.Recv()
	if err != nil {
		return nil, nil, errors.Wrap(err, "error receiving initial msg from sensor")
	}

	if initialMsg.GetConfig() == nil {
		return nil, nil, errors.New("initial msg has a nil config")
	}
	config := initialMsg.GetConfig()
	if config.ContainerRuntime == storage.ContainerRuntime_UNKNOWN_CONTAINER_RUNTIME {
		log.Error("Didn't receive container runtime from sensor. Trying to infer container runtime from cgroups...")
		config.ContainerRuntime, err = k8sutil.InferContainerRuntime()
		if err != nil {
			log.Errorf("Could not infer container runtime from cgroups: %v", err)
		} else {
			log.Infof("Inferred container runtime as %s", config.ContainerRuntime.String())
		}
	}
	return client, config, nil
}
