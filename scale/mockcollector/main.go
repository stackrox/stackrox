package main

import (
	"context"
	"flag"
	"fmt"
	"math/rand"
	"os/exec"
	"strings"
	"time"

	"github.com/pkg/errors"
	v1 "github.com/stackrox/stackrox/generated/api/v1"
	"github.com/stackrox/stackrox/generated/internalapi/sensor"
	"github.com/stackrox/stackrox/generated/storage"
	"github.com/stackrox/stackrox/pkg/clientconn"
	"github.com/stackrox/stackrox/pkg/logging"
	"github.com/stackrox/stackrox/pkg/mtls"
	"github.com/stackrox/stackrox/pkg/sync"
	"github.com/stackrox/stackrox/pkg/uuid"
)

var (
	log         = logging.LoggerForModule()
	processList = []string{"apt-get", "bash", "sh", "bootstrap.sh", "curl", "ls", "cat", "sudo"}
	processSize = 1000000
)

func main() {
	maxCollectors := flag.Int("max-collectors", 1000, "maximum number of collectors to spawn")
	processInterval := flag.Duration("process-interval", 500*time.Millisecond, "interval for sending process signals")
	maxProcesses := flag.Int("max-processes", 100000, "maximum number of processes to send")
	sensorEndpoint := flag.String("sensor", "sensor.stackrox:443", "sensor endpoint")
	flag.Parse()

	containerID, err := getShortID()
	if err != nil {
		log.Panic(err)
	}

	var wg sync.WaitGroup
	for i := 0; i < *maxCollectors; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			collector(*sensorEndpoint, *maxProcesses, *processInterval, containerID)
		}()
	}
	wg.Wait()

	time.Sleep(365 * 24 * time.Hour)
}

// Returns the container ID of the mockcollector
func getShortID() (string, error) {
	cmd := exec.Command("cat", "/proc/self/cgroup")
	result, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Println(err)
		log.Error(err)
	}
	cgroup := string(result)
	lines := strings.Split(cgroup, "\n")
	for _, line := range lines {
		if strings.Contains(line, "name=systemd:/kubepods") {
			kubepodLines := strings.Split(line, "/")
			containerID := kubepodLines[len(kubepodLines)-1]
			shortID := containerID[0:12]
			return shortID, nil
		}
	}

	return "", errors.New("No containerID found in /proc/self/cgroup")
}

func getStream(sensorEndpoint string) (sensor.SignalService_PushSignalsClient, context.CancelFunc, error) {
	conn, err := clientconn.AuthenticatedGRPCConnection(sensorEndpoint, mtls.SensorSubject)
	if err != nil {
		log.Panic(err)
	}

	client := sensor.NewSignalServiceClient(conn)
	ctx, cancel := context.WithCancel(context.Background())

	stream, err := client.PushSignals(ctx)
	if err != nil {
		log.Panic(err)
	}

	return stream, cancel, nil
}

// This is an instance of collector since it establishes a new gRPC connection with sensor.
func collector(sensorEndpoint string, maxProcesses int, processInterval time.Duration, containerID string) {
	stream, cancel, err := getStream(sensorEndpoint)
	if err != nil {
		log.Panic(err)
	}
	defer cancel()
	ticker := time.NewTicker(processInterval)
	var processCount int
	for processCount != maxProcesses {
		<-ticker.C
		streamMsg := generateSignals(containerID)
		if err := stream.Send(streamMsg); err != nil {
			log.Errorf("Error: %v", err)
			_ = stream.CloseSend()
			time.Sleep(time.Second * 2)
			stream, cancel, err = getStream(sensorEndpoint)
			if err != nil {
				log.Panic(err)
			}
			defer cancel()
			continue
		}
		processCount++
	}
	log.Infof("Successfully sent %d process indicators\n", processCount)
}

func generateSignals(containerID string) *sensor.SignalStreamMessage {
	processListSize := len(processList)
	processSignal := storage.ProcessSignal{
		Id:           uuid.NewV4().String(),
		ContainerId:  containerID,
		Name:         fmt.Sprintf("%s-%d", processList[rand.Int()%(processListSize-1)], rand.Int()%processSize),
		ExecFilePath: fmt.Sprintf("/bin/%s-%d", processList[rand.Int()%(processListSize-1)], rand.Int()%processSize),
	}

	signal := v1.Signal{
		Signal: &v1.Signal_ProcessSignal{
			ProcessSignal: &processSignal,
		},
	}

	signalStreamMessage := &sensor.SignalStreamMessage{
		Msg: &sensor.SignalStreamMessage_Signal{
			Signal: &signal,
		},
	}

	return signalStreamMessage
}
