package grpc

import (
	"context"
	"crypto/x509"
	"errors"
	"fmt"
	"os"

	"github.com/stackrox/rox/generated/internalapi/sensor"
	v1 "github.com/stackrox/rox/generated/internalapi/virtualmachine/v1"
	"github.com/stackrox/rox/pkg/logging"
)

var (
	log = logging.LoggerForModule()
)

// SendIndexReport sends an IndexReport via gRPC to the sensor
func SendIndexReport(ctx context.Context, indexReport *v1.IndexReport, config Config) error {
	log.Debugf("Creating gRPC client for %s", config.SensorURL)
	client, conn, err := createClient(config)
	if err != nil {
		return fmt.Errorf("failed to create gRPC client: %w", err)
	}
	defer func() {
		if err := conn.Close(); err != nil {
			log.Warnf("Failed to close gRPC connection: %v", err)
		}
	}()

	// Create request
	request := &sensor.UpsertVirtualMachineIndexReportRequest{
		IndexReport: indexReport,
	}

	log.Debug("Sending IndexReport via gRPC...")
	response, err := client.UpsertVirtualMachineIndexReport(ctx, request)
	if err != nil {
		return fmt.Errorf("failed to send IndexReport via gRPC: %w", err)
	}

	// Check response
	if !response.Success {
		return errors.New("sensor reported failure processing IndexReport")
	}

	log.Debug("Successfully sent IndexReport via gRPC")
	return nil
}

// loadCACertificate loads the CA certificate from file
func loadCACertificate(caCertPath string) (*x509.CertPool, error) {
	caCert, err := os.ReadFile(caCertPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read CA certificate from %s: %w", caCertPath, err)
	}

	caCertPool := x509.NewCertPool()
	if !caCertPool.AppendCertsFromPEM(caCert) {
		return nil, errors.New("failed to parse CA certificate")
	}

	return caCertPool, nil
}
