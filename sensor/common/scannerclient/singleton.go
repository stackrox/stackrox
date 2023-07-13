package scannerclient

import (
	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/pkg/utils"
)

var (
	once          sync.Once
	scannerClient Client
)

// GRPCClientSingleton returns a gRPC Client to a local Scanner.
// Only one Client per Sensor is required.
func GRPCClientSingleton() Client {
	once.Do(func() {
		if !env.LocalImageScanningEnabled.BooleanSetting() {
			log.Info("Local scanning disabled, will not attempt to connect to a local scanner.")
			return
		}

		// Change this line to use a type assertion to get the Client interface.
		grpcClient, ok := scannerClient.(*GrpcClient)
		if !ok {
			log.Errorf("Invalid type for scannerClient. Expected *GrpcClient, got %T", scannerClient)
			utils.Should(errors.New("Invalid type for scannerClient."))
		}
		scannerClient = grpcClient

		// Call Dial function on Client
		_, err := scannerClient.Dial(env.ScannerSlimGRPCEndpoint.Setting())
		utils.Should(err)
	})

	// Return grpcClient outside the once.Do function
	return scannerClient
}
