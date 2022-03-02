package util

import (
	"context"
	"fmt"

	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/buildinfo"
	"github.com/stackrox/rox/pkg/images/defaults"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	warningKernelSupportAvailableUnimplemented = `Central does not support API for checking if kernel support is available. Not using a slim collector image.
Please upgrade Central if slim collector images shall be used.`
	warningFailedAToGetKernelSupportAvailable = "Failed to retrieve KernelSupportAvailable property from Central"
	warningLegacyCentralDefaultMain           = "Central is running on a legacy version, main will be defaulted to %s if no override is provided"
	warningNoClusterDefaultsAPI               = "Central does not implement /v1/cluster-defaults API"
	warningFailedToCallClusterDefaultsAPI     = "Failed to retrieve data from /v1/cluster-defaults API"
)

// CentralEnv contains information about Central's runtime environment.
type CentralEnv struct {
	// KernelSupportAvailable will be fetched from ClusterDefaults API. If talking to a legacy central, the data will
	// be fetched from legacy API GetKernelSupportAvailable. If both calls fail, false is returned and the error is
	// stored in Error property.
	KernelSupportAvailable bool
	// MainImage will only be set if central most likely won't accept an empty main image. In this case roxctl
	// will try to derive the default MainImage locally by checking the release flag.
	// If connecting to a newer (>= 3.69) version of central MainImage will be empty.
	MainImage string
	Warnings  []string
	Error     error
}

func (e *CentralEnv) fetchClusterDefaults(ctx context.Context, service v1.ClustersServiceClient) error {
	clusterDefault, err := service.GetClusterDefaults(ctx, &v1.Empty{})
	if status.Convert(err).Code() == codes.Unimplemented {
		e.Warnings = append(e.Warnings, warningNoClusterDefaultsAPI)
	} else {
		e.Warnings = append(e.Warnings, fmt.Sprintf("%s: %s", warningFailedToCallClusterDefaultsAPI, err))
	}

	if err != nil {
		return err
	}
	e.KernelSupportAvailable = clusterDefault.GetKernelSupportAvailable()
	return nil
}

func (e *CentralEnv) getMainImageFromFlavor() string {
	var flavor defaults.ImageFlavor
	if buildinfo.ReleaseBuild {
		flavor = defaults.RHACSReleaseImageFlavor()
	} else {
		flavor = defaults.DevelopmentBuildImageFlavor()
	}
	return flavor.MainImageNoTag()
}

func (e *CentralEnv) fetchClusterDefaultsLegacy(ctx context.Context, service v1.ClustersServiceClient) {
	resp, err := service.GetKernelSupportAvailable(ctx, &v1.Empty{})
	if err != nil {
		if status.Convert(err).Code() == codes.Unimplemented {
			e.Warnings = append(e.Warnings, warningKernelSupportAvailableUnimplemented)
		} else {
			e.Warnings = append(e.Warnings, warningFailedAToGetKernelSupportAvailable)
		}
		// If all APIs fail, we default KernelSupportAvailable to false and store the error
		e.KernelSupportAvailable = false
		e.Error = err
	} else {
		e.KernelSupportAvailable = resp.KernelSupportAvailable
	}

	// MainImage is only set if roxctl is communicating with a legacy version of central that most likely won't
	// be able to accept empty main image in PostCluster request.
	mainImage := e.getMainImageFromFlavor()
	e.Warnings = append(e.Warnings, fmt.Sprintf(warningLegacyCentralDefaultMain, mainImage))
	e.MainImage = mainImage
}

func (e *CentralEnv) populateCentralEnv(ctx context.Context, service v1.ClustersServiceClient) {
	err := e.fetchClusterDefaults(ctx, service)
	if err != nil {
		e.fetchClusterDefaultsLegacy(ctx, service)
	}
}

// RetrieveCentralEnvOrDefault is a convenience function wrapping `PopulateCentralEnv`. It populates a fresh `CentralEnv`
// struct with defaults and warnings. This function fallbacks to legacy APIs if the defaults are not available.
// Ultimately, it will default values locally when it can't determine configuration from central.
// Warnings are to be used by the caller to properly display them.
// If there is an error fetching defaults from the API the error will be returned in `CentralEnv`.
func RetrieveCentralEnvOrDefault(ctx context.Context, service v1.ClustersServiceClient) CentralEnv {
	env := CentralEnv{}
	env.populateCentralEnv(ctx, service)
	return env
}
