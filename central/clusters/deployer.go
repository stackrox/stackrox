package clusters

import (
	"fmt"
	"strconv"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/defaultimages"
	"github.com/stackrox/rox/pkg/devbuild"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/helm/charts"
	"github.com/stackrox/rox/pkg/images/defaults"
	"github.com/stackrox/rox/pkg/images/utils"
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

func generateCollectorImageNameFromString(collectorImage, tag string) (*storage.ImageName, error) {
	image, _, err := utils.GenerateImageNameFromString(collectorImage)
	if err != nil {
		return nil, err
	}
	utils.SetImageTagNoSha(image, tag)
	return image, nil
}

func generateCollectorImageName(mainImageName *storage.ImageName, collectorImage string) (*storage.ImageName, error) {
	collectorTag := version.GetCollectorVersion()
	var collectorImageName *storage.ImageName
	if collectorImage != "" {
		var err error
		collectorImageName, err = generateCollectorImageNameFromString(collectorImage, collectorTag)
		if err != nil {
			return nil, err
		}
	} else {
		collectorImageName = defaultimages.GenerateNamedImageFromMainImage(mainImageName, collectorTag, defaultimages.Collector)
	}
	return collectorImageName, nil
}

// FieldsFromClusterAndRenderOpts gets the template values for values.yaml
func FieldsFromClusterAndRenderOpts(c *storage.Cluster, opts RenderOptions) (charts.MetaValues, error) {
	mainImage, err := utils.GenerateImageFromStringWithDefaultTag(c.MainImage, version.GetMainVersion())
	if err != nil {
		return nil, err
	}
	mainImageName := mainImage.GetName()

	collectorImageName, err := generateCollectorImageName(mainImageName, c.CollectorImage)
	if err != nil {
		return nil, err
	}

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
	fields := charts.MetaValues{
		"ClusterName": c.Name,
		"ClusterType": c.Type.String(),

		"MainRegistry": urlfmt.FormatURL(mainImageName.GetRegistry(), urlfmt.NONE, urlfmt.NoTrailingSlash),
		"ImageRemote":  mainImageName.GetRemote(),
		"ImageTag":     mainImageName.GetTag(),

		"PublicEndpoint":     urlfmt.FormatURL(c.CentralApiEndpoint, urlfmt.NONE, urlfmt.NoTrailingSlash),
		"AdvertisedEndpoint": urlfmt.FormatURL(env.AdvertisedEndpoint.Setting(), urlfmt.NONE, urlfmt.NoTrailingSlash),

		"CollectorRegistry":        urlfmt.FormatURL(collectorImageName.GetRegistry(), urlfmt.NONE, urlfmt.NoTrailingSlash),
		"CollectorFullImageRemote": collectorImageName.GetRemote(),
		"CollectorSlimImageRemote": collectorImageName.GetRemote(),
		"CollectorFullImageTag":    fmt.Sprintf("%s-latest", collectorImageName.GetTag()),
		"CollectorSlimImageTag":    fmt.Sprintf("%s-slim", collectorImageName.GetTag()),
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
	return fields, nil
}
