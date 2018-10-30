package main

import (
	"context"
	"flag"
	"fmt"
	"time"

	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/clientconn"
	"github.com/stackrox/rox/pkg/fixtures"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/uuid"
)

var log = logging.LoggerForModule()

func main() {
	deploymentInterval := flag.Duration("deployment-interval", 100*time.Millisecond, "interval for sending deployments")
	maxDeployments := flag.Int("max-deployments", 20000, "maximum number of deployments to send")
	centralEndpoint := flag.String("central", "central.stackrox:443", "central endpoint")
	flag.Parse()

	sendDeployments(*centralEndpoint, *maxDeployments, *deploymentInterval)
}

func sendDeployments(centralEndpoint string, maxDeployments int, deploymentInterval time.Duration) {
	conn, err := clientconn.GRPCConnection(centralEndpoint)
	if err != nil {
		panic(err)
	}

	client := v1.NewSensorEventServiceClient(conn)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	stream, err := client.RecordEvent(ctx)
	if err != nil {
		panic(err)
	}
	defer stream.CloseSend()

	deployment := fixtures.GetDeployment()
	sensorEvent := &v1.SensorEvent{
		Action: v1.ResourceAction_CREATE_RESOURCE,
		Resource: &v1.SensorEvent_Deployment{
			Deployment: deployment,
		},
	}

	ticker := time.NewTicker(deploymentInterval)
	var deploymentCount int
	for deploymentCount != maxDeployments {
		<-ticker.C
		id := uuid.NewV4().String()
		sensorEvent.Id = id
		deployment.Id = id
		deployment.Name = fmt.Sprintf("nginx%d", deploymentCount)
		if err := stream.Send(sensorEvent); err != nil {
			log.Errorf("Error: %v", err)
		}
		deploymentCount++
	}
	log.Infof("Finished writing %d deployments", deploymentCount)
}
