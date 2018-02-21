package volumes

// VolumeRegistry has a mapping of a VolumeSource type to a function that returns a VolumeSource
var VolumeRegistry = map[string]func(i interface{}) VolumeSource{}

// VolumeSource is an interface that wrap for specific Kubernetes Volume types need to define
type VolumeSource interface {
	Source() string
	Type() string
}
