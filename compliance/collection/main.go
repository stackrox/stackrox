package main

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
	"github.com/stackrox/rox/compliance/collection/inventory"
	cmetrics "github.com/stackrox/rox/compliance/collection/metrics"
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
	"github.com/stackrox/rox/pkg/orchestrators"
	"github.com/stackrox/rox/pkg/protoutils"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/pkg/utils"
	"github.com/stackrox/rox/pkg/version"
	scannerV1 "github.com/stackrox/scanner/generated/scanner/api/v1"
	"google.golang.org/grpc/metadata"
)

var (
	log = logging.LoggerForModule()

	node string
	once sync.Once
)

func getNode() string {
	once.Do(func() {
		node = os.Getenv(string(orchestrators.NodeName))
		if node == "" {
			log.Fatal("No node name found in the environment")
		}
	})
	return node
}

func runRecv(ctx context.Context, client sensor.ComplianceService_CommunicateClient, config *sensor.MsgToCompliance_ScrapeConfig) error {
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
			if err := runChecks(client, config, t.Trigger); err != nil {
				return errors.Wrap(err, "error running checks")
			}
		case *sensor.MsgToCompliance_AuditLogCollectionRequest_:
			switch r := t.AuditLogCollectionRequest.GetReq().(type) {
			case *sensor.MsgToCompliance_AuditLogCollectionRequest_StartReq:
				if auditReader != nil {
					log.Info("Audit log reader is being restarted")
					auditReader.StopReader() // stop the old one
				}
				auditReader = startAuditLogCollection(ctx, client, r.StartReq)
			case *sensor.MsgToCompliance_AuditLogCollectionRequest_StopReq:
				if auditReader != nil {
					log.Infof("Stopping audit log reader on node %s.", getNode())
					auditReader.StopReader()
					auditReader = nil
				} else {
					log.Warn("Attempting to stop an un-started audit log reader - this is a no-op")
				}
			}
		default:
			utils.Should(errors.Errorf("Unhandled msg type: %T", t))
		}
	}
}

