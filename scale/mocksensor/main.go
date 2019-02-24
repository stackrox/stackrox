package main

import (
	"context"
	"flag"
	"fmt"
	"math"
	"math/rand"
	"time"

	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/clientconn"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/fixtures"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/networkentity"
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
	deploymentInterval := flag.Duration("deployment-interval", 2000*time.Millisecond, "interval for sending deployments")
	maxDeployments := flag.Int("max-deployments", 200, "maximum number of deployments to send")
	indicatorInterval := flag.Duration("indicator-interval", 1000*time.Millisecond, "interval for sending indicators")
	maxIndicators := flag.Int("max-indicators", 5000, "maximum number of indicators to send")
	networkFlowInterval := flag.Duration("network-flow-interval", 30*time.Second, "interval for sending network flow diffs")
	maxNetworkFlows := flag.Int("max-network-flows", 1000, "maximum number of network flows to send at once")
	maxUpdates := flag.Int("max-updates", 40, "total number of network flows updates to send")
	flowDeleteRate := flag.Float64("flow-delete-rate", 0.03, "fraction of flows that will be marked removed in each network flow update")
	centralEndpoint := flag.String("central", "central.stackrox:443", "central endpoint")
	bigDepRate := flag.Float64("big-dep-rate", 0.01, "fraction of giant deployments to send")
	flag.Parse()

	if *maxNetworkFlows > int(math.Pow(float64(*maxDeployments), 2)) {
		logger.Fatalf("Unable to generate specified flows. Increase maxDeployments or decrease maxNetworkFlows")
	}

	conn, err := clientconn.AuthenticatedGRPCConnection(*centralEndpoint, clientconn.Central)
	if err != nil {
		panic(err)
	}

	client := central.NewSensorServiceClient(conn)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	communicateStream, err := client.Communicate(ctx)
	if err != nil {
		panic(err)
	}
	defer func() {
		if err := communicateStream.CloseSend(); err != nil {
			logger.Errorf("Failed to close communication stream: %v", err)
		}
	}()

	stream := &threadSafeStream{
		stream: communicateStream,
	}

	deploymentSignal := concurrency.NewSignal()
	go sendDeployments(stream, *maxDeployments, *deploymentInterval, *bigDepRate, deploymentSignal)

	time.Sleep(5 * time.Second)

	indicatorSignal := concurrency.NewSignal()
	go sendIndicators(stream, *maxIndicators, *indicatorInterval, indicatorSignal)

	networkFlowSignal := concurrency.NewSignal()

	go sendNetworkFlows(stream, *networkFlowInterval, *maxNetworkFlows, *maxUpdates, *flowDeleteRate, &networkFlowSignal)

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

func deploymentSensorEvent(bigDepRate float64) *central.SensorEvent {
	var deployment *storage.Deployment
	if rand.Float64() < bigDepRate {
		deployment = fixtures.GetDeployment()
	} else {
		deployment = fixtures.LightweightDeployment()
	}
	id := getDeploymentID()
	deployment.Id = id
	deployment.Name = fmt.Sprintf("nginx%d", deploymentCount)
	return &central.SensorEvent{
		Id:     id,
		Action: central.ResourceAction_CREATE_RESOURCE,
		Resource: &central.SensorEvent_Deployment{
			Deployment: deployment,
		},
	}
}

func sendDeployments(stream *threadSafeStream, maxDeployments int, deploymentInterval time.Duration, bigDepRate float64, signal concurrency.Signal) {
	ticker := time.NewTicker(deploymentInterval)
	defer ticker.Stop()

	for deploymentCount < maxDeployments {
		<-ticker.C
		if err := stream.SendEvent(deploymentSensorEvent(bigDepRate)); err != nil {
			logger.Errorf("Error: %v", err)
		}
		deploymentCount++
	}
	logger.Infof("Finished writing %d deployments", deploymentCount)
	signal.Signal()
}

func sensorEventFromIndicator(index int, indicator *storage.ProcessIndicator) *central.SensorEvent {
	indicator.Id = uuid.NewV4().String()
	indicator.DeploymentId = getDeploymentID()
	indicator.Signal.ContainerId = getDeploymentID()
	indicator.Signal.ExecFilePath = fmt.Sprintf("EXECFILE%d", index)
	return &central.SensorEvent{
		Id:     indicator.GetId(),
		Action: central.ResourceAction_CREATE_RESOURCE,
		Resource: &central.SensorEvent_ProcessIndicator{
			ProcessIndicator: indicator,
		},
	}
}

func sendIndicators(stream *threadSafeStream, maxIndicators int, indicatorInterval time.Duration, signal concurrency.Signal) {
	ticker := time.NewTicker(indicatorInterval)
	defer ticker.Stop()
	for indicatorCount := 0; indicatorCount < maxIndicators; indicatorCount++ {
		<-ticker.C
		if err := stream.SendEvent(sensorEventFromIndicator(indicatorCount, fixtures.GetProcessIndicator())); err != nil {
			logger.Errorf("Error: %v", err)
		}
	}
	logger.Infof("Finished writing %d indicators", maxIndicators)
	signal.Signal()
}

func sendNetworkFlows(stream *threadSafeStream, networkFlowInterval time.Duration, maxNetworkFlows int, maxUpdates int, flowDeleteRate float64, signal *concurrency.Signal) {
	defer signal.Signal()

	nextTick := time.Now()
	for u := 0; u < maxUpdates; u++ {
		time.Sleep(nextTick.Sub(time.Now()))
		update := generateNetworkFlowUpdate(maxNetworkFlows, flowDeleteRate)
		if err := stream.SendNetworkFlows(update); err != nil {
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

	flows := make([]*storage.NetworkFlow, numFlows)
	for i := range flows {
		srcIndex := rand.Int() % deploymentCount
		dstIndex := rand.Int() % (deploymentCount - 1)
		if dstIndex >= srcIndex {
			dstIndex++
		}

		flow := &storage.NetworkFlow{
			Props: &storage.NetworkFlowProperties{
				SrcEntity:  networkentity.ForDeployment(getGeneratedDeploymentID(srcIndex)).ToProto(),
				DstEntity:  networkentity.ForDeployment(getGeneratedDeploymentID(dstIndex)).ToProto(),
				L4Protocol: storage.L4Protocol_L4_PROTOCOL_TCP,
				DstPort:    80,
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
