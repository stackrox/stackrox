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
	// Initialize scannerClient only once
	once.Do(func() {
		if !env.LocalImageScanningEnabled.BooleanSetting() {
			log.Info("Local scanning disabled, will not attempt to connect to a local scanner.")
			return
		}

		// Get the Client interface based on the type assertion
		if env.EnableScannerV4.BooleanSetting() {
			// Use a type assertion to get the Client interface for V4GRPCClient
			v4GrpcClient, ok := scannerClient.(*V4GRPCClient)
			if !ok {
				log.Errorf("Invalid type for scannerClient. Expected *V4GRPCClient, got %T", scannerClient)
				utils.Should(errors.New("Invalid type for scannerClient."))
			}
			scannerClient = v4GrpcClient
		} else {
			// Use a type assertion to get the Client interface for GrpcClient
			grpcClient, ok := scannerClient.(*GrpcClient)
			if !ok {
				log.Errorf("Invalid type for scannerClient. Expected *GrpcClient, got %T", scannerClient)
				utils.Should(errors.New("Invalid type for scannerClient."))
			}
			scannerClient = grpcClient
		}

		// Call Dial function on the Client
		var err error
		if env.EnableScannerV4.BooleanSetting() {
			_, err = scannerClient.Dial(env.ScannerV4GRPCEndpoint.Setting())
		} else {
			_, err = scannerClient.Dial(env.ScannerSlimGRPCEndpoint.Setting())
		}
		utils.Should(err)
	})

	// Return grpcClient outside the once.Do function
	return scannerClient

}
