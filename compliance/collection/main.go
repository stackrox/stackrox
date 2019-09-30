package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gogo/protobuf/types"
	"github.com/stackrox/rox/compliance/collection/command"
	"github.com/stackrox/rox/compliance/collection/containerruntimes/crio"
	"github.com/stackrox/rox/compliance/collection/containerruntimes/docker"
	"github.com/stackrox/rox/compliance/collection/file"
	"github.com/stackrox/rox/generated/internalapi/compliance"
	"github.com/stackrox/rox/generated/internalapi/sensor"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/clientconn"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/k8sutil"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/mtls"
	"github.com/stackrox/rox/pkg/orchestrators"
	"github.com/stackrox/rox/pkg/retry"
	"github.com/stackrox/rox/pkg/version"
)

var (
	log = logging.LoggerForModule()
)

const requestTimeout = time.Second * 5

func main() {
	log.Infof("Running StackRox Version: %s", version.GetMainVersion())
	thisNodeName := os.Getenv(string(orchestrators.NodeName))
	if thisNodeName == "" {
		log.Fatal("No node name found in the environment")
	}
	thisScrapeID := os.Getenv("ROX_SCRAPE_ID")
	if thisScrapeID == "" {
		log.Fatal("No scrape ID found in the environment")
	}

	// Create a connection with sensor to push scraped data.
	conn, err := clientconn.AuthenticatedGRPCConnection(env.AdvertisedEndpoint.Setting(), mtls.SensorSubject)
	if err != nil {
		log.Fatalf("Could not establish a gRPC connection to Sensor: %v", err)
	}
	log.Info("Initialized Sensor gRPC connection")
	defer func() {
		if err := conn.Close(); err != nil {
			log.Errorf("Failed to close connection: %v", err)
		}
	}()
	cli := sensor.NewComplianceServiceClient(conn)

	log.Info("Querying sensor for scrape configuration")
	getScrapeConfigReq := &sensor.GetScrapeConfigRequest{
		NodeName: thisNodeName,
		ScrapeId: thisScrapeID,
	}

	var scrapeConfig *sensor.ScrapeConfig
	if err := retry.WithRetry(
		func() error { // Try to query sensor, time out after 5 seconds.
			log.Info("Trying to query sensor")
			ctx, cancel := context.WithTimeout(context.Background(), requestTimeout)
			defer cancel()

			var err error
			scrapeConfig, err = cli.GetScrapeConfig(ctx, getScrapeConfigReq)
			return err
		},
		retry.Tries(5), // 5 attempts.
		retry.BetweenAttempts(func(_ int) { // Sleep for a second between attempts
			log.Info("Sleeping between attempts to retrieve scrape config")
			time.Sleep(time.Second)
		}),
		retry.OnFailedAttempts(func(err error) { // Log encountered errors.
			log.Errorf("Error querying sensor for scrape config on %v: %v", env.AdvertisedEndpoint.Setting(), err)
		}),
	); err != nil {
		log.Error("Couldn't query sensor for scrape config despite retries. Trying to infer container runtime from cgrouops ...")
		scrapeConfig = &sensor.ScrapeConfig{}
		scrapeConfig.ContainerRuntime, err = k8sutil.InferContainerRuntime()
		if err != nil {
			log.Errorf("Could not infer container runtime from cgroups: %v", err)
		}
	}

	msgReturn := compliance.ComplianceReturn{
		NodeName: thisNodeName,
		ScrapeId: thisScrapeID,
	}

	log.Infof("Running compliance scrape %q for node %q", thisScrapeID, thisNodeName)

	log.Infof("Container runtime is %v", scrapeConfig.GetContainerRuntime())
	if scrapeConfig.GetContainerRuntime() == storage.ContainerRuntime_DOCKER_CONTAINER_RUNTIME {
		log.Info("Starting to collect Docker data")
		msgReturn.DockerData, msgReturn.ContainerRuntimeInfo, err = docker.GetDockerData()
		if err != nil {
			log.Errorf("Collecting Docker data failed: %v", err)
		} else {
			log.Info("Successfully collected relevant Docker data")
		}
	} else if scrapeConfig.GetContainerRuntime() == storage.ContainerRuntime_CRIO_CONTAINER_RUNTIME {
		log.Info("Collecting relevant CRI-O data")
		msgReturn.ContainerRuntimeInfo, err = crio.GetContainerRuntimeData()
		if err != nil {
			log.Errorf("Collecting CRI-O data failed: %v", err)
		} else {
			log.Info("Successfully collected relevant CRI-O data")
		}
	} else {
		log.Info("Unknown container runtime, not collecting any data ...")
	}

	log.Info("Starting to collect systemd files")
	msgReturn.SystemdFiles, err = file.CollectSystemdFiles()
	if err != nil {
		log.Errorf("Collecting systemd files failed: %v", err)
	}
	log.Info("Successfully collected relevant systemd files")

	log.Info("Starting to collect configuration files")
	msgReturn.Files, err = file.CollectFiles()
	if err != nil {
		log.Errorf("Collecting configuration files failed: %v", err)
	}
	log.Info("Successfully collected relevant configuration files")

	log.Info("Starting to collect command lines")
	msgReturn.CommandLines, err = command.RetrieveCommands()
	if err != nil {
		log.Errorf("Collecting command lines failed: %v", err)
	}
	log.Info("Successfully collected relevant command lines")

	msgReturn.Time = types.TimestampNow()

	// Communicate with sensor, pushing the scraped data.
	if err := retry.WithRetry(
		func() error { // Try to push the data to sensor, time out after 5 seconds.
			log.Info("Trying to push return to sensor")
			ctx, cancel := context.WithTimeout(context.Background(), requestTimeout)
			defer cancel()

			_, err := cli.PushComplianceReturn(ctx, &msgReturn)
			return err
		},
		retry.Tries(5), // 5 attempts.
		retry.BetweenAttempts(func(_ int) { // Sleep for a second between attempts
			log.Info("Sleeping between attempts to post compliance data")
			time.Sleep(time.Second)
		}),
		retry.OnFailedAttempts(func(err error) { // Log encountered errors.
			log.Errorf("Error posting compliance data to %v: %v", env.AdvertisedEndpoint.Setting(), err)
		}),
	); err != nil {
		log.Fatalf("Couldn't post data to sensor despite retries: %v", err)
	}
	log.Info("Successfully pushed data to sensor")

	signalsC := make(chan os.Signal, 1)
	signal.Notify(signalsC, syscall.SIGINT, syscall.SIGTERM)
	// Wait for a signal to terminate
	sig := <-signalsC
	log.Infof("Caught %s signal", sig)
}
