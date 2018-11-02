package main

import (
	"context"
	"flag"
	"fmt"
	"math/rand"
	"time"

	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/clientconn"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/fixtures"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/uuid"
)

var (
	logger           = logging.LoggerForModule()
	deploymentIDBase = uuid.NewV4().String()
	deploymentCount  = 0
)

func main() {
	rand.Seed(time.Now().UnixNano())
	deploymentInterval := flag.Duration("deployment-interval", 100*time.Millisecond, "interval for sending deployments")
	maxDeployments := flag.Int("max-deployments", 10000, "maximum number of deployments to send")
	indicatorInterval := flag.Duration("indicator-interval", 100*time.Millisecond, "interval for sending indicators")
	maxIndicators := flag.Int("max-indicators", 20000, "maximum number of indicators to send")
	centralEndpoint := flag.String("central", "central.stackrox:443", "central endpoint")
	bigDepRate := flag.Float64("big-dep-rate", 0.01, "fraction of giant deployments to send")
	flag.Parse()

	conn, err := clientconn.GRPCConnection(*centralEndpoint)
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

	deploymentSignal := concurrency.NewSignal()
	go sendDeployments(stream, *maxDeployments, *deploymentInterval, *bigDepRate, deploymentSignal)

	indicatorSignal := concurrency.NewSignal()
	go sendIndicators(stream, *maxIndicators, *indicatorInterval, indicatorSignal)

	deploymentSignal.Wait()
	indicatorSignal.Wait()
	logger.Infof("All sending done. The mock sensor will now just sleep forever.")
	time.Sleep(365 * 24 * time.Hour)
}

func getDeploymentID() string {
	return fmt.Sprintf("%s%d", deploymentIDBase, deploymentCount)
}

func deploymentSensorEvent(bigDepRate float64) *v1.SensorEvent {
	var deployment *v1.Deployment
	if rand.Float64() < bigDepRate {
		deployment = fixtures.GetDeployment()
	} else {
		deployment = fixtures.LightweightDeployment()
	}
	id := getDeploymentID()
	deployment.Id = id
	deployment.Name = fmt.Sprintf("nginx%d", deploymentCount)
	return &v1.SensorEvent{
		Id:     id,
		Action: v1.ResourceAction_CREATE_RESOURCE,
		Resource: &v1.SensorEvent_Deployment{
			Deployment: deployment,
		},
	}
}

func sendDeployments(stream v1.SensorEventService_RecordEventClient, maxDeployments int, deploymentInterval time.Duration, bigDepRate float64, signal concurrency.Signal) {
	ticker := time.NewTicker(deploymentInterval)
	defer ticker.Stop()

	for deploymentCount < maxDeployments {
		<-ticker.C
		if err := stream.Send(deploymentSensorEvent(bigDepRate)); err != nil {
			logger.Errorf("Error: %v", err)
		}
		deploymentCount++
	}
	logger.Infof("Finished writing %d deployments", deploymentCount)
	signal.Signal()
}

func sensorEventFromIndicator(index int, indicator *v1.ProcessIndicator) *v1.SensorEvent {
	indicator.Id = uuid.NewV4().String()
	indicator.DeploymentId = getDeploymentID()
	indicator.Signal.ContainerId = getDeploymentID()
	indicator.Signal.ExecFilePath = fmt.Sprintf("EXECFILE%d", index)
	return &v1.SensorEvent{
		Id:     indicator.GetId(),
		Action: v1.ResourceAction_CREATE_RESOURCE,
		Resource: &v1.SensorEvent_ProcessIndicator{
			ProcessIndicator: indicator,
		},
	}
}

func sendIndicators(stream v1.SensorEventService_RecordEventClient, maxIndicators int, indicatorInterval time.Duration, signal concurrency.Signal) {
	ticker := time.NewTicker(indicatorInterval)
	defer ticker.Stop()
	for indicatorCount := 0; indicatorCount < maxIndicators; indicatorCount++ {
		<-ticker.C
		if err := stream.Send(sensorEventFromIndicator(indicatorCount, fixtures.GetProcessIndicator())); err != nil {
			logger.Errorf("Error: %v", err)
		}
	}
	logger.Infof("Finished writing %d indicators", maxIndicators)
	signal.Signal()
}
