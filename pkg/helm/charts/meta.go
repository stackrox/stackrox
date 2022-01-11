package charts

import (
	"reflect"

	"github.com/stackrox/rox/pkg/buildinfo"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/images/defaults"
	"github.com/stackrox/rox/pkg/version"
)

// MetaValuesKey exists exclusively to protect MetaValues from losing typing and becoming exchangeable with
// map[string]interface{}. By doing this we get the opportunity to more reliably trace MetaValues usage throughout the
// codebase.
// TODO(RS-379): Switch MetaValues to be struct and get rid of MetaValuesKey.

// MetaValues are the values to be passed to the StackRox chart templates.
type MetaValues struct {
	Versions                         version.Versions
	MainRegistry                     string
	ImageRemote                      string
	CollectorRegistry                string
	CollectorFullImageRemote         string
	CollectorSlimImageRemote         string
	CollectorFullImageTag            string
	CollectorSlimImageTag            string
	ScannerImageRemote               string
	ScannerSlimImageRemote           string
	ScannerImageTag                  string
	ScannerDBImageRemote             string
	ScannerDBSlimImageRemote         string
	ScannerDBImageTag                string
	RenderMode                       string
	ChartRepo                        defaults.ChartRepo
	ImagePullSecrets                 defaults.ImagePullSecrets
	Operator                         bool
	FeatureFlags                     interface{} // TODO: lvm change ?
	CertsOnly                        bool
	ClusterType                      string
	ClusterName                      string
	KubectlOutput                    bool
	ImageTag                         string
	PublicEndpoint                   string
	AdvertisedEndpoint               string
	CollectionMethod                 string
	TolerationsEnabled               bool
	CreateUpgraderSA                 bool
	EnvVars                          map[string]string
	K8sCommand                       string
	OfflineMode                      bool
	SlimCollector                    bool
	AdmissionController              bool
	AdmissionControlListenOnUpdates  bool
	AdmissionControlListenOnEvents   bool
	DisableBypass                    bool
	TimeoutSeconds                   int32
	ScanInline                       bool
	AdmissionControllerEnabled       bool
	AdmissionControlEnforceOnUpdates bool
}

// GetMetaValuesForFlavor are the default meta values for rendering the StackRox charts in production.
func GetMetaValuesForFlavor(imageFlavor defaults.ImageFlavor) MetaValues {
	metaValues := MetaValues{
		Versions:                 imageFlavor.Versions,
		MainRegistry:             imageFlavor.MainRegistry,
		ImageRemote:              imageFlavor.MainImageName,
		ImageTag:                 imageFlavor.MainImageTag,
		CollectorRegistry:        imageFlavor.CollectorRegistry,
		CollectorFullImageRemote: imageFlavor.CollectorImageName,
		CollectorSlimImageRemote: imageFlavor.CollectorSlimImageName,
		CollectorFullImageTag:    imageFlavor.CollectorImageTag,
		CollectorSlimImageTag:    imageFlavor.CollectorSlimImageTag,
		ScannerImageRemote:       imageFlavor.ScannerImageName,
		ScannerSlimImageRemote:   imageFlavor.ScannerSlimImageName,
		ScannerImageTag:          imageFlavor.ScannerImageTag,
		ScannerDBImageRemote:     imageFlavor.ScannerDBImageName,
		ScannerDBSlimImageRemote: imageFlavor.ScannerDBSlimImageName,
		ScannerDBImageTag:        imageFlavor.ScannerDBImageTag,
		RenderMode:               "",
		ChartRepo:                imageFlavor.ChartRepo,
		ImagePullSecrets:         imageFlavor.ImagePullSecrets,
		Operator:                 false,
	}

	if !buildinfo.ReleaseBuild {
		metaValues.FeatureFlags = getFeatureFlags()
	}
	return metaValues
}

// ToRaw converts MetaValues to map[string]interface{} for use in Go templating.
// Go templating does not like our MetaValuesKey and prefers to have string as a key in the map.
// Unfortunately, an attempt to cast MetaValues to map[string]interface{} does not compile, therefore we need to copy
// the map.
// TODO(RS-379): Switch MetaVals to struct and get rid of ToRaw function.
// TODO: lvm delete this function
func (m MetaValues) ToRaws() map[string]interface{} {
	v := reflect.ValueOf(m)
	result := make(map[string]interface{}, v.NumField())
	for i := 0; i < v.NumField(); i++ {
		if v.Field(i).Interface() == nil {
			continue
		}
		result[v.Type().Field(i).Name] = v.Field(i).Interface()
	}
	return result
}

func getFeatureFlags() map[string]interface{} {
	featureFlagVals := make(map[string]interface{})
	for _, feature := range features.Flags {
		featureFlagVals[feature.EnvVar()] = feature.Enabled()
	}
	return featureFlagVals
}
