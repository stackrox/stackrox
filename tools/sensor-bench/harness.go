package main

import (
	"context"
	"fmt"
	"net"
	"os"
	"path/filepath"

	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/defaults/policies"
	"github.com/stackrox/rox/pkg/metrics"
	"github.com/stackrox/rox/pkg/utils"
	"github.com/stackrox/rox/sensor/common/clusterid"
	sensorCommon "github.com/stackrox/rox/sensor/common/sensor"
	centralDebug "github.com/stackrox/rox/sensor/debugger/central"
	"github.com/stackrox/rox/sensor/debugger/k8s"
	"github.com/stackrox/rox/sensor/debugger/message"
	sensorKit "github.com/stackrox/rox/sensor/kubernetes/sensor"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/test/bufconn"
	apiextv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	fakeDynamic "k8s.io/client-go/dynamic/fake"
)

type Harness struct {
	FakeCentral *centralDebug.FakeService
	FakeClient  *k8s.ClientSet

	sensorInstance *sensorCommon.Sensor
	grpcServer     *grpc.Server
	listener       *bufconn.Listener
}

func NewHarness(cfg *Config) (*Harness, error) {
	if err := setupCertEnv(); err != nil {
		return nil, fmt.Errorf("setting up cert env: %w", err)
	}

	os.Setenv("ROX_METRICS_PORT", ":9090")
	os.Setenv("ROX_ENABLE_SECURE_METRICS", "false")

	policyList, err := loadPolicies(cfg)
	if err != nil {
		return nil, fmt.Errorf("loading policies: %w", err)
	}

	clusterID := "00000000-0000-4000-A000-000000000000"
	fakeCentral := centralDebug.MakeFakeCentralWithInitialMessages(
		message.SensorHello(clusterID),
		message.ClusterConfig(),
		message.PolicySync(policyList),
		message.BaselineSync([]*storage.ProcessBaseline{}),
		message.NetworkBaselineSync([]*storage.NetworkBaseline{}),
	)

	fakeClient := k8s.MakeFakeClient()
	scheme := runtime.NewScheme()
	apiextv1.AddToScheme(scheme)
	fakeClient.SetDynamic(fakeDynamic.NewSimpleDynamicClientWithCustomListKinds(scheme,
		map[schema.GroupVersionResource]string{
			{Group: "apiextensions.k8s.io", Version: "v1", Resource: "customresourcedefinitions"}: "CustomResourceDefinitionList",
		},
	))

	conn, grpcServer, listener := createGRPCConnection(fakeCentral)
	connFactory := centralDebug.MakeFakeConnectionFactory(conn)

	s, err := sensorKit.CreateSensor(sensorKit.ConfigWithDefaults().
		WithK8sClient(fakeClient).
		WithLocalSensor(true).
		WithCentralConnectionFactory(connFactory).
		WithClusterIDHandler(clusterid.NewHandler()))
	if err != nil {
		grpcServer.Stop()
		return nil, fmt.Errorf("creating sensor: %w", err)
	}

	metrics.NewServer(metrics.SensorSubsystem, metrics.NewTLSConfigurerFromEnv()).RunForever()

	go s.Start()
	fakeCentral.ConnectionStarted.Wait()

	return &Harness{
		FakeCentral:    fakeCentral,
		FakeClient:     fakeClient,
		sensorInstance: s,
		grpcServer:     grpcServer,
		listener:       listener,
	}, nil
}

func (h *Harness) Stop() {
	h.FakeCentral.KillSwitch.Signal()
	h.sensorInstance.Stop()
	h.grpcServer.Stop()
	utils.IgnoreError(h.listener.Close)
}

func createGRPCConnection(fakeCentral *centralDebug.FakeService) (*grpc.ClientConn, *grpc.Server, *bufconn.Listener) {
	buffer := 1024 * 1024
	listener := bufconn.Listen(buffer)

	server := grpc.NewServer()
	central.RegisterSensorServiceServer(server, fakeCentral)

	go func() {
		utils.IgnoreError(func() error {
			return server.Serve(listener)
		})
	}()

	//nolint:staticcheck // DialContext eagerly connects, which is needed for the fake connection factory
	conn, err := grpc.DialContext(context.Background(), "",
		grpc.WithContextDialer(func(ctx context.Context, s string) (net.Conn, error) {
			return listener.Dial()
		}),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		panic(fmt.Sprintf("failed to dial bufconn: %v", err))
	}

	return conn, server, listener
}

func setupCertEnv() error {
	certsDir := filepath.Join("tools", "local-sensor", "certs")
	if _, err := os.Stat(filepath.Join(certsDir, "cert.pem")); err != nil {
		return fmt.Errorf("certs not found at %s: run from the repo root", certsDir)
	}
	os.Setenv("ROX_MTLS_CERT_FILE", filepath.Join(certsDir, "cert.pem"))
	os.Setenv("ROX_MTLS_KEY_FILE", filepath.Join(certsDir, "key.pem"))
	os.Setenv("ROX_MTLS_CA_FILE", filepath.Join(certsDir, "caCert.pem"))
	os.Setenv("ROX_MTLS_CA_KEY_FILE", filepath.Join(certsDir, "caKey.pem"))
	return nil
}

func loadPolicies(cfg *Config) ([]*storage.Policy, error) {
	if !cfg.Policies.UseDefaults {
		return []*storage.Policy{}, nil
	}
	policyList, err := policies.DefaultPolicies()
	if err != nil {
		return nil, fmt.Errorf("loading default policies: %w", err)
	}
	return policyList, nil
}
