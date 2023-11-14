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
	cmetrics "github.com/stackrox/rox/compliance/collection/metrics"
	"github.com/stackrox/rox/generated/internalapi/sensor"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/clientconn"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/k8sutil"
	"github.com/stackrox/rox/pkg/metrics"
	"github.com/stackrox/rox/pkg/mtls"
	"github.com/stackrox/rox/pkg/protoutils"
	"github.com/stackrox/rox/pkg/utils"
	"github.com/stackrox/rox/pkg/version"
	"google.golang.org/grpc/metadata"
)

// Compliance represents the Compliance app
type Compliance struct {
	nodeNameProvider NodeNameProvider
	nodeScanner      NodeScanner
	umh              UnconfirmedMessageHandler
	cache            *sensor.MsgFromCompliance
}

// NewComplianceApp contsructs the Compliance app object
func NewComplianceApp(nnp NodeNameProvider, scanner NodeScanner,
	srh UnconfirmedMessageHandler) *Compliance {
	return &Compliance{
		nodeNameProvider: nnp,
		nodeScanner:      scanner,
		umh:              srh,
		cache:            nil,
	}
}

// Start starts the Compliance app
func (c *Compliance) Start() {
	log.Infof("Running StackRox Version: %s", version.GetMainVersion())
	clientconn.SetUserAgent(clientconn.Compliance)

	// Set the random seed based on the current time.
	rand.Seed(time.Now().UnixNano())

	// Start the prometheus metrics server
	metrics.NewServer(metrics.ComplianceSubsystem, metrics.NewTLSConfigurerFromEnv()).RunForever()
	metrics.GatherThrottleMetricsForever(metrics.ComplianceSubsystem.String())

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
	ctx = metadata.AppendToOutgoingContext(ctx, "rox-compliance-nodename", c.nodeNameProvider.GetNodeName())

	stoppedSig := concurrency.NewSignal()

	toSensorC := make(chan *sensor.MsgFromCompliance)
	defer close(toSensorC)
	// the anonymous go func will read from toSensorC and send it using the client
	go func() {
		c.manageStream(ctx, cli, &stoppedSig, toSensorC)
	}()

	if c.nodeScanner.IsActive() {
		nodeInventoriesC := c.manageNodeScanLoop(ctx)
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

func (c *Compliance) manageNodeScanLoop(ctx context.Context) <-chan *sensor.MsgFromCompliance {
	nodeInventoriesC := make(chan *sensor.MsgFromCompliance)
	nodeName := c.nodeNameProvider.GetNodeName()
	go func() {
		defer close(nodeInventoriesC)
		i := c.nodeScanner.GetIntervals()
		t := time.NewTicker(i.Initial())
		for {
			select {
			case <-ctx.Done():
				return
			case _, ok := <-c.umh.RetryCommand():
				if c.cache == nil {
					log.Debug("Requested to retry but cache is empty. Resetting scan timer")
					cmetrics.ObserveNodeInventorySending(nodeName, cmetrics.InventoryTransmissionResendingCacheMiss)
					t.Reset(time.Second)
				} else if ok {
					nodeInventoriesC <- c.cache
					cmetrics.ObserveNodeInventorySending(nodeName, cmetrics.InventoryTransmissionResendingCacheHit)
				}
			case <-t.C:
				log.Infof("Scanning node %q", nodeName)
				msg, err := c.nodeScanner.ScanNode(ctx)
				if err != nil {
					log.Errorf("Error running node scan: %v", err)
				} else {
					cmetrics.ObserveNodeInventoryScan(msg.GetNodeInventory())
					cmetrics.ObserveNodeInventorySending(nodeName, cmetrics.InventoryTransmissionScan)
					c.umh.ObserveSending()
					c.cache = msg.Clone()
					nodeInventoriesC <- msg
				}
				interval := i.Next()
				cmetrics.ObserveRescanInterval(interval, nodeName)
				t.Reset(interval)
			}
		}
	}()
	return nodeInventoriesC
}

func (c *Compliance) manageStream(ctx context.Context, cli sensor.ComplianceServiceClient, sig *concurrency.Signal, toSensorC <-chan *sensor.MsgFromCompliance) {
	for {
		select {
		case <-ctx.Done():
			sig.Signal()
			return
		default:
			// initializeStream must only be called once across all Compliance components,
			// as multiple calls would overwrite associations on the Sensor side.
			client, config, err := c.initializeStream(ctx, cli)
			if err != nil {
				if ctx.Err() != nil {
					// continue and the <-ctx.Done() path should be taken next iteration
					continue
				}
				log.Fatalf("Error initializing stream to sensor: %v", err)
			}
			// A second Context is introduced for cancelling the goroutine if runRecv returns.
			// runRecv only returns on errors, upon which the client will get reinitialized,
			// orphaning manageSendToSensor in the process.
			ctx2, cancelFn := context.WithCancel(ctx)
			if toSensorC != nil {
				go c.manageSendToSensor(ctx2, client, toSensorC)
			}
			if err := c.runRecv(ctx, client, config); err != nil {
				log.Errorf("Error running recv: %v", err)
			}
			cancelFn() // runRecv is blocking, so the context is safely cancelled before the next  call to initializeStream
		}
	}
}

func (c *Compliance) runRecv(ctx context.Context, client sensor.ComplianceService_CommunicateClient, config *sensor.MsgToCompliance_ScrapeConfig) error {
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
			if err := runChecks(client, config, t.Trigger, c.nodeNameProvider); err != nil {
				return errors.Wrap(err, "error running checks")
			}
		case *sensor.MsgToCompliance_AuditLogCollectionRequest_:
			switch r := t.AuditLogCollectionRequest.GetReq().(type) {
			case *sensor.MsgToCompliance_AuditLogCollectionRequest_StartReq:
				if auditReader != nil {
					log.Info("Audit log reader is being restarted")
					auditReader.StopReader() // stop the old one
				}
				auditReader = c.startAuditLogCollection(ctx, client, r.StartReq)
			case *sensor.MsgToCompliance_AuditLogCollectionRequest_StopReq:
				if auditReader != nil {
					log.Infof("Stopping audit log reader on node %s", c.nodeNameProvider.GetNodeName())
					auditReader.StopReader()
					auditReader = nil
				} else {
					log.Warn("Attempting to stop an un-started audit log reader - this is a no-op")
				}
			}
		case *sensor.MsgToCompliance_Ack:
			switch t.Ack.GetAction() {
			case sensor.MsgToCompliance_NodeInventoryACK_ACK:
				c.umh.HandleACK()
			case sensor.MsgToCompliance_NodeInventoryACK_NACK:
				c.umh.HandleNACK()
			}
		default:
			utils.Should(errors.Errorf("Unhandled msg type: %T", t))
		}
	}
}

