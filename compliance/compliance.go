package compliance

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/cenkalti/backoff/v3"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/compliance/collection/auditlog"
	"github.com/stackrox/rox/compliance/collection/compliance_checks"
	cmetrics "github.com/stackrox/rox/compliance/collection/metrics"
	"github.com/stackrox/rox/compliance/node"
	v4 "github.com/stackrox/rox/generated/internalapi/scanner/v4"
	"github.com/stackrox/rox/generated/internalapi/sensor"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/clientconn"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/k8sutil"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/metrics"
	"github.com/stackrox/rox/pkg/mtls"
	"github.com/stackrox/rox/pkg/protoutils"
	"github.com/stackrox/rox/pkg/utils"
	"github.com/stackrox/rox/pkg/version"
	"google.golang.org/grpc/metadata"
)

var log = logging.LoggerForModule()

// Compliance represents the Compliance app
type Compliance struct {
	nodeNameProvider node.NodeNameProvider
	nodeScanner      node.NodeScanner
	nodeIndexer      node.NodeIndexer
	umhNodeInventory node.UnconfirmedMessageHandler
	umhNodeIndex     node.UnconfirmedMessageHandler
	cache            *sensor.MsgFromCompliance
}

// NewComplianceApp constructs the Compliance app object
func NewComplianceApp(nnp node.NodeNameProvider, scanner node.NodeScanner, nodeIndexer node.NodeIndexer,
	umhNodeInv, umhNodeIndex node.UnconfirmedMessageHandler) *Compliance {
	return &Compliance{
		nodeNameProvider: nnp,
		nodeScanner:      scanner,
		nodeIndexer:      nodeIndexer,
		umhNodeInventory: umhNodeInv,
		umhNodeIndex:     umhNodeIndex,
		cache:            nil,
	}
}

// Start starts the Compliance app
func (c *Compliance) Start() {
	log.Infof("Running StackRox Version: %s", version.GetMainVersion())
	clientconn.SetUserAgent(clientconn.Compliance)

	// Start the prometheus metrics server
	metrics.NewServer(metrics.ComplianceSubsystem, metrics.NewTLSConfigurerFromEnv()).RunForever()
	metrics.GatherThrottleMetricsForever(metrics.ComplianceSubsystem.String())

	// Set up Compliance <-> Sensor connection
	conn, err := clientconn.AuthenticatedGRPCConnection(context.Background(), env.AdvertisedEndpoint.Setting(), mtls.SensorSubject)
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
	signalsC := make(chan os.Signal, 1)
	signal.Notify(signalsC, syscall.SIGINT, syscall.SIGTERM)

	toSensorC := make(chan *sensor.MsgFromCompliance)
	defer close(toSensorC)
	// the anonymous go func will read from toSensorC and send it using the client
	go func() {
		c.manageStream(ctx, cli, &stoppedSig, toSensorC)
	}()

	var wg concurrency.WaitGroup
	wg.Add(2)

	go func(ctx context.Context) {
		defer wg.Add(-1)
		if c.nodeScanner.IsActive() {
			log.Infof("Node Inventory v2 enabled")
			nodeInventoriesC := c.manageNodeInventoryScanLoop(ctx)
			// sending nodeInventories into output toSensorC
			for n := range nodeInventoriesC {
				toSensorC <- n
			}
		}
	}(ctx)

	go func(ctx context.Context) {
		defer wg.Add(-1)
		if features.NodeIndexEnabled.Enabled() {
			log.Infof("Node Index v4 enabled")
			nodeIndexesC := c.manageNodeIndexScanLoop(ctx)
			// sending node indexes into output toSensorC
			for n := range nodeIndexesC {
				toSensorC <- n
			}
		}
	}(ctx)

	// Wait for the terminate signal
	go func() {
		sig := <-signalsC
		log.Infof("Caught %s signal. Shutting down", sig)
		// Stop generation of node inventories and node indexes
		cancel()
	}()

	<-wg.Done()
	log.Infof("Generation of node inventories and node indexes stopped")

	stoppedSig.Wait()
	log.Info("Successfully closed Sensor communication")
}

func (c *Compliance) createIndexMsg(report *v4.IndexReport, nodeName string) *sensor.MsgFromCompliance {
	return &sensor.MsgFromCompliance{
		Node: nodeName,
		Msg:  &sensor.MsgFromCompliance_IndexReport{IndexReport: report},
	}
}

func (c *Compliance) manageNodeInventoryScanLoop(ctx context.Context) <-chan *sensor.MsgFromCompliance {
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
			case _, ok := <-c.umhNodeInventory.RetryCommand():
				if c.cache == nil {
					log.Debug("Requested to retry but cache is empty. Resetting scan timer.")
					cmetrics.ObserveNodePackageReportTransmissions(nodeName, cmetrics.InventoryTransmissionResendingCacheMiss, cmetrics.ScannerVersionV2)
					t.Reset(time.Second)
				} else if ok {
					nodeInventoriesC <- c.cache
					cmetrics.ObserveNodePackageReportTransmissions(nodeName, cmetrics.InventoryTransmissionResendingCacheHit, cmetrics.ScannerVersionV2)
				}
			case <-t.C:
				if c.nodeScanner.IsActive() {
					inventory := c.runNodeInventoryScan(ctx)
					if inventory != nil {
						nodeInventoriesC <- inventory
					}
				}
				interval := i.Next()
				cmetrics.ObserveRescanInterval(interval, nodeName, cmetrics.ScannerVersionV2)
				t.Reset(interval)
			}
		}
	}()
	return nodeInventoriesC
}

