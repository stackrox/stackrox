package clusters

import (
	"strconv"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/devbuild"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/helm/charts"
	"github.com/stackrox/rox/pkg/images/defaults"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/urlfmt"
	"github.com/stackrox/rox/pkg/version"
)

var (
	log = logging.LoggerForModule()
)

// RenderOptions are options that control the rendering.
type RenderOptions struct {
	CreateUpgraderSA bool
	SlimCollector    bool
	IstioVersion     string
}

func getBaseMetaValues(c *storage.Cluster, opts *RenderOptions) charts.MetaValues {
	envVars := make(map[string]string)
	if devbuild.IsEnabled() {
		for _, feature := range features.Flags {
			envVars[feature.EnvVar()] = strconv.FormatBool(feature.Enabled())
		}
	}

	command := "kubectl"
	if c.Type == storage.ClusterType_OPENSHIFT_CLUSTER || c.Type == storage.ClusterType_OPENSHIFT4_CLUSTER {
		command = "oc"
	}

	return charts.MetaValues{
		"ClusterName": c.Name,
		"ClusterType": c.Type.String(),

		"PublicEndpoint":     urlfmt.FormatURL(c.CentralApiEndpoint, urlfmt.NONE, urlfmt.NoTrailingSlash),
		"AdvertisedEndpoint": urlfmt.FormatURL(env.AdvertisedEndpoint.Setting(), urlfmt.NONE, urlfmt.NoTrailingSlash),

		"CollectionMethod":         c.CollectionMethod.String(),

		// Hardcoding RHACS charts repo for now.
		// TODO: fill ChartRepo based on the current image flavor.
		"ChartRepo": defaults.ChartRepo{
			URL: "http://mirror.openshift.com/pub/rhacs/charts",
		},

		"TolerationsEnabled": !c.GetTolerationsConfig().GetDisabled(),
		"CreateUpgraderSA":   opts.CreateUpgraderSA,

		"EnvVars": envVars,

		"K8sCommand": command,

		"OfflineMode": env.OfflineModeEnv.BooleanSetting(),

		"SlimCollector": opts.SlimCollector,

		"KubectlOutput": true,

		"Versions": version.GetAllVersions(),

		"FeatureFlags": make(map[string]string),

		"AdmissionController":              c.AdmissionController,
		"AdmissionControlListenOnUpdates":  c.GetAdmissionControllerUpdates(),
		"AdmissionControlListenOnEvents":   c.GetAdmissionControllerEvents(),
		"DisableBypass":                    c.GetDynamicConfig().GetAdmissionControllerConfig().GetDisableBypass(),
		"TimeoutSeconds":                   c.GetDynamicConfig().GetAdmissionControllerConfig().GetTimeoutSeconds(),
		"ScanInline":                       c.GetDynamicConfig().GetAdmissionControllerConfig().GetScanInline(),
		"AdmissionControllerEnabled":       c.GetDynamicConfig().GetAdmissionControllerConfig().GetEnabled(),
		"AdmissionControlEnforceOnUpdates": c.GetDynamicConfig().GetAdmissionControllerConfig().GetEnforceOnUpdates(),
	}
}

// FieldsFromClusterAndRenderOpts gets the template values for values.yaml
func FieldsFromClusterAndRenderOpts(c *storage.Cluster, flavor *defaults.ImageFlavor, opts RenderOptions) (charts.MetaValues, error) {
	baseValues := getBaseMetaValues(c, &opts)
	overrides, err := NewImageOverrides(flavor, c)
	if err != nil {
		return nil, err
	}

	overrides.SetMainOverride(baseValues)
	overrides.SetCollectorFullOverride(baseValues)
	overrides.SetCollectorSlimOverride(baseValues)

	return baseValues, nil
}
