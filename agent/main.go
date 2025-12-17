package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"

	"github.com/stackrox/rox/agent/internal/cpe"
	"github.com/stackrox/rox/agent/internal/grpc"
	"github.com/stackrox/rox/agent/internal/report"
	"github.com/stackrox/rox/agent/internal/rpm"
	"github.com/stackrox/rox/agent/internal/vsock"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/mtls"
)

type Config struct {
	TransmissionMode string
	CertPath         string
	SensorURL        string
	// TLS certificate files - args override env vars
	CACertFile string
	ClientCert string
	ClientKey  string
}

func parseArgs() (*Config, error) {
	mode := flag.String("mode", "vsock", "Transmission mode: vsock or grpc")
	certPath := flag.String("cert-path", "", "Certificate path for gRPC mode")
	sensorURL := flag.String("sensor-url", "", "Sensor URL for gRPC mode")
	flag.Parse()

	config := &Config{
		TransmissionMode: *mode,
		CertPath:         *certPath,
		SensorURL:        *sensorURL,
	}

	// Set certificate files from ROX_MTLS_* environment variables
	config.CACertFile = mtls.CAFilePath()   // Uses ROX_MTLS_CA_FILE env var
	config.ClientCert = mtls.CertFilePath() // Uses ROX_MTLS_CERT_FILE env var
	config.ClientKey = mtls.KeyFilePath()   // Uses ROX_MTLS_KEY_FILE env var

	// Validate transmission mode
	if config.TransmissionMode != "vsock" && config.TransmissionMode != "grpc" {
		return nil, fmt.Errorf("invalid transmission mode: %s (must be 'vsock' or 'grpc')", config.TransmissionMode)
	}

	// Validate mode-specific requirements
	if config.TransmissionMode == "grpc" {
		if config.SensorURL == "" {
			return nil, errors.New("grpc mode requires --sensor-url")
		}

		// Check if we have either cert-path (legacy) or ROX_MTLS env vars
		hasLegacyCertPath := config.CertPath != ""
		hasEnvVarCerts := os.Getenv("ROX_MTLS_CA_FILE") != "" &&
			os.Getenv("ROX_MTLS_CERT_FILE") != "" &&
			os.Getenv("ROX_MTLS_KEY_FILE") != ""

		if !hasLegacyCertPath && !hasEnvVarCerts {
			return nil, errors.New("grpc mode requires either --cert-path or ROX_MTLS_* environment variables")
		}
	}

	return config, nil
}

func checkVSockAvailability() error {
	// Check if /dev/vsock exists and is accessible
	if _, err := os.Stat("/dev/vsock"); os.IsNotExist(err) {
		return errors.New("VSOCK device not available: ensure VM has autoattachVSOCK: true configured")
	}
	return nil
}

func main() {
	log := logging.LoggerForModule()

	config, err := parseArgs()
	if err != nil {
		log.Fatalf("Error parsing arguments: %v", err)
	}

	// For VSOCK mode, check availability early
	if config.TransmissionMode == "vsock" {
		if err := checkVSockAvailability(); err != nil {
			log.Fatalf("VSOCK not available: %v", err)
		}
		log.Info("Using VSOCK communication mode")
	} else {
		log.Infof("Using gRPC communication mode (URL: %s)", config.SensorURL)
	}

	ctx := context.Background()

	// Step 1: Collect RPM package information
	log.Info("Collecting package information...")
	packages, err := rpm.CollectPackages()
	if err != nil {
		log.Fatalf("Failed to collect RPM packages: %v", err)
	}
	log.Infof("Collected %d packages", len(packages))

	// Step 2: Parse OS information for CPE generation
	log.Info("Parsing OS information...")
	osInfo, err := cpe.ParseOSRelease()
	if err != nil {
		log.Fatalf("Failed to parse OS information: %v", err)
	}
	log.Infof("Detected OS: %s %s", osInfo.Name, osInfo.Version)
	log.Infof("OS Details - ID: %s, VersionID: %s, CPE: %s, Arch: %s", osInfo.ID, osInfo.VersionID, osInfo.CPEName, osInfo.Arch)

	// Step 3: Build IndexReport with packages and proper CPEs
	log.Info("Building index report...")
	indexReport, err := report.BuildIndexReport(packages, osInfo)
	if err != nil {
		log.Fatalf("Failed to build index report: %v", err)
	}

	// Step 4: Transmit the IndexReport
	log.Info("Transmitting index report...")
	switch config.TransmissionMode {
	case "vsock":
		if err := vsock.SendIndexReport(ctx, indexReport); err != nil {
			log.Fatalf("Failed to send via VSOCK: %v", err)
		}
		log.Info("Successfully sent index report via VSOCK")
	case "grpc":
		grpcConfig := grpc.Config{
			SensorURL:  config.SensorURL,
			CertPath:   config.CertPath,
			CACertFile: config.CACertFile,
			ClientCert: config.ClientCert,
			ClientKey:  config.ClientKey,
		}
		if err := grpc.SendIndexReport(ctx, indexReport, grpcConfig); err != nil {
			log.Fatalf("Failed to send via gRPC: %v", err)
		}
		log.Info("Successfully sent index report via gRPC")
	}
}
