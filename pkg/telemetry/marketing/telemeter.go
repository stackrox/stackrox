package marketing

import (
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/logging"
)

type Telemeter interface {
	Start()
	Stop()
	Identify(props map[string]any)
	Track(userAgent, event string)
	TrackProp(userAgent, event string, key string, value any)
	TrackProps(userAgent, event string, props map[string]any)
}

var (
	log = logging.LoggerForModule()
)

func Enabled() bool {
	return env.AmplitudeApiKey.Setting() != ""
}

// Device represents the central instance properties.
type Device struct {
	ID       string
	Version  string
	ApiPaths []string
}

// GetDeviceProperties collects the central instance properties.
func GetDeviceProperties() *Device {
	d, err := getK8SData()
	if err != nil {
		log.Errorf("Failed to get device data: %v", err)
		return nil
	}
	return d
}
