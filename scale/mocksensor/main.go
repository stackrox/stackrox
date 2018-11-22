package main

import (
	"context"
	"flag"
	"fmt"
	"math"
	"math/rand"
	"time"

	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/pkg/clientconn"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/fixtures"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/timestamp"
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
	networkFlowInterval := flag.Duration("network-flow-interval", 5*time.Second, "interval for sending network flow diffs")
	maxNetworkFlows := flag.Int("max-network-flows", 1000, "maximum number of network flows to send at once")
	maxUpdates := flag.Int("max-updates", 100, "total number of network flows updates to send")
	flowDeleteRate := flag.Float64("flow-delete-rate", 0.03, "fraction of flows that will be marked removed in each network flow update")
	centralEndpoint := flag.String("central", "central.stackrox:443", "central endpoint")
	bigDepRate := flag.Float64("big-dep-rate", 0.01, "fraction of giant deployments to send")
	flag.Parse()

	if *maxNetworkFlows > int(math.Pow(float64(*maxDeployments), 2)) {
		logger.Fatalf("Unable to generate specified flows. Increase maxDeployments or decrease maxNetworkFlows")
	}

	conn, err := clientconn.GRPCConnection(*centralEndpoint, clientconn.Central)
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

	time.Sleep(5 * time.Second)

	indicatorSignal := concurrency.NewSignal()
	go sendIndicators(stream, *maxIndicators, *indicatorInterval, indicatorSignal)

	nfClient := central.NewNetworkFlowServiceClient(conn)
	nfStream, err := nfClient.PushNetworkFlows(ctx)

	if err != nil {
		panic(err)
	}
	defer nfStream.CloseSend()
	networkFlowSignal := concurrency.NewSignal()

	go sendNetworkFlows(nfStream, *networkFlowInterval, *maxNetworkFlows, *maxUpdates, *flowDeleteRate, &networkFlowSignal)

	deploymentSignal.Wait()
	indicatorSignal.Wait()
	networkFlowSignal.Wait()

	logger.Infof("All sending done. The mock sensor will now just sleep forever.")
	time.Sleep(365 * 24 * time.Hour)
}

func getDeploymentID() string {
	return getGeneratedDeploymentID(deploymentCount)
}

func getGeneratedDeploymentID(i int) string {
	return fmt.Sprintf("%s%d", deploymentIDBase, i)
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

func sendNetworkFlows(stream central.NetworkFlowService_PushNetworkFlowsClient, networkFlowInterval time.Duration, maxNetworkFlows int, maxUpdates int, flowDeleteRate float64, signal *concurrency.Signal) {
	defer signal.Signal()

	nextTick := time.Now()
	for u := 0; u < maxUpdates; u++ {
		time.Sleep(nextTick.Sub(time.Now()))
		update := generateNetworkFlowUpdate(maxNetworkFlows, flowDeleteRate)
		if err := stream.Send(update); err != nil {
			logger.Errorf("Error: %v", err)
		}
		logger.Infof("Finished sending update %d with %d network flows", u, len(update.Updated))
		nextTick = nextTick.Add(networkFlowInterval)
	}
}

func generateNetworkFlowUpdate(maxNetworkFlows int, flowDeleteRate float64) *central.NetworkFlowUpdate {
	numFlows := deploymentCount * deploymentCount
	if numFlows > maxNetworkFlows {
		numFlows = maxNetworkFlows
	}

	flows := make([]*v1.NetworkFlow, numFlows)
	for i := range flows {
		srcIndex := rand.Int() % deploymentCount
		dstIndex := rand.Int() % (deploymentCount - 1)
		if dstIndex >= srcIndex {
			dstIndex++
		}

		flow := &v1.NetworkFlow{
			Props: &v1.NetworkFlowProperties{
				SrcDeploymentId: getGeneratedDeploymentID(srcIndex),
				DstDeploymentId: getGeneratedDeploymentID(dstIndex),
				L4Protocol:      v1.L4Protocol_L4_PROTOCOL_TCP,
				DstPort:         80,
			},
		}

		if rand.Float64() < flowDeleteRate {
			flow.LastSeenTimestamp = timestamp.Now().GogoProtobuf()
		}
		flows[i] = flow
	}

	return &central.NetworkFlowUpdate{
		Updated: flows,
		Time:    timestamp.Now().GogoProtobuf(),
	}
}
