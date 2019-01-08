package main

import (
	"context"
	"time"

	"github.com/stackrox/rox/compliance/collection/command"
	"github.com/stackrox/rox/compliance/collection/docker"
	file2 "github.com/stackrox/rox/compliance/collection/file"
	"github.com/stackrox/rox/generated/internalapi/compliance"
	"github.com/stackrox/rox/generated/internalapi/sensor"
	"github.com/stackrox/rox/pkg/clientconn"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/retry"
)

var (
	log = logging.LoggerForModule()
)

const requestTimeout = time.Second * 5

func main() {
	var msgReturn compliance.ComplianceReturn
	var err error

	msgReturn.DockerData, err = docker.GetDockerData()
	if err != nil {
		log.Error(err)
	}

	msgReturn.Files, err = file2.CollectFiles()
	if err != nil {
		log.Error(err)
	}

	msgReturn.CommandLines, err = command.RetrieveCommands()
	if err != nil {
		log.Error(err)
	}

	// Create a connection with sensor to push scraped data.
	conn, err := clientconn.UnauthenticatedGRPCConnection(env.AdvertisedEndpoint.Setting())
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
