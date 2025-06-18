package agent

import (
	"context"
	"github.com/stackrox/rox/generated/internalapi/sensor"
	"github.com/stackrox/rox/pkg/clientconn"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/mtls"
	"google.golang.org/grpc/connectivity"
)

var (
	log = logging.LoggerForModule()
)

func main() {
	clientconn.SetUserAgent(clientconn.AdmissionController)
	ctx := context.Background()

	sensorConn, err := clientconn.AuthenticatedGRPCConnection(
		ctx,
		"sensor.default.svc:443",
		mtls.SensorSubject,
	)

	if err != nil {
		log.Errorf("Could not establish a gRPC connection to Sensor: %v.")
	}

	sensorConn.WaitForStateChange(ctx, connectivity.Ready)

	imageClient := sensor.NewImageServiceClient(sensorConn)
	imageClient.GetImage(ctx, &sensor.GetImageRequest{})
}
