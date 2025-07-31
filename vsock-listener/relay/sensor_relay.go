package relay

import (
	"context"
	"sync"
	"time"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/internalapi/sensor"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/clientconn"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/mtls"
	"github.com/stackrox/rox/pkg/safe"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
)

var (
	log = logging.LoggerForModule()
)

// VMData represents data received from a VM
type VMData struct {
	VMUID       string
	VMName      string
	VMNamespace string
	Data        []byte
	Timestamp   time.Time
}

// SensorRelay manages communication with the sensor
type SensorRelay struct {
	ctx        context.Context
	cancel     context.CancelFunc
	sensorAddr string
	conn       *grpc.ClientConn
	client     sensor.VirtualMachineServiceClient
	stopper    concurrency.Stopper
	dataChan   chan *VMData
	wg         sync.WaitGroup
}

// NewSensorRelay creates a new sensor relay
func NewSensorRelay(ctx context.Context, sensorAddr string) (*SensorRelay, error) {
	ctx, cancel := context.WithCancel(ctx)

	return &SensorRelay{
		ctx:        ctx,
		cancel:     cancel,
		sensorAddr: sensorAddr,
		stopper:    concurrency.NewStopper(),
		dataChan:   make(chan *VMData, 100), // Buffer for 100 messages
	}, nil
}

// Start starts the sensor relay
func (r *SensorRelay) Start() error {
	log.Infof("Starting sensor relay to %s", r.sensorAddr)

	// Create gRPC connection to sensor
	conn, err := r.createSensorConnection()
	if err != nil {
		return errors.Wrap(err, "failed to create sensor connection")
	}
	r.conn = conn

	// Create VirtualMachine service client
	r.client = sensor.NewVirtualMachineServiceClient(conn)

	// Start processing messages
	r.wg.Add(1)
	go r.processMessages()

	log.Info("Sensor relay started")
	return nil
}

// Stop stops the sensor relay
func (r *SensorRelay) Stop() error {
	log.Info("Stopping sensor relay...")

	r.cancel()
	r.stopper.Client().Stop()

	// Close connection
	if r.conn != nil {
		if err := r.conn.Close(); err != nil {
			log.Warnf("Error closing sensor connection: %v", err)
		}
	}

	// Close data channel
	close(r.dataChan)

	// Wait for processing to finish
	r.wg.Wait()

	log.Info("Sensor relay stopped")
	return nil
}

// SendVMData sends VM data to the sensor
func (r *SensorRelay) SendVMData(data *VMData) error {
	select {
	case r.dataChan <- data:
		return nil
	case <-r.ctx.Done():
		return r.ctx.Err()
	default:
		return errors.New("data channel full, dropping message")
	}
}

// createSensorConnection creates a gRPC connection to the sensor
func (r *SensorRelay) createSensorConnection() (*grpc.ClientConn, error) {
	clientconn.SetUserAgent("Rox VSOCK Listener")

	if err := safe.RunE(func() error {
		if err := configureCA(); err != nil {
			return err
		}
		if err := configureCerts("stackrox"); err != nil {
			return err
		}
		return nil
	}); err != nil {
		log.Errorf("Failed to configure certificates: %v. Connection to sensor might fail.", err)
	}

	conn, err := clientconn.AuthenticatedGRPCConnection(r.ctx, r.sensorAddr, mtls.SensorSubject)

	if err != nil {
		return nil, errors.Wrapf(err, "creating gRPC connection to sensor at %s", r.sensorAddr)
	}

	return conn, nil
}

// processMessages processes VM data messages and sends them to sensor
func (r *SensorRelay) processMessages() {
	defer r.wg.Done()

	for {
		select {
		case <-r.ctx.Done():
			return
		case data, ok := <-r.dataChan:
			if !ok {
				return // Channel closed
			}

			if err := r.sendToSensor(data); err != nil {
				log.Errorf("Failed to send VM data to sensor: %v", err)
				// In a production system, you might want to retry or queue failed messages
			}
		}
	}
}

// sendToSensor sends VM data to the sensor
func (r *SensorRelay) sendToSensor(data *VMData) error {
	// Convert VMData to protobuf message
	vmMessage := r.convertToVMMessage(data)

	// Send to sensor using existing VM service
	ctx, cancel := context.WithTimeout(r.ctx, 30*time.Second)
	defer cancel()

	// Use the existing VirtualMachine service to send data
	// This assumes the sensor has been extended to handle VM data messages
	_, err := r.client.UpsertVirtualMachine(ctx, &sensor.UpsertVirtualMachineRequest{
		VirtualMachine: vmMessage,
	})

	if err != nil {
		return errors.Wrap(err, "failed to send VM message to sensor")
	}

	log.Debugf("Sent VM data for %s/%s to sensor", data.VMNamespace, data.VMName)
	return nil
}

// convertToVMMessage converts VMData to a storage.VirtualMachine protobuf message
func (r *SensorRelay) convertToVMMessage(data *VMData) *storage.VirtualMachine {
	// Unmarshal the protobuf data directly
	var vm storage.VirtualMachine
	if err := proto.Unmarshal(data.Data, &vm); err != nil {
		log.Errorf("Failed to unmarshal VM protobuf data from %s: %v", data.VMName, err)
		// Return a minimal VM message on error
		return &storage.VirtualMachine{
			Id:          data.VMUID,
			Name:        data.VMName,
			Namespace:   data.VMNamespace,
			LastUpdated: timestamppb.New(data.Timestamp),
		}
	}

	// Update metadata from VSOCK connection
	vm.Id = data.VMUID
	vm.Name = data.VMName
	vm.Namespace = data.VMNamespace
	vm.LastUpdated = timestamppb.New(data.Timestamp)

	if vm.Scan != nil {
		log.Debugf("Processed %d components for VM %s", len(vm.Scan.Components), data.VMName)
	}

	return &vm
}
