package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gogo/protobuf/types"
	"github.com/stackrox/rox/compliance/collection/command"
	"github.com/stackrox/rox/compliance/collection/docker"
	"github.com/stackrox/rox/compliance/collection/file"
	"github.com/stackrox/rox/generated/internalapi/compliance"
	"github.com/stackrox/rox/generated/internalapi/sensor"
	"github.com/stackrox/rox/pkg/clientconn"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/mtls"
	"github.com/stackrox/rox/pkg/orchestrators"
	"github.com/stackrox/rox/pkg/retry"
)

var (
	log = logging.LoggerForModule()
)

const requestTimeout = time.Second * 5

func main() {
	thisNodeName := os.Getenv(string(orchestrators.NodeName))
	if thisNodeName == "" {
		log.Fatal("No node name found in the environment")
	}
	thisScrapeID := os.Getenv("ROX_SCRAPE_ID")
	if thisScrapeID == "" {
		log.Fatal("No scrape ID found in the environment")
	}
	msgReturn := compliance.ComplianceReturn{
		NodeName: thisNodeName,
		ScrapeId: thisScrapeID,
	}

	log.Infof("Running compliance scrape %q for node %q", thisScrapeID, thisNodeName)

	log.Infof("Starting to collect Docker data")
	var err error
	msgReturn.DockerData, err = docker.GetDockerData()
	if err != nil {
		log.Error(err)
	}

	log.Infof("Successfully collected relevant Docker data")

	log.Infof("Starting to collect systemd files")
	msgReturn.SystemdFiles, err = file.CollectSystemdFiles()
	if err != nil {
		log.Error(err)
	}
	log.Infof("Successfully collected relevant systemd files")

	log.Infof("Starting to collect configuration files")
	msgReturn.Files, err = file.CollectFiles()
	if err != nil {
		log.Error(err)
	}
	log.Infof("Successfully collected relevant configuration files")

	log.Infof("Starting to collect command lines")
	msgReturn.CommandLines, err = command.RetrieveCommands()
	if err != nil {
		log.Error(err)
	}
	log.Infof("Successfully collected relevant command lines")

	msgReturn.Time = types.TimestampNow()

	// Create a connection with sensor to push scraped data.
	conn, err := clientconn.AuthenticatedGRPCConnection(env.AdvertisedEndpoint.Setting(), mtls.SensorSubject)
	if err != nil {
		log.Fatal(err)
	}
	log.Infof("Initialized Sensor gRPC connection")
	defer func() {
		if err := conn.Close(); err != nil {
			log.Errorf("Failed to close connection: %v", err)
		}
	}()
	cli := sensor.NewComplianceServiceClient(conn)

	// Communicate with sensor, pushing the scraped data.
	if err := retry.WithRetry(
		func() error { // Try to push the data to sensor, time out after 5 seconds.
			log.Infof("Trying to push return to sensor")
			ctx, cancel := context.WithTimeout(context.Background(), requestTimeout)
			defer cancel()

			_, err := cli.PushComplianceReturn(ctx, &msgReturn)
			return err
		},
		retry.Tries(5), // 5 attempts.
		retry.BetweenAttempts(func() { // Sleep for a second between attempts
			log.Info("Sleeping between attempts to post compliance data")
			time.Sleep(time.Second)
		}),
		retry.OnFailedAttempts(func(err error) { // Log encountered errors.
			log.Errorf("Error posting compliance data to %v: %v", env.AdvertisedEndpoint.Setting(), err)
		}),
	); err != nil {
		log.Fatalf("Couldn't post data to sensor despite retries: %v", err)
	}
	log.Infof("Successfully pushed data to sensor")

	signalsC := make(chan os.Signal, 1)
	signal.Notify(signalsC, syscall.SIGINT, syscall.SIGTERM)
	// Wait for a signal to terminate
	sig := <-signalsC
	log.Infof("Caught %s signal", sig)
}
