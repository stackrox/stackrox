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
func GRPCClientSingleton(usingScannerV4 bool) Client {
	once.Do(func() {
		if !env.LocalImageScanningEnabled.BooleanSetting() {
			log.Info("Local scanning disabled, will not attempt to connect to a local scanner.")
			return
		}

		if usingScannerV4 {
			v4GrpcClient, ok := scannerClient.(*V4GRPCClient)
			if !ok {
				log.Errorf("Invalid type for scannerClient. Expected Client, got %T", v4GrpcClient)
				utils.Should(errors.New("Invalid type for scannerClient."))
			}
			scannerClient = v4GrpcClient
		} else {
			grpcClient, ok := scannerClient.(*GrpcClient)
			if !ok {
				log.Errorf("Invalid type for scannerClient. Expected Client, got %T", grpcClient)
				utils.Should(errors.New("Invalid type for scannerClient."))
			}
			scannerClient = grpcClient
		}
		// Call Dial function on Client
		endpoint := env.ScannerSlimGRPCEndpoint.Setting()
		if usingScannerV4 {
			endpoint = env.ScannerV4GRPCEndpoint.Setting()
		}
		_, err := scannerClient.Dial(endpoint)
		utils.Should(err)
	})

	// Return grpcClient outside the once.Do function
	return scannerClient
}
