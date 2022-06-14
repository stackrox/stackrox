package main

import (
	"context"
	"flag"
	"fmt"
	"math"
	"math/rand"
	"strings"
	"sync/atomic"
	"time"

	v1 "github.com/stackrox/stackrox/generated/api/v1"
	"github.com/stackrox/stackrox/generated/internalapi/central"
	"github.com/stackrox/stackrox/generated/storage"
	"github.com/stackrox/stackrox/pkg/centralsensor"
	"github.com/stackrox/stackrox/pkg/clientconn"
	"github.com/stackrox/stackrox/pkg/concurrency"
	"github.com/stackrox/stackrox/pkg/fixtures"
	"github.com/stackrox/stackrox/pkg/images/defaults"
	"github.com/stackrox/stackrox/pkg/logging"
	"github.com/stackrox/stackrox/pkg/mtls"
	"github.com/stackrox/stackrox/pkg/networkgraph"
	"github.com/stackrox/stackrox/pkg/timestamp"
	"github.com/stackrox/stackrox/pkg/uuid"
)

var (
	logger           = logging.LoggerForModule()
	deploymentIDBase = uuid.NewV4().String()
	deploymentCount  = int64(0)
	admissionCount   = int64(0)
)

func getDeploymentCount() int {
	return int(atomic.LoadInt64(&deploymentCount))
}

func incrementDeploymentCount() {
	atomic.AddInt64(&deploymentCount, 1)
}

func getAdmissionCount() int {
	return int(atomic.LoadInt64(&admissionCount))
}

func incrementAndGetAdmissionCount() int {
	return int(atomic.AddInt64(&admissionCount, 1))
}

func main() {
	rand.Seed(time.Now().UnixNano())
	deploymentInterval := flag.Duration("deployment-interval", 2000*time.Millisecond, "interval for sending deployments")
	maxDeployments := flag.Int("max-deployments", 200, "maximum number of deployments to send")
	useImagesFromList := flag.Bool("use-images-from-list", false, "generate deployments with different image names from a list of known image names")

	admissionInterval := flag.Duration("admission-interval", 2000*time.Millisecond, "interval for sending admission controller requests")
	maxAdmissionRequests := flag.Int("max-admission-requests", 0, "maximum number of admission controller requests to send")

	deploymentUpdateInterval := flag.Duration("deployment-update-interval", 2000*time.Millisecond, "interval for sending deployment updates")
	maxDeploymentUpdates := flag.Int("max-deployment-updates", 0, "maximum number of deployment updates to send")

	indicatorInterval := flag.Duration("indicator-interval", 1000*time.Millisecond, "interval for sending indicators")
	maxIndicators := flag.Int("max-indicators", 5000, "maximum number of indicators to send")

	networkFlowInterval := flag.Duration("network-flow-interval", 30*time.Second, "interval for sending network flow diffs")
	maxNetworkFlows := flag.Int("max-network-flows", 1000, "maximum number of network flows to send at once")
	maxUpdates := flag.Int("max-updates", 40, "total number of network flows updates to send")
	flowDeleteRate := flag.Float64("flow-delete-rate", 0.03, "fraction of flows that will be marked removed in each network flow update")

	centralEndpoint := flag.String("central", "central.stackrox:443", "central endpoint")
	bigDepRate := flag.Float64("big-dep-rate", 0.01, "fraction of giant deployments to send")

	maxNodes := flag.Int("max-nodes", 500, "total number of nodes to send")
	nodeInterval := flag.Duration("node-interval", 10*time.Millisecond, "interval for sending nodes")

	flag.Parse()

	if *maxNetworkFlows > int(math.Pow(float64(*maxDeployments), 2)) {
		logger.Fatal("Unable to generate specified flows. Increase maxDeployments or decrease maxNetworkFlows")
	}

	conn, err := clientconn.AuthenticatedGRPCConnection(*centralEndpoint, mtls.CentralSubject, clientconn.UseServiceCertToken(true))
	if err != nil {
		panic(err)
	}

	client := central.NewSensorServiceClient(conn)
	admissionClient := v1.NewDetectionServiceClient(conn)
	clusterClient := v1.NewClustersServiceClient(conn)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	capsSet := centralsensor.NewSensorCapabilitySet(centralsensor.ComplianceInNodesCap)
	sensorHello := &central.SensorHello{
		Capabilities: centralsensor.CapSetToStringSlice(capsSet),
	}
	ctx, err = centralsensor.AppendSensorHelloInfoToOutgoingMetadata(ctx, sensorHello)
	if err != nil {
		panic(err)
	}
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

	go stream.StartReceiving()

	deploymentSignal := concurrency.NewSignal()
	go sendDeployments(stream, *maxDeployments, *deploymentInterval, *bigDepRate, *useImagesFromList, deploymentSignal)

	time.Sleep(5 * time.Second)

	indicatorSignal := concurrency.NewSignal()
	go sendIndicators(stream, *maxIndicators, *indicatorInterval, indicatorSignal)

	networkFlowSignal := concurrency.NewSignal()

	go sendNetworkFlows(stream, *networkFlowInterval, *maxNetworkFlows, *maxUpdates, *flowDeleteRate, &networkFlowSignal)

	admissionControllerSignal := concurrency.NewSignal()

	go sendAdmissionControllerRequests(ctx, clusterClient, admissionClient, *admissionInterval, *maxAdmissionRequests, &admissionControllerSignal)

	deploymentUpdateSignal := concurrency.NewSignal()
	go sendDeploymentUpdates(stream, *deploymentUpdateInterval, *maxDeploymentUpdates, &deploymentUpdateSignal)

	nodeSignal := concurrency.NewSignal()
	go sendNodes(stream, *maxNodes, *nodeInterval, nodeSignal)

	nodeSignal.Wait()
	deploymentSignal.Wait()
	indicatorSignal.Wait()
	networkFlowSignal.Wait()
	admissionControllerSignal.Wait()
	deploymentUpdateSignal.Wait()

	logger.Info("All sending done. The mock sensor will now just sleep forever.")
	time.Sleep(365 * 24 * time.Hour)
}

