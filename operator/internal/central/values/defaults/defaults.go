package defaults

import (
	"encoding/json"
	"fmt"
	"reflect"

	"github.com/go-logr/logr"
	platform "github.com/stackrox/rox/operator/api/v1alpha1"
	"github.com/stackrox/rox/operator/internal/common/defaulting"
)

var staticDefaults = platform.CentralSpec{
	Central:   nil,
	Scanner:   nil,
	ScannerV4: nil,
	Egress: &platform.Egress{
		ConnectivityPolicy: platform.ConnectivityOnline.Pointer(),
	},
	TLS:              nil,
	ImagePullSecrets: nil,
	Customize:        nil,
	Misc:             nil,
	Overlays:         nil,
	Monitoring:       nil,
	Network:          nil,
	ConfigAsCode:     nil,
}

var CentralStaticDefaults = defaulting.CentralDefaultingFlow{
	Name: "static-defaults",
	DefaultingFunc: func(_ logr.Logger, _ *platform.CentralStatus, _ map[string]string, _ *platform.CentralSpec, defaults *platform.CentralSpec) error {
		if !reflect.DeepEqual(defaults, &platform.CentralSpec{}) {
			return fmt.Errorf("supplied central's .Default is not empty: %s", marshal(defaults))
		}
		staticDefaults.DeepCopyInto(defaults)
		return nil
	},
}

func marshal(defaults *platform.CentralSpec) string {
	marshalled, err := json.Marshal(defaults)
	if err == nil {
		return string(marshalled)
	}
	// Should never happen, but returning SOMETHING is better than panicking.
	return fmt.Sprintf("%+v", defaults) // Not as pretty for embedded pointers.
}
