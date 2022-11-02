package marketing

import (
	mpkg "github.com/stackrox/rox/pkg/telemetry/marketing"
	"github.com/stackrox/rox/pkg/telemetry/marketing/amplitude"
	"google.golang.org/grpc"
)

func Init() grpc.UnaryServerInterceptor {
	if mpkg.Enabled() {
		device := mpkg.GetDeviceProperties()
		telemeter := amplitude.Init(device)
		NewGatherer(telemeter)
		return interceptor(device, telemeter)
	}
	return nil
}
