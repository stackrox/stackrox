package main

import (
	"context"
	"os"
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

	var err error
	msgReturn.DockerData, err = docker.GetDockerData()
	if err != nil {
		log.Error(err)
	}

	msgReturn.SystemdFiles, err = file.CollectSystemdFiles()

	msgReturn.Files, err = file.CollectFiles()
	if err != nil {
		log.Error(err)
	}

	msgReturn.CommandLines, err = command.RetrieveCommands()
	if err != nil {
		log.Error(err)
	}

	msgReturn.Time = types.TimestampNow()

	// Create a connection with sensor to push scraped data.
	conn, err := clientconn.AuthenticatedGRPCConnection(env.AdvertisedEndpoint.Setting(), clientconn.Sensor)
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()
	cli := sensor.NewComplianceServiceClient(conn)

	// Communicate with sensor, pushing the scraped data.
	retry.WithRetry(
		func() error { // Try to push the data to sensor, time out after 5 seconds.
			ctx, cancel := context.WithTimeout(context.Background(), requestTimeout)
			_, err := cli.PushComplianceReturn(ctx, &msgReturn)
			cancel()
			return err
		},
		retry.Tries(5), // 5 attempts.
		retry.BetweenAttempts(func() { // Sleep for a second between attempts
			log.Info("Sleeping between attempts to post compliance data")
			time.Sleep(time.Second)
		}),
		retry.OnFailedAttempts(func(err error) { // Log encountered errors.
			log.Errorf("Error posting compliance data to %v: %+v", env.AdvertisedEndpoint.Setting(), err)
		}),
	)
}
