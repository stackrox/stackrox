package clusters

import (
	"fmt"
	"strconv"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/defaultimages"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/images/utils"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/version"
)

var (
	log = logging.LoggerForModule()
)

// RenderOptions are options that control the rendering.
type RenderOptions struct {
	CreateUpgraderSA bool
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
	collectorTag := fmt.Sprintf("%s-latest", version.GetCollectorVersion())
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
func FieldsFromClusterAndRenderOpts(c *storage.Cluster, opts RenderOptions) (map[string]interface{}, error) {
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
	for _, feature := range features.Flags {
		envVars[feature.EnvVar()] = strconv.FormatBool(feature.Enabled())
	}

	fields := map[string]interface{}{
		"ClusterName": c.Name,
		"ClusterType": c.Type.String(),

		"ImageRegistry": mainImageName.GetRegistry(),
		"ImageRemote":   mainImageName.GetRemote(),
		"ImageTag":      mainImageName.GetTag(),

		"PublicEndpoint":     c.CentralApiEndpoint,
		"AdvertisedEndpoint": env.AdvertisedEndpoint.Setting(),

		"CollectorRegistry":    collectorImageName.GetRegistry(),
		"CollectorImageRemote": collectorImageName.GetRemote(),
		"CollectorImageTag":    collectorImageName.GetTag(),
		"CollectionMethod":     c.CollectionMethod.String(),

		"TolerationsEnabled": !c.GetTolerationsConfig().GetDisabled(),
		"CreateUpgraderSA":   opts.CreateUpgraderSA,

		"EnvVars":             envVars,
		"AdmissionController": false,
	}

	if features.AdmissionControlService.Enabled() && c.AdmissionController {
		fields["AdmissionController"] = true
		fields["AdmissionControlListenOnUpdates"] = features.AdmissionControlEnforceOnUpdate.Enabled() &&
			c.GetAdmissionControllerUpdates()
		fields["DisableBypass"] = c.GetDynamicConfig().GetAdmissionControllerConfig().GetDisableBypass()
		fields["TimeoutSeconds"] = c.GetDynamicConfig().GetAdmissionControllerConfig().GetTimeoutSeconds()
		fields["ScanInline"] = c.GetDynamicConfig().GetAdmissionControllerConfig().GetScanInline()
		fields["AdmissionControllerEnabled"] = c.GetDynamicConfig().GetAdmissionControllerConfig().GetEnabled()
		fields["AdmissionControlEnforceOnUpdates"] = c.GetDynamicConfig().GetAdmissionControllerConfig().GetEnforceOnUpdates()
	}

	return fields, nil
}
