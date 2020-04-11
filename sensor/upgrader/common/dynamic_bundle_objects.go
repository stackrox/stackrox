package common

type (
	// DynamicBundleObjectKind is the kind of a dynamic bundle object (ConfigMap or an opaque Secret)
	DynamicBundleObjectKind int
)

const (
	// OpaqueSecret is a v1/Secret object with type `Opaque`
	OpaqueSecret DynamicBundleObjectKind = iota
	// ConfigMap is a v1/ConfigMap
	ConfigMap
)

// DynamicBundleObjectDesc describes a dynamic bundle object, i.e., an object that is not represented by a YAML file,
// but instead must be constructed from other files in the bundle.
type DynamicBundleObjectDesc struct {
	Kind     DynamicBundleObjectKind
	Name     string
	Optional bool

	Files []string
}

var (
	// DynamicBundleObjects is the list of dynamic bundle objects.
	// IMPORTANT: This must be kept in sync with the commands ran as part of the sensor.sh script.
	DynamicBundleObjects = []DynamicBundleObjectDesc{
		{
			Kind:     OpaqueSecret,
			Name:     "monitoring-client",
			Optional: true,
			Files:    []string{"monitoring-client-cert.pem", "monitoring-client-key.pem", "monitoring-ca.pem"},
		},
		{
			Kind:     ConfigMap,
			Name:     "telegraf",
			Optional: true,
			Files:    []string{"telegraf.conf"},
		},
	}
)