func getDeploymentID() string {
	return getGeneratedDeploymentID(getDeploymentCount())
}

func getGeneratedDeploymentID(i int) string {
	return fmt.Sprintf("%s%d", deploymentIDBase, i)
}

func deploymentSensorEvent(bigDepRate float64, useImagesFromList bool) *central.SensorEvent {
	var deployment *storage.Deployment
	if rand.Float64() < bigDepRate {
		deployment = fixtures.GetDeployment()
	} else {
		deployment = fixtures.LightweightDeployment()
	}
	if useImagesFromList {
		replaceImages(deployment)
	}
	return deploymentSensorEventForNum(getDeploymentCount(), deployment)
}

func deploymentSensorEventForNum(depNum int, deployment *storage.Deployment) *central.SensorEvent {
	id := getGeneratedDeploymentID(depNum)
	deployment.Id = id
	deployment.Name = fmt.Sprintf("nginx%d", depNum)
	return &central.SensorEvent{
		Id:     id,
		Action: central.ResourceAction_CREATE_RESOURCE,
		Resource: &central.SensorEvent_Deployment{
			Deployment: deployment,
		},
	}
}

func replaceImages(deployment *storage.Deployment) {
	for _, container := range deployment.GetContainers() {
		container.Image = knownDockerContainerImage()
	}
}

func knownDockerContainerImage() *storage.ContainerImage {
	nameAndID := fixtures.GetRandomImage()
	return &storage.ContainerImage{
		Id: nameAndID.ID,
		Name: &storage.ImageName{
			Registry: "docker.io",
			Remote:   nameAndID.Name,
			Tag:      "latest",
			FullName: fmt.Sprintf("docker.io/%s", nameAndID.Name),
		},
	}
}

func nodeSensorEvent(name string) *central.SensorEvent {
	id := uuid.NewV4().String()
	return &central.SensorEvent{
		Id:     id,
		Action: central.ResourceAction_CREATE_RESOURCE,
		Resource: &central.SensorEvent_Node{
			Node: &storage.Node{
				Id:   id,
				Name: name,
				ContainerRuntime: &storage.ContainerRuntimeInfo{
					Type: storage.ContainerRuntime_DOCKER_CONTAINER_RUNTIME,
				},
			},
		},
	}
}

func sendNodes(stream *threadSafeStream, maxNodes int, nodeInterval time.Duration, signal concurrency.Signal) {
	ticker := time.NewTicker(nodeInterval)
	defer ticker.Stop()

	var nodeCount int
	for nodeCount < maxNodes {
		<-ticker.C
		nodeName := fmt.Sprintf("node-%d", nodeCount)
		if err := stream.SendEvent(nodeSensorEvent(nodeName)); err != nil {
			logger.Errorf("Error: %v", err)
		}
		nodeCount++
	}
	logger.Infof("Finished writing %d nodes", nodeCount)
	signal.Signal()
}