func (c *Compliance) startAuditLogCollection(ctx context.Context, client sensor.ComplianceService_CommunicateClient, request *sensor.MsgToCompliance_AuditLogCollectionRequest_StartRequest) auditlog.Reader {
	if request.GetCollectStartState() == nil {
		log.Infof("Starting audit log reader on node %s in cluster %s with no saved state", c.nodeNameProvider.GetNodeName(), request.GetClusterId())
	} else {
		log.Infof("Starting audit log reader on node %s in cluster %s using previously saved state: %s)",
			c.nodeNameProvider.GetNodeName(), request.GetClusterId(), protoutils.NewWrapper(request.GetCollectStartState()))
	}

	auditReader := auditlog.NewReader(client, c.nodeNameProvider.GetNodeName(), request.GetClusterId(), request.GetCollectStartState())
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

func (c *Compliance) manageSendToSensor(ctx context.Context, cli sensor.ComplianceService_CommunicateClient, toSensorC <-chan *sensor.MsgFromCompliance) {
	for {
		select {
		case <-ctx.Done():
			return
		case sc := <-toSensorC:
			if err := cli.Send(sc); err != nil {
				log.Errorf("Failed sending node scan to sensor: %v", err)
			}
		}
	}
}

func (c *Compliance) initializeStream(ctx context.Context, cli sensor.ComplianceServiceClient) (sensor.ComplianceService_CommunicateClient, *sensor.MsgToCompliance_ScrapeConfig, error) {
	eb := backoff.NewExponentialBackOff()
	eb.MaxInterval = 30 * time.Second
	eb.MaxElapsedTime = 3 * time.Minute

	var client sensor.ComplianceService_CommunicateClient
	var config *sensor.MsgToCompliance_ScrapeConfig

	operation := func() error {
		var err error
		client, config, err = c.initialClientAndConfig(ctx, cli)
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

func (c *Compliance) initialClientAndConfig(ctx context.Context, cli sensor.ComplianceServiceClient) (sensor.ComplianceService_CommunicateClient, *sensor.MsgToCompliance_ScrapeConfig, error) {
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
