package service

import (
	"context"
	"sync"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/vsock-listener/k8s"
	"github.com/stackrox/rox/vsock-listener/relay"
	"github.com/stackrox/rox/vsock-listener/vsock"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

const (
	// VSOCK_PORT is the port used for VSOCK communication
	// Using port 818 (unassigned) to avoid conflicts with system services
	VSOCK_PORT = 818
)

var (
	log = logging.LoggerForModule()
)

// VSockListener manages VSOCK communication with VM agents
type VSockListener struct {
	ctx           context.Context
	cancel        context.CancelFunc
	stopper       concurrency.Stopper
	k8sClient     kubernetes.Interface
	dynamicClient dynamic.Interface
	vmWatcher     *k8s.VMWatcher
	vsockSrv      *vsock.Server
	relay         *relay.SensorRelay
	wg            sync.WaitGroup
}

// NewVSockListener creates a new VSOCK listener service
func NewVSockListener(ctx context.Context) (*VSockListener, error) {
	ctx, cancel := context.WithCancel(ctx)

	// Create Kubernetes client
	config, err := rest.InClusterConfig()
	if err != nil {
		cancel()
		return nil, errors.Wrap(err, "failed to create in-cluster config")
	}

	k8sClient, err := kubernetes.NewForConfig(config)
	if err != nil {
		cancel()
		return nil, errors.Wrap(err, "failed to create Kubernetes client")
	}

	// Create dynamic client
	dynamicClient, err := dynamic.NewForConfig(config)
	if err != nil {
		cancel()
		return nil, errors.Wrap(err, "failed to create dynamic client")
	}

	// Create VM watcher using dynamic client
	vmWatcher, err := k8s.NewVMWatcher(ctx, dynamicClient)
	if err != nil {
		cancel()
		return nil, errors.Wrap(err, "failed to create VM watcher")
	}

	// Create sensor relay
	sensorEndpoint := env.SensorEndpoint.Setting()
	if sensorEndpoint == "" {
		sensorEndpoint = "sensor:443"
	}

	relay, err := relay.NewSensorRelay(ctx, sensorEndpoint)
	if err != nil {
		cancel()
		return nil, errors.Wrap(err, "failed to create sensor relay")
	}

	// Create VSOCK server
	vsockSrv, err := vsock.NewServer(ctx, VSOCK_PORT, vmWatcher, relay)
	if err != nil {
		cancel()
		return nil, errors.Wrap(err, "failed to create VSOCK server")
	}

	return &VSockListener{
		ctx:           ctx,
		cancel:        cancel,
		stopper:       concurrency.NewStopper(),
		k8sClient:     k8sClient,
		dynamicClient: dynamicClient,
		vmWatcher:     vmWatcher,
		vsockSrv:      vsockSrv,
		relay:         relay,
	}, nil
}

// Start starts the VSOCK listener service
func (v *VSockListener) Start() error {
	log.Info("Starting VSOCK listener service...")

	// Start VM watcher
	if err := v.vmWatcher.Start(); err != nil {
		return errors.Wrap(err, "failed to start VM watcher")
	}

	// Start sensor relay
	if err := v.relay.Start(); err != nil {
		return errors.Wrap(err, "failed to start sensor relay")
	}

	// Start VSOCK server
	if err := v.vsockSrv.Start(); err != nil {
		return errors.Wrap(err, "failed to start VSOCK server")
	}

	log.Infof("VSOCK listener service started on port %d", VSOCK_PORT)
	return nil
}

// Stop stops the VSOCK listener service
func (v *VSockListener) Stop() error {
	log.Info("Stopping VSOCK listener service...")

	v.cancel()
	v.stopper.Client().Stop()

	// Stop all components
	var errs []error

	if err := v.vsockSrv.Stop(); err != nil {
		errs = append(errs, errors.Wrap(err, "failed to stop VSOCK server"))
	}

	if err := v.relay.Stop(); err != nil {
		errs = append(errs, errors.Wrap(err, "failed to stop sensor relay"))
	}

	if err := v.vmWatcher.Stop(); err != nil {
		errs = append(errs, errors.Wrap(err, "failed to stop VM watcher"))
	}

	// Wait for all goroutines to finish
	v.wg.Wait()

	if len(errs) > 0 {
		return errors.Errorf("errors stopping service: %v", errs)
	}

	log.Info("VSOCK listener service stopped")
	return nil
}
