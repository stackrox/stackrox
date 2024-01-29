package charts

import (
	"github.com/stackrox/rox/pkg/buildinfo"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/images/defaults"
	"github.com/stackrox/rox/pkg/version"
)

// MetaValues are the values to be passed to the StackRox chart templates.
type MetaValues struct {
	Versions                         version.Versions
	MainRegistry                     string
	ImageRemote                      string
	CentralDBImageTag                string
	CentralDBImageRemote             string
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
	ScannerV4ImageRemote             string
	ScannerV4DBImageRemote           string
	ScannerV4ImageTag                string
	RenderMode                       string
	ChartRepo                        defaults.ChartRepo
	ImagePullSecrets                 defaults.ImagePullSecrets
	Operator                         bool
	FeatureFlags                     map[string]interface{}
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
	K8sConfig                        map[string]interface{} // renderer.K8sConfig // introduces a cycle in the dependencies
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
	ReleaseBuild                     bool
	TelemetryEnabled                 bool
	TelemetryKey                     string
	TelemetryEndpoint                string

	AutoSensePodSecurityPolicies bool
	EnablePodSecurityPolicies    bool // Only used in the Helm chart if AutoSensePodSecurityPolicies is false.
}

// GetMetaValuesForFlavor are the default meta values for rendering the StackRox charts in production.
func GetMetaValuesForFlavor(imageFlavor defaults.ImageFlavor) *MetaValues {
	metaValues := MetaValues{
		Versions:                 imageFlavor.Versions,
		MainRegistry:             imageFlavor.MainRegistry,
		ImageRemote:              imageFlavor.MainImageName,
		ImageTag:                 imageFlavor.MainImageTag,
		CentralDBImageTag:        imageFlavor.CentralDBImageTag,
		CentralDBImageRemote:     imageFlavor.CentralDBImageName,
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
		ScannerV4ImageRemote:     imageFlavor.ScannerV4ImageName,
		ScannerV4DBImageRemote:   imageFlavor.ScannerV4DBImageName,
		ScannerV4ImageTag:        imageFlavor.ScannerV4ImageTag,
		RenderMode:               "renderAll",
		ChartRepo:                imageFlavor.ChartRepo,
		ImagePullSecrets:         imageFlavor.ImagePullSecrets,
		Operator:                 false,
		ReleaseBuild:             buildinfo.ReleaseBuild,
		FeatureFlags:             getFeatureFlags(),
		TelemetryEnabled:         true,

		AutoSensePodSecurityPolicies: true,
	}

	return &metaValues
}

func getFeatureFlags() map[string]interface{} {
	featureFlagVals := make(map[string]interface{})
	for _, feature := range features.Flags {
		featureFlagVals[feature.EnvVar()] = feature.Enabled()
	}
	return featureFlagVals
}