func startAuditLogCollection(ctx context.Context, client sensor.ComplianceService_CommunicateClient, request *sensor.MsgToCompliance_AuditLogCollectionRequest_StartRequest) auditlog.Reader {
	if request.GetCollectStartState() == nil {
		log.Infof("Starting audit log reader on node %s in cluster %s with no saved state", getNode(), request.GetClusterId())
	} else {
		log.Infof("Starting audit log reader on node %s in cluster %s using previously saved state: %s)",
			getNode(), request.GetClusterId(), protoutils.NewWrapper(request.GetCollectStartState()))
	}

	auditReader := auditlog.NewReader(client, getNode(), request.GetClusterId(), request.GetCollectStartState())
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

func manageStream(ctx context.Context, cli sensor.ComplianceServiceClient, sig *concurrency.Signal, sensorC <-chan *sensor.MsgFromCompliance) {
	for {
		select {
		case <-ctx.Done():
			sig.Signal()
			return
		default:
			// initializeStream must only be called once across all Compliance components,
			// as multiple calls would overwrite associations on the Sensor side.
			client, config, err := initializeStream(ctx, cli)
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
			if sensorC != nil {
				go manageSendToSensor(ctx2, client, sensorC)
			}
			if err := runRecv(ctx, client, config); err != nil {
				log.Errorf("error running recv: %v", err)
			}
			cancelFn() // runRecv is blocking, so the context is safely cancelled before the next  call to initializeStream
		}
	}
}

func manageSendToSensor(ctx context.Context, cli sensor.ComplianceService_CommunicateClient, sensorC <-chan *sensor.MsgFromCompliance) {
	for {
		select {
		case <-ctx.Done():
			return
		case sc := <-sensorC:
			if err := cli.Send(sc); err != nil {
				log.Errorf("failed sending node scan to sensor: %v", err)
			}
		}
	}
}

func manageNodeScanLoop(ctx context.Context, i intervals.NodeScanIntervals, scanner scannerV1.NodeInventoryServiceClient) <-chan *sensor.MsgFromCompliance {
	nodeInventoriesC := make(chan *sensor.MsgFromCompliance)
	nodeName := getNode()
	go func() {
		defer close(nodeInventoriesC)
		t := time.NewTicker(i.Initial())
		for {
			select {
			case <-ctx.Done():
				return
			case <-t.C:
				log.Infof("starting a node scan for node %q", nodeName)
				msg, err := scanNode(scanner)
				if err != nil {
					log.Errorf("error running scanNode: %v", err)
				} else {
					nodeInventoriesC <- msg
				}
				interval := i.Next()
				cmetrics.ObserveRescanInterval(interval, getNode())
				t.Reset(interval)
			}
		}
	}()
	return nodeInventoriesC
}

func scanNode(scanner scannerV1.NodeInventoryServiceClient) (*sensor.MsgFromCompliance, error) {
	result, err := scanner.GetNodeInventory(context.Background(), &scannerV1.GetNodeInventoryRequest{})
	if err != nil {
		return nil, err
	}
	inv := inventory.ToNodeInventory(result)
	msg := &sensor.MsgFromCompliance{
		Node: result.GetNodeName(),
		Msg:  &sensor.MsgFromCompliance_NodeInventory{NodeInventory: inv},
	}
	cmetrics.ObserveInventoryProtobufMessage(msg)
	return msg, nil
}

func initialClientAndConfig(ctx context.Context, cli sensor.ComplianceServiceClient) (sensor.ComplianceService_CommunicateClient, *sensor.MsgToCompliance_ScrapeConfig, error) {
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

func initializeStream(ctx context.Context, cli sensor.ComplianceServiceClient) (sensor.ComplianceService_CommunicateClient, *sensor.MsgToCompliance_ScrapeConfig, error) {
	eb := backoff.NewExponentialBackOff()
	eb.MaxInterval = 30 * time.Second
	eb.MaxElapsedTime = 3 * time.Minute

	var client sensor.ComplianceService_CommunicateClient
	var config *sensor.MsgToCompliance_ScrapeConfig

	operation := func() error {
		var err error
		client, config, err = initialClientAndConfig(ctx, cli)
		if err != nil && ctx.Err() != nil {
			return backoff.Permanent(err)
		}
		return err
	}
	err := backoff.RetryNotify(operation, eb, func(err error, t time.Duration) {
		log.Infof("Sleeping for %0.2f seconds between attempts to connect to Sensor", t.Seconds())
	})
	if err != nil {
		return nil, nil, errors.Wrap(err, "Failed to initialize sensor connection")
	}
	log.Infof("Successfully connected to Sensor at %s", env.AdvertisedEndpoint.Setting())

	return client, config, nil
}

func main() {
	log.Infof("Running StackRox Version: %s", version.GetMainVersion())
	clientconn.SetUserAgent(clientconn.Compliance)

	// Set the random seed based on the current time.
	rand.Seed(time.Now().UnixNano())

	var nodeInventoryClient scannerV1.NodeInventoryServiceClient

	if features.RHCOSNodeScanning.Enabled() {
		// Start the prometheus metrics server
		metrics.NewDefaultHTTPServer(metrics.ComplianceSubsystem).RunForever()
		metrics.GatherThrottleMetricsForever(metrics.ComplianceSubsystem.String())

		// Set up Compliance <-> NodeInventory connection
		niConn, err := clientconn.AuthenticatedGRPCConnection(env.NodeScanningEndpoint.Setting(), mtls.Subject{}, clientconn.UseInsecureNoTLS(true))
		if err != nil {
			log.Errorf("Could not initialize connection to NodeInventory service. Node Scanning will be unavailable: %v", err)
		}
		if niConn != nil {
			log.Info("Initialized NodeInventory gRPC connection")
			nodeInventoryClient = scannerV1.NewNodeInventoryServiceClient(niConn)
		}
	}

	// Set up Compliance <-> Sensor connection
	conn, err := clientconn.AuthenticatedGRPCConnection(env.AdvertisedEndpoint.Setting(), mtls.SensorSubject)
	if err != nil {
		log.Fatal(err)
	}
	log.Info("Initialized Sensor gRPC stream connection")
	defer func() {
		if err := conn.Close(); err != nil {
			log.Errorf("Failed to close connection: %v", err)
		}
	}()

	cli := sensor.NewComplianceServiceClient(conn)

	ctx, cancel := context.WithCancel(context.Background())
	ctx = metadata.AppendToOutgoingContext(ctx, "rox-compliance-nodename", getNode())

	stoppedSig := concurrency.NewSignal()

	sensorC := make(chan *sensor.MsgFromCompliance)
	defer close(sensorC)
	go manageStream(ctx, cli, &stoppedSig, sensorC)

	// TODO(ROX-13935): Remove FakeNodeInventory and its FF
	if features.RHCOSNodeScanning.Enabled() && nodeInventoryClient != nil {
		i := intervals.NewNodeScanIntervalFromEnv()
		nodeInventoriesC := manageNodeScanLoop(ctx, i, nodeInventoryClient)

		// multiplex producers (nodeInventoriesC) into the output channel (sensorC)
		go func() {
			for {
				select {
				case <-ctx.Done():
					return
				case msg, more := <-nodeInventoriesC:
					if !more {
						return
					}
					sensorC <- msg
				}
			}
		}()
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