func (c *Compliance) manageNodeIndexScanLoop(ctx context.Context) <-chan *sensor.MsgFromCompliance {
	nodeIndexesC := make(chan *sensor.MsgFromCompliance)
	nodeName := c.nodeNameProvider.GetNodeName()
	go func() {
		defer close(nodeIndexesC)
		i := c.nodeIndexer.GetIntervals()
		t := time.NewTicker(i.Initial())
		for {
			select {
			case <-ctx.Done():
				return
			case _, ok := <-c.umhNodeIndex.RetryCommand():
				if c.cache == nil {
					log.Debug("Requested to retry but cache is empty. Resetting scan timer.")
					cmetrics.ObserveNodePackageReportTransmissions(nodeName, cmetrics.InventoryTransmissionResendingCacheMiss, cmetrics.ScannerVersionV4)
					t.Reset(time.Second)
				} else if ok {
					nodeIndexesC <- c.cache
					cmetrics.ObserveNodePackageReportTransmissions(nodeName, cmetrics.InventoryTransmissionResendingCacheHit, cmetrics.ScannerVersionV4)
				}
			case <-t.C:
				if features.NodeIndexEnabled.Enabled() {
					index := c.runNodeIndex(ctx)
					if index != nil {
						nodeIndexesC <- index
					}
				}
				interval := i.Next()
				cmetrics.ObserveRescanInterval(interval, nodeName, cmetrics.ScannerVersionV4)
				t.Reset(interval)
			}
		}
	}()
	return nodeIndexesC
}

func (c *Compliance) runNodeInventoryScan(ctx context.Context) *sensor.MsgFromCompliance {
	nodeName := c.nodeNameProvider.GetNodeName()
	msg, err := c.nodeScanner.ScanNode(ctx)
	if err != nil {
		log.Errorf("Error running node scan: %v", err)
		return nil
	}
	cmetrics.ObserveNodeInventoryScan(msg.GetNodeInventory())
	cmetrics.ObserveNodePackageReportTransmissions(nodeName, cmetrics.InventoryTransmissionScan, cmetrics.ScannerVersionV2)
	c.umhNodeInventory.ObserveSending()
	c.cache = msg.CloneVT()
	return msg
}

func (c *Compliance) runNodeIndex(ctx context.Context) *sensor.MsgFromCompliance {
	nodeName := c.nodeNameProvider.GetNodeName()
	log.Infof("Creating v4 Node Index report for node %s", nodeName)
	cmetrics.ObserveIndexesTotal(nodeName)
	startTime := time.Now()
	report, err := c.nodeIndexer.IndexNode(ctx)
	duration := time.Since(startTime)
	cmetrics.ObserveIndexDuration(duration, nodeName, err)
	if err != nil {
		log.Errorf("Error creating node index: %v", err)
		return nil
	}
	c.umhNodeIndex.ObserveSending()
	cmetrics.ObserveNodeIndexReport(report, nodeName)
	msg := c.createIndexMsg(report, nodeName)
	cmetrics.ObserveReportProtobufMessage(msg, cmetrics.ScannerVersionV4)
	cmetrics.ObserveNodePackageReportTransmissions(nodeName, cmetrics.InventoryTransmissionScan, cmetrics.ScannerVersionV4)
	return msg
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
			cancelFn() // runRecv is blocking, so the context is safely cancelled before the next call to initializeStream
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
			return errors.Wrap(err, "receiving msg from sensor")
		}
		switch t := msg.Msg.(type) {
		case *sensor.MsgToCompliance_Trigger:
			if err := compliance_checks.RunChecks(client, config, t.Trigger, c.nodeNameProvider); err != nil {
				return errors.Wrap(err, "running compliance checks")
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
					log.Infof("Stopping audit log reader on node %s.", c.nodeNameProvider.GetNodeName())
					auditReader.StopReader()
					auditReader = nil
				} else {
					log.Warn("Attempting to stop an un-started audit log reader - this is a no-op")
				}
			}
		case *sensor.MsgToCompliance_Ack:
			switch t.Ack.GetAction() {
			case sensor.MsgToCompliance_NodeInventoryACK_ACK:
				switch t.Ack.GetMessageType() {
				case sensor.MsgToCompliance_NodeInventoryACK_NodeInventory:
					c.umhNodeInventory.HandleACK()
				case sensor.MsgToCompliance_NodeInventoryACK_NodeIndexer:
					c.umhNodeIndex.HandleACK()
				default:
					log.Errorf("Unknown ACK Type: %s", t.Ack.GetMessageType())
				}
			case sensor.MsgToCompliance_NodeInventoryACK_NACK:
				switch t.Ack.GetMessageType() {
				case sensor.MsgToCompliance_NodeInventoryACK_NodeInventory:
					c.umhNodeInventory.HandleNACK()
				case sensor.MsgToCompliance_NodeInventoryACK_NodeIndexer:
					c.umhNodeIndex.HandleNACK()
				default:
					log.Errorf("Unknown ACK Type: %s", t.Ack.GetMessageType())
				}
			default:
				log.Errorf("Unknown ACK Action: %s", t.Ack.GetAction())
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
	eb.MaxElapsedTime = 15 * time.Minute

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