func sendDeployments(stream *threadSafeStream, maxDeployments int, deploymentInterval time.Duration, bigDepRate float64, useImagesFromList bool, signal concurrency.Signal) {
	ticker := time.NewTicker(deploymentInterval)
	defer ticker.Stop()

	for getDeploymentCount() < maxDeployments {
		<-ticker.C
		if err := stream.SendEvent(deploymentSensorEvent(bigDepRate, useImagesFromList)); err != nil {
			logger.Errorf("Error: %v", err)
		}
		incrementDeploymentCount()
	}
	logger.Infof("Finished writing %d deployments", getDeploymentCount())
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
		time.Sleep(time.Until(nextTick))
		update := generateNetworkFlowUpdate(maxNetworkFlows, flowDeleteRate)
		if update == nil {
			continue
		}
		if err := stream.SendNetworkFlows(update); err != nil {
			logger.Errorf("Error: %v", err)
		}
		logger.Infof("Finished sending update %d with %d network flows", u, len(update.Updated))
		nextTick = nextTick.Add(networkFlowInterval)
	}
}

func generateNetworkFlowUpdate(maxNetworkFlows int, flowDeleteRate float64) *central.NetworkFlowUpdate {
	deploymentCount := getDeploymentCount()
	if deploymentCount < 2 {
		return nil
	}
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
				SrcEntity:  networkgraph.EntityForDeployment(getGeneratedDeploymentID(srcIndex)).ToProto(),
				DstEntity:  networkgraph.EntityForDeployment(getGeneratedDeploymentID(dstIndex)).ToProto(),
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

func sendAdmissionControllerRequests(ctx context.Context, clusterClient v1.ClustersServiceClient, admissionClient v1.DetectionServiceClient, admissionInterval time.Duration, maxAdmissionRequests int, signal *concurrency.Signal) {
	defer signal.Signal()
	ticker := time.NewTicker(admissionInterval)
	defer ticker.Stop()

	var clusterID string
	if maxAdmissionRequests > 0 {
		admissionControllerConfig := &storage.AdmissionControllerConfig{}
		admissionControllerConfig.Enabled = true
		admissionControllerConfig.TimeoutSeconds = 999
		flavor := defaults.GetImageFlavorFromEnv()
		cluster := &storage.Cluster{
			Id:                  "",
			Name:                "prod cluster",
			Type:                1,
			MainImage:           flavor.MainImageNoTag(),
			CollectorImage:      "",
			CentralApiEndpoint:  "central.stackrox:443",
			CollectionMethod:    0,
			AdmissionController: true,
			Status:              nil,
			DynamicConfig: &storage.DynamicClusterConfig{
				AdmissionControllerConfig: admissionControllerConfig,
			},
		}
		response, err := clusterClient.PostCluster(ctx, cluster)
		if err != nil && !strings.Contains(err.Error(), "AlreadyExists") {
			logging.Fatal(err)
		}
		clusterID = response.GetCluster().GetId()
	}

	wg := concurrency.NewWaitGroup(maxAdmissionRequests)
	start := time.Now()
	for getAdmissionCount() <= maxAdmissionRequests {
		<-ticker.C
		admissionNum := incrementAndGetAdmissionCount()
		go func() {
			defer wg.Add(-1)
			if _, err := admissionClient.DetectDeployTime(ctx, getDeployDetectionRequest(admissionNum, clusterID)); err != nil {
				logger.Errorf("Error: %v", err)
			}
		}()
	}
	<-wg.Done()
	logger.Infof("Finished writing %d admission controller requests in %s", getAdmissionCount(), time.Since(start))
}

func getDeployDetectionRequest(reqNum int, clusterID string) *v1.DeployDetectionRequest {
	deployment := fixtures.LightweightDeployment()
	id := getGeneratedDeploymentID(reqNum)
	deployment.Id = id
	deployment.Name = fmt.Sprintf("nginx%d", reqNum)
	deployment.ClusterId = clusterID
	return &v1.DeployDetectionRequest{
		Resource: &v1.DeployDetectionRequest_Deployment{
			Deployment: deployment,
		},
		NoExternalMetadata: true,
		EnforcementOnly:    true,
		ClusterId:          clusterID,
	}
}

func sendDeploymentUpdates(stream *threadSafeStream, updateInterval time.Duration, maxUpdates int, signal *concurrency.Signal) {
	defer signal.Signal()
	ticker := time.NewTicker(updateInterval)
	defer ticker.Stop()

	for i := 0; i < maxUpdates; i++ {
		<-ticker.C
		deployment := fixtures.LightweightDeployment()
		deployment.Labels = map[string]string{uuid.NewV4().String(): uuid.NewV4().String()}
		numDeps := getDeploymentCount()
		if numDeps == 0 {
			// Avoid divide by zero.
			numDeps = 1
		}
		if err := stream.SendEvent(deploymentSensorEventForNum(rand.Int()%numDeps, deployment)); err != nil {
			logger.Errorf("Error sending deployment update: %v", err)
		}
	}
	logger.Infof("Finished writing %d deployment updates", maxUpdates)
}
