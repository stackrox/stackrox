package main

import (
	"context"
	"os"
	"os/signal"

	"github.com/stackrox/rox/pkg/clientconn"
	"github.com/stackrox/rox/pkg/devmode"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/metrics"
	"github.com/stackrox/rox/pkg/mtls"
	"github.com/stackrox/rox/pkg/premain"
	"github.com/stackrox/rox/pkg/utils"
	"github.com/stackrox/rox/pkg/version"
	"github.com/stackrox/rox/sensor/kubernetes/client"
	"github.com/stackrox/rox/sensor/kubernetes/fake"
	"github.com/stackrox/rox/sensor/kubernetes/sensor"
	scannerV1 "github.com/stackrox/scanner/generated/scanner/api/v1"
	"golang.org/x/sys/unix"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

var (
	log = logging.LoggerForModule()
)

func main() {
	premain.StartMain()

	devmode.StartOnDevBuilds("bin/kubernetes-sensor")

	log.Infof("Running StackRox Version: %s", version.GetMainVersion())

	// Start the prometheus metrics server
	metrics.NewDefaultHTTPServer().RunForever()
	metrics.GatherThrottleMetricsForever(metrics.SensorSubsystem.String())

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, os.Interrupt, unix.SIGTERM)

	var sharedClientInterface client.Interface

	// Workload manager is only non-nil when we are mocking out the k8s client
	workloadManager := fake.NewWorkloadManager()
	if workloadManager != nil {
		sharedClientInterface = workloadManager.Client()
	} else {
		sharedClientInterface = client.MustCreateInterface()
	}
	s, err := sensor.CreateSensor(sharedClientInterface, workloadManager)
	utils.CrashOnError(err)

	tlsConfig, err := clientconn.TLSConfig(mtls.ScannerSubject, clientconn.TLSConfigOptions{
		UseClientCert: clientconn.MustUseClientCert,
	})
	if err != nil {
		log.Error("Creating Scanner TLS Config")
	}
	conn, err := grpc.Dial(env.ScannerEndpoint.Setting(), grpc.WithTransportCredentials(credentials.NewTLS(tlsConfig)))
	if err != nil {
		log.Errorf("Dialing scanner: %v", err)
	}
	pingSvc := scannerV1.NewPingServiceClient(conn)
	resp, err := pingSvc.Ping(context.Background(), new(scannerV1.Empty))
	log.Errorf("Resp from Scanner ping: %v, Error: %v", resp, err)
	scanSvc := scannerV1.NewImageScanServiceClient(conn)
	resp2, err := scanSvc.GetImageComponents(context.Background(), new(scannerV1.GetImageComponentsRequest))
	log.Errorf("Resp from Scanner: %v, Error: %v", resp2, err)

	s.Start()

	for {
		select {
		case sig := <-sigs:
			log.Infof("Caught %s signal", sig)
			s.Stop()
		case <-s.Stopped().Done():
			if err := s.Stopped().Err(); err != nil {
				log.Fatalf("Sensor exited with error: %v", err)
			} else {
				log.Info("Sensor exited normally")
			}
			return
		}
	}
}
